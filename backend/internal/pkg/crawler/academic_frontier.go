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

var _ biz.CrawlerPluginInterface = (*AcademicFrontierPlugin)(nil)

type academicFrontierQuery struct {
	query  string
	source string
}

// AcademicFrontierPlugin 学术前沿抓取（arXiv / HuggingFace Papers）
type AcademicFrontierPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewAcademicFrontierPlugin(logger log.Logger, getConstraints func() []string) *AcademicFrontierPlugin {
	return &AcademicFrontierPlugin{
		client:         newCrawlerHTTPClient(15 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *AcademicFrontierPlugin) Name() string { return "academic_frontier" }
func (p *AcademicFrontierPlugin) Label() string {
	return "学术前沿抓取（arXiv/HuggingFace Papers）"
}

func (p *AcademicFrontierPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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
			t.Snippet = fmt.Sprintf("[学术前沿] %s", truncateStr(t.Snippet, 220))
			if t.Popularity == 0 {
				t.Popularity = 120
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

	p.log.Infof("[AcademicFrontier] Fetched %d frontier topics", len(allTopics))
	return allTopics, nil
}

func (p *AcademicFrontierPlugin) buildQueries() []academicFrontierQuery {
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

	return []academicFrontierQuery{
		{query: "site:arxiv.org abs AI agent framework tool" + constraintSuffix, source: "arxiv_frontier"},
		{query: "site:huggingface.co/papers AI trending open source" + constraintSuffix, source: "hf_papers_frontier"},
		{query: "site:paperswithcode.com AI state of the art practical", source: "paperswithcode_frontier"},
	}
}

func (p *AcademicFrontierPlugin) searchBing(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     2,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("[AcademicFrontier] search error for query '%s': %v", query, err)
		return nil
	}
	return parseBingToTopics(body, query, limit)
}
