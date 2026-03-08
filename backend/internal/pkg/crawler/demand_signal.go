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

var _ biz.CrawlerPluginInterface = (*DemandSignalPlugin)(nil)

// DemandSignalPlugin 需求信号抓取（Product Hunt / HN 需求讨论 / Dev.to 开发者社区）
type DemandSignalPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewDemandSignalPlugin(logger log.Logger, getConstraints func() []string) *DemandSignalPlugin {
	return &DemandSignalPlugin{
		client:         newCrawlerHTTPClient(20 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *DemandSignalPlugin) Name() string { return "demand_signal" }
func (p *DemandSignalPlugin) Label() string {
	return "需求信号抓取（Product Hunt / HN需求 / Dev.to）"
}

func (p *DemandSignalPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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

	// 来源 1: Product Hunt Atom feed（新产品发布 = 市场需求信号）
	addTopics(p.fetchProductHunt(ctx, perSource))

	// 来源 2: HN Algolia（AI 需求/招聘相关热门讨论）
	addTopics(p.fetchHNDemand(ctx, perSource))

	// 来源 3: Dev.to RSS（开发者社区热门文章 = 技术需求信号）
	if len(allTopics) < limit {
		remaining := limit - len(allTopics)
		addTopics(p.fetchDevToRSS(ctx, remaining))
	}

	p.log.Infof("[DemandSignal] Fetched %d demand topics", len(allTopics))
	return allTopics, nil
}

// ── 来源 1: Product Hunt Atom feed ──

// atomFeed Atom feed 解析结构
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string    `xml:"title"`
	Links   []atomLink `xml:"link"`
	Content string    `xml:"content"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func (p *DemandSignalPlugin) fetchProductHunt(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://www.producthunt.com/feed", crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://www.producthunt.com/",
		MaxRetries:     2,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[DemandSignal] Product Hunt Atom error: %v", err)
		return nil
	}

	var feed atomFeed
	if err := xml.Unmarshal([]byte(body), &feed); err != nil {
		p.log.Warnf("[DemandSignal] Product Hunt Atom parse error: %v", err)
		return nil
	}

	var topics []*biz.RawTopic
	for i, entry := range feed.Entries {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(entry.Title)
		if title == "" {
			continue
		}

		link := ""
		for _, l := range entry.Links {
			if l.Rel == "alternate" || l.Rel == "" {
				link = l.Href
				break
			}
		}
		if link == "" && len(entry.Links) > 0 {
			link = entry.Links[0].Href
		}
		if link == "" {
			continue
		}

		desc := strings.TrimSpace(stripHTMLTags(entry.Content))
		pop := max(50, 350-i*10)

		snippet := fmt.Sprintf("[ProductHunt] %s", truncateStr(desc, 220))
		if desc == "" {
			snippet = fmt.Sprintf("[ProductHunt] %s", truncateStr(title, 220))
		}

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        link,
			Source:     "producthunt",
			Popularity: pop,
			Snippet:    snippet,
		})
	}

	p.log.Infof("[DemandSignal] Product Hunt: %d products", len(topics))
	return topics
}

// ── 来源 2: HN Algolia（AI 需求/招聘热门讨论）──

func (p *DemandSignalPlugin) fetchHNDemand(ctx context.Context, limit int) []*biz.RawTopic {
	apiURL := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?query=AI+startup+hiring+tool&tags=story&hitsPerPage=%d",
		limit*2,
	)

	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[DemandSignal] HN Algolia error: %v", err)
		return nil
	}

	var resp struct {
		Hits []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			ObjectID    string `json:"objectID"`
			Points      int    `json:"points"`
			NumComments int    `json:"num_comments"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		p.log.Warnf("[DemandSignal] HN Algolia parse error: %v", err)
		return nil
	}

	var topics []*biz.RawTopic
	for _, hit := range resp.Hits {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(hit.Title)
		if title == "" {
			continue
		}

		link := hit.URL
		if link == "" {
			link = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}

		pop := hit.Points
		if pop == 0 {
			pop = 50
		}

		snippet := fmt.Sprintf("[HN需求] %d points, %d comments", hit.Points, hit.NumComments)

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        link,
			Source:     "hn_demand",
			Popularity: pop,
			Snippet:    snippet,
		})
	}

	p.log.Infof("[DemandSignal] HN Algolia demand: %d stories", len(topics))
	return topics
}

// ── 来源 3: Dev.to RSS（开发者社区热门文章）──

// devToDescRe 提取 Dev.to RSS description 中的文本
var devToDescRe = regexp.MustCompile(`<[^>]*>`)

func (p *DemandSignalPlugin) fetchDevToRSS(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://dev.to/feed", crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://dev.to/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[DemandSignal] Dev.to RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "devto", "Dev.to", limit)
	p.log.Infof("[DemandSignal] Dev.to RSS: %d articles", len(topics))
	return topics
}
