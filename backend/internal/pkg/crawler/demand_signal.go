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
		client:         &http.Client{Timeout: 15 * time.Second},
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

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		p.log.Warnf("[DemandSignal] request build error: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warnf("[DemandSignal] search error for query '%s': %v", query, err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return parseBingToTopics(string(body), query, limit)
}
