package crawler

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.CrawlerPluginInterface = (*AcademicFrontierPlugin)(nil)

// AcademicFrontierPlugin 学术前沿抓取（arXiv API / HuggingFace Daily Papers / PapersWithCode）
type AcademicFrontierPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewAcademicFrontierPlugin(logger log.Logger, getConstraints func() []string) *AcademicFrontierPlugin {
	return &AcademicFrontierPlugin{
		client:         newCrawlerHTTPClient(20 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *AcademicFrontierPlugin) Name() string { return "academic_frontier" }
func (p *AcademicFrontierPlugin) Label() string {
	return "学术前沿抓取（arXiv API / HuggingFace Papers / PapersWithCode）"
}

func (p *AcademicFrontierPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	if limit <= 0 {
		limit = 10
	}

	perSource := limit / 3
	if perSource < 3 {
		perSource = 3
	}

	seen := make(map[string]struct{})
	allTopics := make([]*biz.RawTopic, 0, limit)

	addTopics := func(topics []*biz.RawTopic) {
		for _, t := range topics {
			if len(allTopics) >= limit {
				break
			}
			key := strings.ToLower(strings.TrimSpace(t.Title))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			allTopics = append(allTopics, t)
		}
	}

	// 来源 1: arXiv API — 最新 AI/ML 论文
	addTopics(p.fetchArXiv(ctx, perSource))

	// 来源 2: HuggingFace Daily Papers — 社区精选热门论文
	addTopics(p.fetchHFDailyPapers(ctx, perSource))

	// 来源 3: PapersWithCode — 热门论文（含代码实现）
	if len(allTopics) < limit {
		remaining := limit - len(allTopics)
		addTopics(p.fetchPapersWithCode(ctx, remaining))
	}

	p.log.Infof("[AcademicFrontier] Fetched %d frontier topics", len(allTopics))
	return allTopics, nil
}

// ── 来源 1: arXiv API（官方免费 Atom API）──

type arxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID      string         `xml:"id"`
	Title   string         `xml:"title"`
	Summary string         `xml:"summary"`
	Links   []arxivLink    `xml:"link"`
	Authors []arxivAuthor  `xml:"author"`
}

type arxivLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type arxivAuthor struct {
	Name string `xml:"name"`
}

func (p *AcademicFrontierPlugin) fetchArXiv(ctx context.Context, limit int) []*biz.RawTopic {
	// arXiv API: 最新 cs.AI + cs.LG + cs.CL 论文
	categories := "cat:cs.AI+OR+cat:cs.LG+OR+cat:cs.CL"
	apiURL := fmt.Sprintf(
		"https://export.arxiv.org/api/query?search_query=%s&sortBy=submittedDate&sortOrder=descending&max_results=%d",
		categories, limit*2, // 多取一些以备过滤
	)

	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		MaxRetries:     2,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[AcademicFrontier] arXiv API error: %v", err)
		return nil
	}

	var feed arxivFeed
	if err := xml.Unmarshal([]byte(body), &feed); err != nil {
		p.log.Warnf("[AcademicFrontier] arXiv XML parse error: %v", err)
		return nil
	}

	var topics []*biz.RawTopic
	for _, entry := range feed.Entries {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(entry.Title)
		title = strings.Join(strings.Fields(title), " ") // 合并多行空白
		if title == "" {
			continue
		}

		paperURL := strings.TrimSpace(entry.ID)
		for _, link := range entry.Links {
			if link.Rel == "alternate" && link.Href != "" {
				paperURL = link.Href
				break
			}
		}

		summary := strings.TrimSpace(entry.Summary)
		summary = strings.Join(strings.Fields(summary), " ")

		var authors []string
		for _, a := range entry.Authors {
			if len(authors) >= 3 {
				break
			}
			authors = append(authors, strings.TrimSpace(a.Name))
		}
		authorStr := strings.Join(authors, ", ")
		if len(entry.Authors) > 3 {
			authorStr += " et al."
		}

		snippet := fmt.Sprintf("[arXiv] %s — %s", authorStr, truncateStr(summary, 180))

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        paperURL,
			Source:     "arxiv_api",
			Popularity: 200, // arXiv 无点赞，给默认值
			Snippet:    snippet,
		})
	}

	p.log.Infof("[AcademicFrontier] arXiv API: %d papers", len(topics))
	return topics
}

// ── 来源 2: HuggingFace Daily Papers（JSON API）──

type hfDailyPaper struct {
	Paper struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Summary   string `json:"summary"`
	} `json:"paper"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	NumComments int    `json:"numComments"`
}

func (p *AcademicFrontierPlugin) fetchHFDailyPapers(ctx context.Context, limit int) []*biz.RawTopic {
	apiURL := "https://huggingface.co/api/daily_papers"
	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[AcademicFrontier] HF daily papers error: %v", err)
		return nil
	}

	var papers []hfDailyPaper
	if err := json.Unmarshal([]byte(body), &papers); err != nil {
		p.log.Warnf("[AcademicFrontier] HF daily papers parse error: %v", err)
		return nil
	}

	var topics []*biz.RawTopic
	for i, p2 := range papers {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(p2.Paper.Title)
		if title == "" {
			title = strings.TrimSpace(p2.Title)
		}
		if title == "" {
			continue
		}

		paperID := p2.Paper.ID
		paperURL := fmt.Sprintf("https://huggingface.co/papers/%s", paperID)

		summary := strings.TrimSpace(p2.Paper.Summary)
		if summary == "" {
			summary = strings.TrimSpace(p2.Summary)
		}

		// 排名越靠前 popularity 越高
		pop := max(50, 500-i*10)

		snippet := fmt.Sprintf("[HF Papers] %s", truncateStr(summary, 220))

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        paperURL,
			Source:     "hf_daily_papers",
			Popularity: pop,
			Snippet:    snippet,
		})
	}

	p.log.Infof("[AcademicFrontier] HF daily papers: %d papers", len(topics))
	return topics
}

// ── 来源 3: PapersWithCode 热门论文（HTML 解析）──

var pwcPaperRe = regexp.MustCompile(`<h3[^>]*>\s*<a\s+href="/papers/([^"]+)"[^>]*>(.*?)</a>`)

func (p *AcademicFrontierPlugin) fetchPapersWithCode(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://paperswithcode.com/greatest", crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://paperswithcode.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[AcademicFrontier] PapersWithCode error: %v", err)
		return nil
	}

	matches := pwcPaperRe.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{})
	var topics []*biz.RawTopic

	for i, m := range matches {
		if len(topics) >= limit {
			break
		}
		if len(m) < 3 {
			continue
		}
		paperSlug := strings.TrimSpace(m[1])
		title := strings.TrimSpace(stripHTMLTags(m[2]))
		if title == "" || paperSlug == "" {
			continue
		}
		// 去重（同一篇论文可能出现多次）
		if _, ok := seen[paperSlug]; ok {
			continue
		}
		seen[paperSlug] = struct{}{}

		paperURL := fmt.Sprintf("https://paperswithcode.com/papers/%s", paperSlug)
		pop := max(50, 400-i*8)

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        paperURL,
			Source:     "paperswithcode",
			Popularity: pop,
			Snippet:    fmt.Sprintf("[PapersWithCode] %s", truncateStr(title, 220)),
		})
	}

	p.log.Infof("[AcademicFrontier] PapersWithCode: %d papers", len(topics))
	return topics
}
