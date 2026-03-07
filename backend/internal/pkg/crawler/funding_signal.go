package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.CrawlerPluginInterface = (*FundingSignalPlugin)(nil)

type fundingSignalQuery struct {
	query  string
	source string
}

// FundingSignalPlugin 融资信号抓取（36氪 / TechCrunch / Crunchbase）
type FundingSignalPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewFundingSignalPlugin(logger log.Logger, getConstraints func() []string) *FundingSignalPlugin {
	return &FundingSignalPlugin{
		client:         &http.Client{Timeout: 15 * time.Second},
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *FundingSignalPlugin) Name() string { return "funding_signal" }
func (p *FundingSignalPlugin) Label() string {
	return "融资信号抓取（36氪/TechCrunch/Crunchbase）"
}

func (p *FundingSignalPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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
			t.Snippet = fmt.Sprintf("[融资信号] %s", truncateStr(t.Snippet, 220))
			if t.Popularity == 0 {
				t.Popularity = 180
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

	p.log.Infof("[FundingSignal] Fetched %d funding topics", len(allTopics))
	return allTopics, nil
}

func (p *FundingSignalPlugin) buildQueries() []fundingSignalQuery {
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

	return []fundingSignalQuery{
		{query: "site:36kr.com AI 融资 新产品" + constraintSuffix, source: "36kr_funding"},
		{query: "site:techcrunch.com AI startup raised seed series A" + constraintSuffix, source: "techcrunch_funding"},
		{query: "site:crunchbase.com AI startup funding round", source: "crunchbase_funding"},
	}
}

func (p *FundingSignalPlugin) searchBing(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		p.log.Warnf("[FundingSignal] request build error: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warnf("[FundingSignal] search error for query '%s': %v", query, err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return parseBingToTopics(string(body), query, limit)
}
