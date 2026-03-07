package crawler

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// ── 默认搜索关键词（当DB无配置时使用）──

var defaultTrendQueriesCN = []string{
	"AI创业 个人开发者 低成本 最新",
	"AI产品 独立开发 新发布 本周",
	"AI SaaS 小团队 落地工具",
	"AI Agent 个人开发 实际应用",
	"低成本 AI创业 变现案例",
}

var defaultTrendQueriesEN = []string{
	"AI indie hacker solo developer tools 2026",
	"AI SaaS bootstrapped startup low cost",
	"AI agent solo developer product launch",
}

// TrendSearchPlugin 搜索热点抓取插件 — 从 Bing/百度搜索最新AI创业热点
type TrendSearchPlugin struct {
	client          *http.Client
	log             *log.Helper
	getConstraints  func() []string                   // 获取约束标签
	getTrendQueries func() (cn []string, en []string) // 获取自定义搜索词
}

var _ biz.CrawlerPluginInterface = (*TrendSearchPlugin)(nil)

func NewTrendSearchPlugin(
	logger log.Logger,
	getConstraints func() []string,
	getTrendQueries func() (cn []string, en []string),
) *TrendSearchPlugin {
	return &TrendSearchPlugin{
		client:          &http.Client{Timeout: 15 * time.Second},
		log:             log.NewHelper(logger),
		getConstraints:  getConstraints,
		getTrendQueries: getTrendQueries,
	}
}

func (p *TrendSearchPlugin) Name() string  { return "trend_search" }
func (p *TrendSearchPlugin) Label() string { return "搜索热点趋势抓取（Bing/百度实时热点）" }

func (p *TrendSearchPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	// 从DB读取自定义搜索词，无则用默认
	cnPool, enPool := defaultTrendQueriesCN, defaultTrendQueriesEN
	if p.getTrendQueries != nil {
		dbCN, dbEN := p.getTrendQueries()
		if len(dbCN) > 0 {
			cnPool = dbCN
		}
		if len(dbEN) > 0 {
			enPool = dbEN
		}
	}

	// 读取约束标签，自动追加到搜索词中
	constraintSuffix := ""
	if p.getConstraints != nil {
		constraints := p.getConstraints()
		if len(constraints) > 0 {
			constraintSuffix = " " + strings.Join(constraints, " ")
		}
	}

	// 随机选 2 个中文 + 1 个英文搜索词
	cnQueries := shuffleStrings(cnPool)
	if len(cnQueries) > 2 {
		cnQueries = cnQueries[:2]
	}
	// 追加约束词
	for i := range cnQueries {
		cnQueries[i] = cnQueries[i] + constraintSuffix
	}

	enQueries := shuffleStrings(enPool)
	if len(enQueries) > 1 {
		enQueries = enQueries[:1]
	}

	var allTopics []*biz.RawTopic

	perQuery := limit / 3
	if perQuery < 3 {
		perQuery = 3
	}

	// Bing 中文搜索
	for _, q := range cnQueries {
		topics := p.searchBing(ctx, q, perQuery)
		allTopics = append(allTopics, topics...)
	}

	// 百度搜索
	if len(cnQueries) > 0 {
		topics := p.searchBaidu(ctx, cnQueries[0], perQuery)
		allTopics = append(allTopics, topics...)
	}

	// Bing 英文搜索
	for _, q := range enQueries {
		topics := p.searchBingEN(ctx, q, perQuery)
		allTopics = append(allTopics, topics...)
	}

	if len(allTopics) > limit {
		allTopics = allTopics[:limit]
	}

	p.log.Infof("[TrendSearch] Fetched %d raw search results", len(allTopics))
	return allTopics, nil
}

func (p *TrendSearchPlugin) searchBing(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://cn.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	body := p.doGet(ctx, searchURL, "zh-CN,zh;q=0.9,en;q=0.8")
	if body == "" {
		return nil
	}
	return parseBingToTopics(body, query, limit)
}

func (p *TrendSearchPlugin) searchBingEN(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	body := p.doGet(ctx, searchURL, "en-US,en;q=0.9")
	if body == "" {
		return nil
	}
	topics := parseBingToTopics(body, query, limit)
	// 标记为英文来源
	for _, t := range topics {
		t.Source = "bing_trend_en"
	}
	return topics
}

func (p *TrendSearchPlugin) searchBaidu(ctx context.Context, query string, limit int) []*biz.RawTopic {
	searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d", url.QueryEscape(query), limit)
	body := p.doGet(ctx, searchURL, "zh-CN,zh;q=0.9")
	if body == "" {
		return nil
	}
	topics := parseBaiduToTopics(body, query, limit)
	// 重新标记来源
	for _, t := range topics {
		t.Source = "baidu_trend"
	}
	return topics
}

func (p *TrendSearchPlugin) doGet(ctx context.Context, rawURL, acceptLang string) string {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", acceptLang)

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warnf("[TrendSearch] HTTP error: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

func shuffleStrings(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(dst), func(i, j int) { dst[i], dst[j] = dst[j], dst[i] })
	return dst
}
