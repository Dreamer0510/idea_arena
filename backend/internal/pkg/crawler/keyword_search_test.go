package crawler

import (
	"context"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type keywordSearchMockTransport struct {
	mu          sync.Mutex
	bingQueries []string
	baiduWds    []string
	bingCounts  []string
	baiduRNs    []string
}

func (m *keywordSearchMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	queryValues, _ := neturl.ParseQuery(req.URL.RawQuery)
	if req.URL.Host == "cn.bing.com" && req.URL.Path == "/search" {
		q := queryValues.Get("q")
		m.mu.Lock()
		m.bingQueries = append(m.bingQueries, q)
		m.bingCounts = append(m.bingCounts, queryValues.Get("count"))
		m.mu.Unlock()
		return buildMockHTTPResponse(`<li class="b_algo"><h2><a href="https://example.com/bing">` + q + `</a></h2><p>` + q + `</p></li>`), nil
	}
	if req.URL.Host == "www.baidu.com" && req.URL.Path == "/s" {
		wd := queryValues.Get("wd")
		m.mu.Lock()
		m.baiduWds = append(m.baiduWds, wd)
		m.baiduRNs = append(m.baiduRNs, queryValues.Get("rn"))
		m.mu.Unlock()
		return buildMockHTTPResponse(`<div class="result"><h3><a href="https://example.com/baidu">` + wd + `</a></h3><span class="content-right_test">` + wd + `</span></div></div>`), nil
	}
	return buildMockHTTPResponse(""), nil
}

func buildMockHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func (m *keywordSearchMockTransport) snapshot() (bingQueries []string, baiduWds []string, bingCounts []string, baiduRNs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.bingQueries...),
		append([]string(nil), m.baiduWds...),
		append([]string(nil), m.bingCounts...),
		append([]string(nil), m.baiduRNs...)
}

func TestDedupAndRoundRobinTopics(t *testing.T) {
	bucketA := []*biz.RawTopic{
		{Title: "A1", URL: "https://example.com/a1", Source: "bing_keyword"},
		{Title: "A2", URL: "https://example.com/a2", Source: "bing_keyword"},
	}
	bucketB := []*biz.RawTopic{
		{Title: "B1", URL: "https://example.com/b1", Source: "baidu_keyword"},
		{Title: "A1", URL: "https://example.com/a1", Source: "baidu_keyword"},
	}
	merged := mergeTopicsRoundRobin([][]*biz.RawTopic{bucketA, bucketB}, 4)
	if len(merged) != 4 {
		t.Fatalf("expected 4 merged topics, got %d", len(merged))
	}
	deduped := dedupRawTopics(merged)
	if len(deduped) != 3 {
		t.Fatalf("expected 3 deduped topics, got %d", len(deduped))
	}
	if deduped[0].Title != "A1" || deduped[1].Title != "B1" {
		t.Fatalf("round robin order not applied as expected")
	}
}

func TestBuildSearchQueryForExpandedKeyword(t *testing.T) {
	base := "AI 自动化办公"
	expanded := "AI 自动化办公 在物流报关单据自动识别与录入中的实际应用"

	if !strings.Contains(buildBingQuery(base), "AI 创业 最新") {
		t.Fatalf("base bing query should append suffix")
	}
	if buildBingQuery(expanded) != expanded {
		t.Fatalf("expanded bing query should keep original keyword")
	}

	if !strings.Contains(buildBaiduQuery(base), "AI 创业 机会") {
		t.Fatalf("base baidu query should append suffix")
	}
	if buildBaiduQuery(expanded) != expanded {
		t.Fatalf("expanded baidu query should keep original keyword")
	}
}

func TestDiversifyRawTopicsByDomain(t *testing.T) {
	topics := []*biz.RawTopic{
		{Title: "T1", URL: "https://www.zhihu.com/q1"},
		{Title: "T2", URL: "https://www.zhihu.com/q2"},
		{Title: "T3", URL: "https://36kr.com/p1"},
		{Title: "T4", URL: "https://www.baidu.com/s?wd=ai"},
	}

	diversified := diversifyRawTopicsByDomain(topics, 1)
	if len(diversified) != 3 {
		t.Fatalf("expected 3 topics after domain diversify, got %d", len(diversified))
	}
	if diversified[0].Title != "T1" || diversified[1].Title != "T3" || diversified[2].Title != "T4" {
		t.Fatalf("unexpected diversified order")
	}
}

func TestFilterExpandedKeywordsExcludesBaseKeyword(t *testing.T) {
	base := []string{"AI 自动化办公"}
	expanded := []string{
		"AI 自动化办公",
		"AI自动化办公",
		"AI 自动化办公 在物流报关单据自动识别与录入中的实际应用",
	}

	filtered := filterExpandedKeywords(base, expanded)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 expanded keyword after filtering base keyword, got %d", len(filtered))
	}
	if filtered[0] != "AI 自动化办公 在物流报关单据自动识别与录入中的实际应用" {
		t.Fatalf("unexpected filtered keyword: %s", filtered[0])
	}
}

