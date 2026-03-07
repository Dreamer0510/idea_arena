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

var _ biz.CrawlerPluginInterface = (*PolicySignalPlugin)(nil)

type policySignalQuery struct {
	query  string
	source string
}

// PolicySignalPlugin 政策信号抓取（政府公告/专项资金/行业监管）
type PolicySignalPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewPolicySignalPlugin(logger log.Logger, getConstraints func() []string) *PolicySignalPlugin {
	return &PolicySignalPlugin{
		client:         newCrawlerHTTPClient(15 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *PolicySignalPlugin) Name() string { return "policy_signal" }
func (p *PolicySignalPlugin) Label() string {
	return "政策信号抓取（政府公告/专项资金/监管动态）"
}

func (p *PolicySignalPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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
			t.Snippet = fmt.Sprintf("[政策信号] %s", truncateStr(t.Snippet, 220))
			if t.Popularity == 0 {
				t.Popularity = 140
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

	p.log.Infof("[PolicySignal] Fetched %d policy topics", len(allTopics))
	return allTopics, nil
}

func (p *PolicySignalPlugin) buildQueries() []policySignalQuery {
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

	return []policySignalQuery{
		{query: "site:gov.cn 人工智能 政策 通知 专项" + constraintSuffix, source: "gov_policy"},
		{query: "site:ndrc.gov.cn 人工智能 项目 资金 支持" + constraintSuffix, source: "ndrc_policy"},
		{query: "site:miit.gov.cn 人工智能 监管 指南 试点", source: "miit_policy"},
	}
}

func (p *PolicySignalPlugin) searchBing(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     2,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("[PolicySignal] search error for query '%s': %v", query, err)
		return nil
	}
	return parseBingToTopics(body, query, limit)
}
