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

var _ biz.CrawlerPluginInterface = (*FundingSignalPlugin)(nil)

// FundingSignalPlugin 融资信号抓取（36氪 RSS + 快讯 / TechCrunch RSS / Crunchbase News RSS）
type FundingSignalPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewFundingSignalPlugin(logger log.Logger, getConstraints func() []string) *FundingSignalPlugin {
	return &FundingSignalPlugin{
		client:         newCrawlerHTTPClient(20 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *FundingSignalPlugin) Name() string { return "funding_signal" }
func (p *FundingSignalPlugin) Label() string {
	return "融资信号抓取（36氪 / TechCrunch / Crunchbase News）"
}

func (p *FundingSignalPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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

	// 来源 1: 36氪 RSS + 快讯 API（中文融资/创投信息）
	addTopics(p.fetch36kr(ctx, perSource))

	// 来源 2: TechCrunch RSS（英文科技融资新闻）
	addTopics(p.fetchTechCrunchRSS(ctx, perSource))

	// 来源 3: Crunchbase News RSS（专业融资报道）
	if len(allTopics) < limit {
		remaining := limit - len(allTopics)
		addTopics(p.fetchCrunchbaseNewsRSS(ctx, remaining))
	}

	p.log.Infof("[FundingSignal] Fetched %d funding topics", len(allTopics))
	return allTopics, nil
}

// ── 来源 1: 36氪 RSS + 快讯 API ──

// rssChannel 通用 RSS 2.0 解析结构
type rssChannel struct {
	XMLName xml.Name `xml:"rss"`
	Items   []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// parseRSSToTopics 通用 RSS 解析
func parseRSSToTopics(body string, source string, snippetPrefix string, limit int) []*biz.RawTopic {
	var rss rssChannel
	if err := xml.Unmarshal([]byte(body), &rss); err != nil {
		return nil
	}

	var topics []*biz.RawTopic
	for i, item := range rss.Items {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		link := strings.TrimSpace(item.Link)
		if link == "" {
			continue
		}

		desc := strings.TrimSpace(stripHTMLTags(item.Description))
		pop := max(50, 300-i*8)

		snippet := fmt.Sprintf("[%s] %s", snippetPrefix, truncateStr(desc, 220))
		if desc == "" {
			snippet = fmt.Sprintf("[%s] %s", snippetPrefix, truncateStr(title, 220))
		}

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        link,
			Source:     source,
			Popularity: pop,
			Snippet:    snippet,
		})
	}
	return topics
}

func (p *FundingSignalPlugin) fetch36kr(ctx context.Context, limit int) []*biz.RawTopic {
	// 优先 RSS
	topics := p.fetch36krRSS(ctx, limit)
	if len(topics) >= limit {
		return topics
	}

	// 补充：快讯 API
	remaining := limit - len(topics)
	newsTopics := p.fetch36krNewsflash(ctx, remaining)
	topics = append(topics, newsTopics...)

	p.log.Infof("[FundingSignal] 36kr: %d topics (RSS + newsflash)", len(topics))
	return topics
}

func (p *FundingSignalPlugin) fetch36krRSS(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://36kr.com/feed", crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://36kr.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[FundingSignal] 36kr RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "36kr_rss", "36氪", limit)
	p.log.Infof("[FundingSignal] 36kr RSS: %d items", len(topics))
	return topics
}

type kr36NewsflashResp struct {
	Code int `json:"code"`
	Data struct {
		Items []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			NewsURL     string `json:"news_url"`
			PublishedAt string `json:"published_at"`
		} `json:"items"`
	} `json:"data"`
}

func (p *FundingSignalPlugin) fetch36krNewsflash(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://36kr.com/api/newsflash", crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://36kr.com/newsflashes",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[FundingSignal] 36kr newsflash error: %v", err)
		return nil
	}

	var resp kr36NewsflashResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		p.log.Warnf("[FundingSignal] 36kr newsflash parse error: %v", err)
		return nil
	}

	var topics []*biz.RawTopic
	for i, item := range resp.Data.Items {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}

		link := item.NewsURL
		if link == "" {
			link = fmt.Sprintf("https://36kr.com/newsflashes/%d", item.ID)
		}

		desc := strings.TrimSpace(item.Description)
		pop := max(50, 250-i*8)

		snippet := fmt.Sprintf("[36氪快讯] %s", truncateStr(desc, 220))
		if desc == "" {
			snippet = fmt.Sprintf("[36氪快讯] %s", truncateStr(title, 220))
		}

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        link,
			Source:     "36kr_newsflash",
			Popularity: pop,
			Snippet:    snippet,
		})
	}

	p.log.Infof("[FundingSignal] 36kr newsflash: %d items", len(topics))
	return topics
}

// ── 来源 2: TechCrunch RSS ──

// tcCDATALinkRe 提取 CDATA 中的链接（36kr 用 CDATA 包裹 link）
var tcCDATALinkRe = regexp.MustCompile(`<!\[CDATA\[(.*?)\]\]>`)

func (p *FundingSignalPlugin) fetchTechCrunchRSS(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://techcrunch.com/feed/", crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://techcrunch.com/",
		MaxRetries:     2,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[FundingSignal] TechCrunch RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "techcrunch_rss", "TechCrunch", limit)
	p.log.Infof("[FundingSignal] TechCrunch RSS: %d items", len(topics))
	return topics
}

// ── 来源 3: Crunchbase News RSS ──

func (p *FundingSignalPlugin) fetchCrunchbaseNewsRSS(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://news.crunchbase.com/feed/", crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://news.crunchbase.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[FundingSignal] Crunchbase News RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "crunchbase_news", "Crunchbase", limit)
	p.log.Infof("[FundingSignal] Crunchbase News RSS: %d items", len(topics))
	return topics
}