func TestFilterTopicsByKeywordRelevance(t *testing.T) {
	keyword := "AI 自动化办公 在律师事务所合同条款合规性审查中的实际应用"
	topics := []*biz.RawTopic{
		{
			Title:   "AI 自动化办公在律师事务所合同条款合规性审查中的应用实践",
			URL:     "https://example.com/legal-ai",
			Snippet: "落地案例总结",
		},
		{
			Title:   "Movie Theater in Bellevue: Lincoln Square Cinemas",
			URL:     "https://example.com/movie",
			Snippet: "Get tickets and showtimes",
		},
	}

	filtered := filterTopicsByKeywordRelevance(keyword, topics)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 relevant topic, got %d", len(filtered))
	}
	if filtered[0].URL != "https://example.com/legal-ai" {
		t.Fatalf("unexpected kept topic: %s", filtered[0].URL)
	}
}

func TestFilterTopicsByKeywordRelevance_IgnoresKeywordPrefixNoise(t *testing.T) {
	keyword := "AI 自动化办公 在跨境电商海外仓库存管理与智能补货中的实际应用"
	topics := []*biz.RawTopic{
		{
			Title:   "Unrelated title",
			URL:     "https://example.com/random",
			Snippet: "[关键词: AI 自动化办公 在跨境电商海外仓库存管理与智能补货中的实际应用] random text only",
		},
	}

	filtered := filterTopicsByKeywordRelevance(keyword, topics)
	if len(filtered) != 0 {
		t.Fatalf("expected unrelated topic to be filtered out, got %d", len(filtered))
	}
}

func TestKeywordSearchPluginFetch_UsesExpandedKeywordsOnly(t *testing.T) {
	baseKeyword := "AI 自动化办公"
	expandedKeyword := "AI 自动化办公 在物流报关单据自动识别与录入中的实际应用"

	mockTransport := &keywordSearchMockTransport{}
	plugin := NewKeywordSearchPlugin(log.NewStdLogger(io.Discard), func() []string {
		return []string{baseKeyword}
	})
	plugin.client = &http.Client{
		Timeout:   10 * time.Second,
		Transport: mockTransport,
	}
	plugin.SetAIExpansionHooks(
		func() string {
			return `{"ai_expansion":{"enabled":true,"provider_id":1,"model":"mock","extra_per_keyword":1}}`
		},
		func(ctx context.Context, keywords []string, providerID int64, model string, extraPerKeyword int) ([]string, error) {
			return []string{baseKeyword, expandedKeyword}, nil
		},
	)

	topics, err := plugin.Fetch(context.Background(), 6)
	if err != nil {
		t.Fatalf("fetch should not fail: %v", err)
	}
	if len(topics) == 0 {
		t.Fatalf("expected non-empty topics")
	}

	bingQueries, baiduWds, bingCounts, baiduRNs := mockTransport.snapshot()
	if len(bingQueries) == 0 || len(baiduWds) == 0 {
		t.Fatalf("expected bing/baidu search requests, got bing=%d baidu=%d", len(bingQueries), len(baiduWds))
	}
	for _, count := range bingCounts {
		if count != "10" {
			t.Fatalf("expected bing count=10, got %s", count)
		}
	}
	for _, rn := range baiduRNs {
		if rn != "10" {
			t.Fatalf("expected baidu rn=10, got %s", rn)
		}
	}

	baseBingQuery := buildBingQuery(baseKeyword)
	baseBaiduQuery := buildBaiduQuery(baseKeyword)
	for _, q := range bingQueries {
		if q == baseBingQuery {
			t.Fatalf("should not search base bing query when AI expansion enabled: %s", q)
		}
	}
	for _, wd := range baiduWds {
		if wd == baseBaiduQuery {
			t.Fatalf("should not search base baidu query when AI expansion enabled: %s", wd)
		}
	}

	expandedBingQuery := buildBingQuery(expandedKeyword)
	expandedBaiduQuery := buildBaiduQuery(expandedKeyword)
	foundExpandedBing := false
	for _, q := range bingQueries {
		if q == expandedBingQuery {
			foundExpandedBing = true
			break
		}
	}
	if !foundExpandedBing {
		t.Fatalf("expected expanded bing query not found, queries=%v", bingQueries)
	}
	foundExpandedBaidu := false
	for _, wd := range baiduWds {
		if wd == expandedBaiduQuery {
			foundExpandedBaidu = true
			break
		}
	}
	if !foundExpandedBaidu {
		t.Fatalf("expected expanded baidu query not found, wds=%v", baiduWds)
	}
}

