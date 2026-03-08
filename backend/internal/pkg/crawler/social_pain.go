package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.CrawlerPluginInterface = (*SocialPainPlugin)(nil)

// ── 痛点/吐槽关键词（用于从通用内容中筛选痛点话题）──

var painKeywords = []string{
	"痛点", "吐槽", "差评", "难用", "坑", "不好用", "退款", "失望",
	"bug", "问题", "槽点", "不满", "投诉", "体验差", "垃圾", "骗",
	"frustrat", "pain", "problem", "issue", "hate", "terrible",
	"broken", "waste", "disappoint", "annoying", "useless",
}

func isPainRelated(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range painKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// SocialPainPlugin 社交平台痛点抓取（HN Algolia + 百度热搜痛点 + Bing RSS降级）
type SocialPainPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewSocialPainPlugin(logger log.Logger, getConstraints func() []string) *SocialPainPlugin {
	return &SocialPainPlugin{
		client:         newCrawlerHTTPClient(15 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *SocialPainPlugin) Name() string { return "social_pain" }
func (p *SocialPainPlugin) Label() string {
	return "社交平台痛点抓取（HN讨论/百度热搜/Bing搜索）"
}

func (p *SocialPainPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	if limit <= 0 {
		limit = 10
	}

	perSource := limit / 3
	if perSource < 2 {
		perSource = 2
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

	// 来源 1: HN Algolia — 英文 AI/科技痛点讨论
	addTopics(p.fetchHNPain(ctx, perSource))

	// 来源 2: 百度热搜 — 中文 AI/科技痛点热点
	addTopics(p.fetchBaiduPain(ctx, perSource))

	// 来源 3: Bing RSS — 中文平台痛点（知乎/小红书关键词搜索，降级兜底）
	if len(allTopics) < limit {
		remaining := limit - len(allTopics)
		addTopics(p.fetchBingPain(ctx, remaining))
	}

	p.log.Infof("[SocialPain] Fetched %d social pain topics", len(allTopics))
	return allTopics, nil
}

// ── 来源 1: HN Algolia Search ──

type hnAlgoliaResp struct {
	Hits []struct {
		Title    string `json:"title"`
		URL      string `json:"url"`
		ObjectID string `json:"objectID"`
		Points   int    `json:"points"`
		Author   string `json:"author"`
	} `json:"hits"`
}

func (p *SocialPainPlugin) fetchHNPain(ctx context.Context, limit int) []*biz.RawTopic {
	queries := []string{
		"AI tool frustration problem",
		"AI product pain point startup",
		"LLM limitation issue workflow",
	}

	var topics []*biz.RawTopic
	for _, q := range queries {
		if len(topics) >= limit {
			break
		}
		apiURL := fmt.Sprintf(
			"https://hn.algolia.com/api/v1/search?query=%s&tags=story&hitsPerPage=%d",
			url.QueryEscape(q), limit,
		)
		body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
			AcceptLanguage: "en-US,en;q=0.9",
			MaxRetries:     1,
			DetectAntiBot:  false,
		})
		if err != nil {
			p.log.Warnf("[SocialPain] HN Algolia error for '%s': %v", q, err)
			continue
		}

		var resp hnAlgoliaResp
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			p.log.Warnf("[SocialPain] HN Algolia parse error: %v", err)
			continue
		}

		for _, hit := range resp.Hits {
			if len(topics) >= limit {
				break
			}
			title := strings.TrimSpace(hit.Title)
			if title == "" {
				continue
			}
			sourceURL := hit.URL
			if sourceURL == "" {
				sourceURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
			}
			topics = append(topics, &biz.RawTopic{
				Title:      title,
				URL:        sourceURL,
				Source:     "hn_pain",
				Popularity: hit.Points,
				Snippet:    fmt.Sprintf("[HN痛点] score: %d, by: %s", hit.Points, hit.Author),
			})
		}
	}

	if len(topics) == 0 {
		p.log.Warnf("[SocialPain] HN Algolia got 0, fallback to Bing RSS")
		return p.searchBingRSS(ctx, "AI tool frustration pain point site:news.ycombinator.com", limit)
	}

	p.log.Infof("[SocialPain] HN Algolia: %d pain topics", len(topics))
	return topics
}

// ── 来源 2: 百度热搜痛点过滤 ──

func (p *SocialPainPlugin) fetchBaiduPain(ctx context.Context, limit int) []*biz.RawTopic {
	apiURL := "https://top.baidu.com/api/board?platform=wise&tab=realtime"
	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://top.baidu.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[SocialPain] Baidu hot API error: %v, fallback to Bing RSS", err)
		return p.searchBingRSS(ctx, "AI 工具 痛点 吐槽 问题", limit)
	}

	var resp baiduHotBoardResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		p.log.Warnf("[SocialPain] Baidu hot parse error: %v", err)
		return p.searchBingRSS(ctx, "AI 工具 痛点 吐槽 问题", limit)
	}

	// 展开嵌套结构
	type hotItem struct {
		Word   string
		Desc   string
		URL    string
		HotTag string
		Index  int
	}
	var allItems []hotItem
	for _, card := range resp.Data.Cards {
		for _, outer := range card.Content {
			for _, item := range outer.Content {
				allItems = append(allItems, hotItem{
					Word: item.Word, Desc: item.Desc, URL: item.URL, HotTag: item.HotTag, Index: item.Index,
				})
			}
		}
	}

	// 筛选 AI/科技 + 痛点相关话题
	var topics []*biz.RawTopic
	for _, item := range allItems {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(item.Word)
		if title == "" {
			continue
		}
		text := title + " " + item.Desc
		if !isTechRelated(text) {
			continue
		}
		// 优先取痛点相关，但科技话题也可接受（用户在社交平台讨论的科技话题本身有参考价值）
		sourceURL := item.URL
		if sourceURL == "" {
			sourceURL = fmt.Sprintf("https://www.baidu.com/s?wd=%s", url.QueryEscape(title))
		}
		desc := item.Desc
		if desc == "" {
			desc = title
		}
		pop := 500
		if item.HotTag == "3" {
			pop = 1000
		} else if item.HotTag == "2" {
			pop = 800
		} else if item.HotTag == "1" {
			pop = 600
		} else if item.Index > 0 {
			pop = max(100, 500-item.Index*10)
		}
		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        sourceURL,
			Source:     "baidu_social_pain",
			Popularity: pop,
			Snippet:    fmt.Sprintf("[百度热搜痛点] %s", truncateStr(desc, 220)),
		})
	}

	if len(topics) == 0 {
		p.log.Warnf("[SocialPain] Baidu hot got 0 AI/tech pain topics, fallback to Bing RSS")
		return p.searchBingRSS(ctx, "AI 工具 痛点 吐槽 问题", limit)
	}

	p.log.Infof("[SocialPain] Baidu hot pain: %d topics", len(topics))
	return topics
}

// ── 来源 3: Bing RSS 痛点搜索（降级兜底）──

func (p *SocialPainPlugin) fetchBingPain(ctx context.Context, limit int) []*biz.RawTopic {
	queries := []string{
		"知乎 AI 产品 痛点 吐槽",
		"小红书 AI 工具 差评 体验",
	}
	var topics []*biz.RawTopic
	perQ := limit/len(queries) + 1
	for _, q := range queries {
		if len(topics) >= limit {
			break
		}
		results := p.searchBingRSS(ctx, q, perQ)
		for _, t := range results {
			t.Source = "cn_social_pain"
			t.Snippet = fmt.Sprintf("[社交痛点] %s", truncateStr(t.Snippet, 220))
			topics = append(topics, t)
		}
	}
	return topics
}

func (p *SocialPainPlugin) searchBingRSS(ctx context.Context, query string, limit int) []*biz.RawTopic {
	rssURL := fmt.Sprintf("https://www.bing.com/search?q=%s&format=rss&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, rssURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[SocialPain] Bing RSS error for '%s': %v", query, err)
		return nil
	}
	return parseBingRSSItemsToTopics(body, query, limit)
}
