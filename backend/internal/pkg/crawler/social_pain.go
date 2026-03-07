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

var _ biz.CrawlerPluginInterface = (*SocialPainPlugin)(nil)

type socialPainQuery struct {
	query  string
	source string
}

// SocialPainPlugin 社交平台痛点抓取（小红书/Reddit/知乎）
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
	return "社交平台痛点抓取（小红书/Reddit/知乎）"
}

func (p *SocialPainPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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

	seen := make(map[string]struct{})
	allTopics := make([]*biz.RawTopic, 0, limit)

	for _, q := range queries {
		if len(allTopics) >= limit {
			break
		}

		topics := p.searchBing(ctx, q.query, perQuery)
		for _, t := range topics {
			t.Source = q.source
			t.Snippet = fmt.Sprintf("[社交痛点] %s", truncateStr(t.Snippet, 220))

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

	p.log.Infof("[SocialPain] Fetched %d social pain topics", len(allTopics))
	return allTopics, nil
}

func (p *SocialPainPlugin) buildQueries() []socialPainQuery {
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

	return []socialPainQuery{
		{query: "site:xiaohongshu.com AI 工具 吐槽 痛点" + constraintSuffix, source: "xiaohongshu_pain"},
		{query: "site:zhihu.com AI 产品 痛点 需求" + constraintSuffix, source: "zhihu_pain"},
		{query: "site:reddit.com AI tool pain points frustration startup", source: "reddit_pain"},
	}
}

func (p *SocialPainPlugin) searchBing(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     2,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("[SocialPain] search error for query '%s': %v", query, err)
		return nil
	}
	return parseBingToTopics(body, query, limit)
}