func TestSelectTopTopicsByRelevance_ReturnsTopFive(t *testing.T) {
	scored := []scoredRawTopic{
		{Topic: &biz.RawTopic{Title: "T1", URL: "https://e.com/1"}, Score: 1},
		{Topic: &biz.RawTopic{Title: "T2", URL: "https://e.com/2"}, Score: 5},
		{Topic: &biz.RawTopic{Title: "T3", URL: "https://e.com/3"}, Score: 3},
		{Topic: &biz.RawTopic{Title: "T4", URL: "https://e.com/4"}, Score: 7},
		{Topic: &biz.RawTopic{Title: "T5", URL: "https://e.com/5"}, Score: 2},
		{Topic: &biz.RawTopic{Title: "T6", URL: "https://e.com/6"}, Score: 6},
		{Topic: &biz.RawTopic{Title: "T7", URL: "https://e.com/7"}, Score: 4},
	}
	result := selectTopTopicsByRelevance(scored, 5)
	if len(result) != 5 {
		t.Fatalf("expected 5 topics, got %d", len(result))
	}
	expectedOrder := []string{"T4", "T6", "T2", "T7", "T3"}
	for idx, expectedTitle := range expectedOrder {
		if result[idx].Title != expectedTitle {
			t.Fatalf("unexpected order at %d: got=%s want=%s", idx, result[idx].Title, expectedTitle)
		}
	}
}

func TestKeywordSearchPluginFetch_ReturnsErrorWhenAIExpansionEmpty(t *testing.T) {
	plugin := NewKeywordSearchPlugin(log.NewStdLogger(io.Discard), func() []string {
		return []string{"AI 自动化办公"}
	})
	plugin.SetAIExpansionHooks(
		func() string {
			return `{"ai_expansion":{"enabled":true,"provider_id":1,"model":"mock","extra_per_keyword":2}}`
		},
		func(ctx context.Context, keywords []string, providerID int64, model string, extraPerKeyword int) ([]string, error) {
			return []string{}, nil
		},
	)

	_, err := plugin.Fetch(context.Background(), 5)
	if err == nil {
		t.Fatalf("expected error when ai expansion returns empty keywords")
	}
	if !strings.Contains(err.Error(), "AI模型扩展关键词失败，请检查API稳定性") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseBingRSSItemsToTopics(t *testing.T) {
	content := `<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>AI 创业案例一</title>
      <link>https://example.com/a</link>
      <description>描述A</description>
    </item>
    <item>
      <title>AI 创业案例二</title>
      <link>https://example.com/b</link>
      <description>描述B</description>
    </item>
  </channel>
</rss>`

	topics := parseBingRSSItemsToTopics(content, "AI", 10)
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}
	if topics[0].Source != "bing_keyword" {
		t.Fatalf("unexpected source: %s", topics[0].Source)
	}
	if !strings.Contains(topics[0].Snippet, "[关键词: AI]") {
		t.Fatalf("unexpected snippet: %s", topics[0].Snippet)
	}
}

func TestParseBaiduSugrecToTopics(t *testing.T) {
	content := `{"q":"ai创业","g":[{"q":"ai创业项目"},{"q":"ai创业方向"}]}`
	topics := parseBaiduSugrecToTopics(content, "AI", 10)
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}
	if topics[0].Source != "baidu_keyword" {
		t.Fatalf("unexpected source: %s", topics[0].Source)
	}
	if !strings.Contains(topics[0].URL, "https://www.baidu.com/s?wd=") {
		t.Fatalf("unexpected url: %s", topics[0].URL)
	}
}

func TestKeywordSearchPlugin_LiveBingAndBaidu(t *testing.T) {
	if os.Getenv("LIVE_CRAWLER_TEST") != "1" {
		t.Skip("set LIVE_CRAWLER_TEST=1 to run live crawler validation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	plugin := NewKeywordSearchPlugin(log.NewStdLogger(io.Discard), func() []string {
		return []string{"AI 自动化办公"}
	})

	bingTopics := plugin.searchBing(ctx, "AI 自动化办公", 5)
	if len(bingTopics) == 0 {
		t.Fatalf("bing topics should not be empty")
	}
	for _, topic := range bingTopics {
		if topic.Source != "bing_keyword" {
			t.Fatalf("unexpected bing source: %s", topic.Source)
		}
		if topic.URL == "" {
			t.Fatalf("bing topic url should not be empty")
		}
	}

	baiduTopics := plugin.searchBaidu(ctx, "AI 自动化办公", 5)
	if len(baiduTopics) == 0 {
		t.Fatalf("baidu topics should not be empty")
	}
	for _, topic := range baiduTopics {
		if topic.Source != "baidu_keyword" {
			t.Fatalf("unexpected baidu source: %s", topic.Source)
		}
		if !strings.Contains(topic.URL, "baidu.com/s?wd=") {
			t.Fatalf("unexpected baidu url: %s", topic.URL)
		}
	}
}
