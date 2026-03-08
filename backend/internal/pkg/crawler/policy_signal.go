package crawler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.CrawlerPluginInterface = (*PolicySignalPlugin)(nil)

// PolicySignalPlugin 政策信号抓取（人民网时政 / 人民网IT / The Regulatory Review）
type PolicySignalPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewPolicySignalPlugin(logger log.Logger, getConstraints func() []string) *PolicySignalPlugin {
	return &PolicySignalPlugin{
		client:         newCrawlerHTTPClient(20 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *PolicySignalPlugin) Name() string { return "policy_signal" }
func (p *PolicySignalPlugin) Label() string {
	return "政策信号抓取（人民网时政 / 人民网IT / The Regulatory Review）"
}

func (p *PolicySignalPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
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

	// 来源 1: 人民网时政频道 RSS（中文政策/时政新闻）
	addTopics(p.fetchPeoplePolitics(ctx, perSource))

	// 来源 2: 人民网 IT 频道 RSS（中文科技政策/产业动态）
	addTopics(p.fetchPeopleIT(ctx, perSource))

	// 来源 3: The Regulatory Review RSS（英文监管/政策分析）
	if len(allTopics) < limit {
		remaining := limit - len(allTopics)
		addTopics(p.fetchRegulatoryReview(ctx, remaining))
	}

	p.log.Infof("[PolicySignal] Fetched %d policy topics", len(allTopics))
	return allTopics, nil
}

// ── 来源 1: 人民网时政频道 RSS ──

func (p *PolicySignalPlugin) fetchPeoplePolitics(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "http://www.people.com.cn/rss/politics.xml", crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "http://politics.people.com.cn/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[PolicySignal] people.com.cn politics RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "people_politics", "人民网时政", limit)
	p.log.Infof("[PolicySignal] people.com.cn politics RSS: %d items", len(topics))
	return topics
}

// ── 来源 2: 人民网 IT 频道 RSS ──

func (p *PolicySignalPlugin) fetchPeopleIT(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "http://www.people.com.cn/rss/it.xml", crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "http://it.people.com.cn/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[PolicySignal] people.com.cn IT RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "people_it", "人民网IT", limit)
	p.log.Infof("[PolicySignal] people.com.cn IT RSS: %d items", len(topics))
	return topics
}

// ── 来源 3: The Regulatory Review RSS（英文监管政策分析）──

func (p *PolicySignalPlugin) fetchRegulatoryReview(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://www.theregreview.org/feed/", crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://www.theregreview.org/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[PolicySignal] The Regulatory Review RSS error: %v", err)
		return nil
	}

	topics := parseRSSToTopics(body, "regulatory_review", "RegReview", limit)

	// 调整 snippet 前缀
	for _, t := range topics {
		t.Snippet = fmt.Sprintf("[政策监管] %s", truncateStr(t.Snippet, 220))
	}

	p.log.Infof("[PolicySignal] The Regulatory Review RSS: %d items", len(topics))
	return topics
}
