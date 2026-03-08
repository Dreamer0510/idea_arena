package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.CrawlerPluginInterface = (*DemandSignalPlugin)(nil)

type demandSignalQuery struct {
	query  string
	source string
}

// DemandSignalPlugin 付费需求信号抓取（招聘/采购/评论）
type DemandSignalPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewDemandSignalPlugin(logger log.Logger, getConstraints func() []string) *DemandSignalPlugin {
	return &DemandSignalPlugin{
		client:         newCrawlerHTTPClient(15 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *DemandSignalPlugin) Name() string { return "demand_signal" }
func (p *DemandSignalPlugin) Label() string {
	return "付费需求信号抓取（招聘/采购/评论）"
}

func (p *DemandSignalPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	if limit <= 0 {
		limit = 10
	}

	queries := p.buildQueries()
	if len(queries) == 0 {
		return nil, nil
	}

	perQuery := limit / len(queries)
	if perQuery < 2 {
		perQuery = 2
	}

	allTopics := make([]*biz.RawTopic, 0, limit)
	seen := make(map[string]struct{})

	for _, q := range queries {
		if len(allTopics) >= limit {
			break
		}

		topics := p.searchBing(ctx, q.query, perQuery)
		for _, t := range topics {
			t.Source = q.source
			t.Snippet = fmt.Sprintf("[需求信号] %s", truncateStr(t.Snippet, 220))
			if t.Popularity == 0 {
				t.Popularity = 160
			}

			key := strings.ToLower(strings.TrimSpace(t.Title)) + "|" + t.Source
			if key == "|" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			allTopics = append(allTopics, t)
			if len(allTopics) >= limit {
				break
			}
		}
	}

	p.log.Infof("[DemandSignal] Fetched %d demand topics", len(allTopics))
	return allTopics, nil
}

func (p *DemandSignalPlugin) buildQueries() []demandSignalQuery {
	constraintSuffix := ""
	if p.getConstraints != nil {
		constraints := p.getConstraints()
		if len(constraints) > 0 {
			if len(constraints) > 2 {
				constraints = constraints[:2]
			}
			constraintSuffix = " " + strings.Join(constraints, " ")
		}
	}

	return []demandSignalQuery{
		{query: "site:linkedin.com AI product manager hiring workflow automation" + constraintSuffix, source: "linkedin_demand"},
		{query: "site:zhaopin.com 人工智能 招聘 工具 效率" + constraintSuffix, source: "zhaopin_demand"},
		{query: "site:g2.com AI software reviews pricing pain", source: "g2_review_demand"},
	}
}

func (p *DemandSignalPlugin) searchBing(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     2,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("[DemandSignal] Bing HTML error for '%s': %v, fallback to RSS", query, err)
		return p.searchBingRSS(ctx, query, limit)
	}
	topics := parseBingToTopics(body, query, limit)
	if len(topics) > 0 {
		return topics
	}
	p.log.Warnf("[DemandSignal] Bing HTML got 0 for '%s', fallback to RSS", query)
	return p.searchBingRSS(ctx, query, limit)
}

func (p *DemandSignalPlugin) searchBingRSS(ctx context.Context, query string, limit int) []*biz.RawTopic {
	rssURL := fmt.Sprintf("https://www.bing.com/search?q=%s&format=rss&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, rssURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[DemandSignal] Bing RSS error for '%s': %v", query, err)
		return nil
	}
	return parseBingRSSItemsToTopics(body, query, limit)
}
