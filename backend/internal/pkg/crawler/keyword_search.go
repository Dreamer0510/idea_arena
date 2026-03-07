package crawler

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// compile-time check: KeywordSearchPlugin implements biz.CrawlerPluginInterface
var _ biz.CrawlerPluginInterface = (*KeywordSearchPlugin)(nil)

const (
	keywordSearchPerEngineLimit = 10
	keywordSearchFinalTopN      = 5
)

// KeywordSearchPlugin 搜索引擎关键词爬虫 — 用预设关键词从 Bing/百度搜索
type KeywordSearchPlugin struct {
	client             *http.Client
	log                *log.Helper
	keywords           func() []string // 动态获取关键词列表
	getPluginConfig    func() string
	expandKeywordsByAI func(ctx context.Context, keywords []string, providerID int64, model string, extraPerKeyword int) ([]string, error)
}

// NewKeywordSearchPlugin 创建搜索引擎关键词爬虫
func NewKeywordSearchPlugin(logger log.Logger, keywordsFn func() []string) *KeywordSearchPlugin {
	return &KeywordSearchPlugin{
		client:          newCrawlerHTTPClient(15 * time.Second),
		log:             log.NewHelper(logger),
		keywords:        keywordsFn,
		getPluginConfig: func() string { return "{}" },
	}
}

func (p *KeywordSearchPlugin) SetAIExpansionHooks(
	getConfigFn func() string,
	expandFn func(ctx context.Context, keywords []string, providerID int64, model string, extraPerKeyword int) ([]string, error),
) {
	if getConfigFn != nil {
		p.getPluginConfig = getConfigFn
	}
	p.expandKeywordsByAI = expandFn
}

func (p *KeywordSearchPlugin) Name() string  { return "keyword_search" }
func (p *KeywordSearchPlugin) Label() string { return "搜索引擎关键词抓取（Bing/百度）" }

func (p *KeywordSearchPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	baseKeywords := dedupKeywords(p.keywords())
	keywords := baseKeywords
	if len(baseKeywords) == 0 {
		return nil, nil
	}

	if cfg := p.readAIExpansionConfig(); cfg.Enabled && p.expandKeywordsByAI != nil {
		expanded, err := p.expandKeywordsByAI(ctx, baseKeywords, cfg.ProviderID, cfg.Model, cfg.ExtraPerKeyword)
		if err != nil {
			p.log.Warnf("keyword AI expansion failed: %v", err)
			return nil, fmt.Errorf("AI模型扩展关键词失败，请检查API稳定性")
		}
		keywords = dedupKeywords(filterExpandedKeywords(baseKeywords, expanded))
		if len(keywords) == 0 {
			p.log.Warnf("keyword AI expansion enabled but got 0 expanded keywords")
			return nil, fmt.Errorf("AI模型扩展关键词失败，请检查API稳定性")
		}
		if len(keywords) > 12 {
			keywords = keywords[:12]
		}
		p.log.Infof(
			"keyword AI expansion enabled: base=%d expanded=%d using_expanded_only=%d preview=%v",
			len(baseKeywords), len(expanded), len(keywords), previewKeywords(keywords, 3),
		)
	}

	var scoredTopics []scoredRawTopic
	for _, kw := range keywords {
		bingTopics, baiduTopics := p.searchKeywordByEnginesConcurrently(ctx, kw, keywordSearchPerEngineLimit)
		merged := dedupRawTopics(append(bingTopics, baiduTopics...))
		if len(merged) == 0 {
			continue
		}
		for _, topic := range merged {
			if topic == nil {
				continue
			}
			relevanceScore := scoreTopicRelevance(kw, topic)
			if relevanceScore < minRelevanceScore(kw) {
				continue
			}
			scoredTopics = append(scoredTopics, scoredRawTopic{
				Topic: topic,
				Score: relevanceScore,
			})
		}
	}

	resultLimit := keywordSearchFinalTopN
	if limit > 0 && limit < resultLimit {
		resultLimit = limit
	}
	if resultLimit <= 0 {
		resultLimit = keywordSearchFinalTopN
	}

	return selectTopTopicsByRelevance(scoredTopics, resultLimit), nil
}

func (p *KeywordSearchPlugin) searchKeywordByEnginesConcurrently(ctx context.Context, keyword string, perEngineLimit int) ([]*biz.RawTopic, []*biz.RawTopic) {
	p.log.Infof("keyword concurrent search: keyword=%s engines=bing,baidu per_engine_limit=%d", keyword, perEngineLimit)
	var wg sync.WaitGroup
	wg.Add(2)

	var bingTopics []*biz.RawTopic
	var baiduTopics []*biz.RawTopic

	go func() {
		defer wg.Done()
		bingTopics = p.searchBing(ctx, keyword, perEngineLimit)
	}()
	go func() {
		defer wg.Done()
		baiduTopics = p.searchBaidu(ctx, keyword, perEngineLimit)
	}()

	wg.Wait()
	return bingTopics, baiduTopics
}

func (p *KeywordSearchPlugin) searchBing(ctx context.Context, keyword string, limit int) []*biz.RawTopic {
	query := buildBingQuery(keyword)
	searchURL := fmt.Sprintf("https://cn.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)
	htmlCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	body, err := fetchHTMLWithRetry(htmlCtx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://cn.bing.com/",
		MaxRetries:     0,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("Bing HTML search error for '%s': %v", keyword, err)
		return p.searchBingRSS(ctx, query, keyword, limit)
	}
	topics := parseBingToTopics(body, keyword, limit)
	if len(topics) > 0 {
		return topics
	}
	p.log.Warnf("Bing HTML search got 0 topics for '%s', switch to Bing RSS", keyword)
	return p.searchBingRSS(ctx, query, keyword, limit)
}

type keywordSearchPluginConfig struct {
	AIExpansion keywordSearchAIExpansion `json:"ai_expansion"`
}

type keywordSearchAIExpansion struct {
	Enabled         bool   `json:"enabled"`
	ProviderID      int64  `json:"provider_id"`
	Model           string `json:"model"`
	ExtraPerKeyword int    `json:"extra_per_keyword"`
}

func (p *KeywordSearchPlugin) readAIExpansionConfig() keywordSearchAIExpansion {
	raw := "{}"
	if p.getPluginConfig != nil {
		raw = p.getPluginConfig()
	}
	var cfg keywordSearchPluginConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return keywordSearchAIExpansion{Enabled: false}
	}
	if cfg.AIExpansion.ExtraPerKeyword <= 0 {
		cfg.AIExpansion.ExtraPerKeyword = 2
	}
	if cfg.AIExpansion.ExtraPerKeyword > 5 {
		cfg.AIExpansion.ExtraPerKeyword = 5
	}
	return cfg.AIExpansion
}

func dedupKeywords(keywords []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		candidate := strings.TrimSpace(keyword)
		if candidate == "" {
			continue
		}
		key := strings.ToLower(candidate)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func filterExpandedKeywords(baseKeywords, expanded []string) []string {
	baseSet := make(map[string]struct{}, len(baseKeywords))
	for _, keyword := range baseKeywords {
		normalized := normalizeKeywordForCompare(keyword)
		if normalized == "" {
			continue
		}
		baseSet[normalized] = struct{}{}
	}

	result := make([]string, 0, len(expanded))
	for _, keyword := range expanded {
		candidate := strings.TrimSpace(keyword)
		if candidate == "" {
			continue
		}
		if _, exists := baseSet[normalizeKeywordForCompare(candidate)]; exists {
			continue
		}
		result = append(result, candidate)
	}
	return result
}

func normalizeKeywordForCompare(keyword string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(keyword)), ""))
}

func filterTopicsByKeywordRelevance(keyword string, topics []*biz.RawTopic) []*biz.RawTopic {
	if strings.TrimSpace(keyword) == "" || len(topics) == 0 {
		return topics
	}
	filtered := make([]*biz.RawTopic, 0, len(topics))
	for _, topic := range topics {
		if topic == nil {
			continue
		}
		if scoreTopicRelevance(keyword, topic) >= minRelevanceScore(keyword) {
			filtered = append(filtered, topic)
		}
	}
	return filtered
}

func minRelevanceScore(_ string) int {
	return 1
}

func scoreTopicRelevance(keyword string, topic *biz.RawTopic) int {
	if topic == nil {
		return 0
	}
	keywordNormalized := normalizeKeywordForCompare(keyword)
	if keywordNormalized == "" {
		return 0
	}
	snippet := stripKeywordPrefix(topic.Snippet)
	content := normalizeKeywordForCompare(topic.Title + " " + snippet + " " + topic.URL)
	if content == "" {
		return 0
	}
	score := 0
	if strings.Contains(content, keywordNormalized) {
		score += 3
	}
	for _, term := range extractKeywordTerms(keyword) {
		if term == "" || term == keywordNormalized {
			continue
		}
		if strings.Contains(content, term) {
			score++
		}
	}
	return score
}

var keywordPrefixPattern = regexp.MustCompile(`^\[关键词:\s*.*?\]\s*`)

func stripKeywordPrefix(snippet string) string {
	return strings.TrimSpace(keywordPrefixPattern.ReplaceAllString(strings.TrimSpace(snippet), ""))
}

func extractKeywordTerms(keyword string) []string {
	replacer := strings.NewReplacer(
		"中的实际应用", " ",
		"实际应用", " ",
		"在", " ",
		"中的", " ",
		"行业", " ",
		"场景", " ",
		"应用", " ",
		"与", " ",
		"和", " ",
		"及", " ",
		"（", " ",
		"）", " ",
		"(", " ",
		")", " ",
		"，", " ",
		"。", " ",
		"：", " ",
		":", " ",
		"、", " ",
		"/", " ",
		"\\", " ",
		"-", " ",
		"_", " ",
	)
	cleaned := replacer.Replace(strings.TrimSpace(keyword))
	rawTerms := strings.Fields(cleaned)
	seen := make(map[string]struct{}, len(rawTerms))
	result := make([]string, 0, len(rawTerms))
	for _, term := range rawTerms {
		normalized := normalizeKeywordForCompare(term)
		if normalized == "" {
			continue
		}
		if len([]rune(normalized)) < 2 {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func previewKeywords(keywords []string, max int) []string {
	if max <= 0 || len(keywords) == 0 {
		return nil
	}
	if len(keywords) <= max {
		return keywords
	}
	return keywords[:max]
}

type scoredRawTopic struct {
	Topic *biz.RawTopic
	Score int
}

func selectTopTopicsByRelevance(scoredTopics []scoredRawTopic, limit int) []*biz.RawTopic {
	if limit <= 0 || len(scoredTopics) == 0 {
		return nil
	}
	sort.SliceStable(scoredTopics, func(i, j int) bool {
		if scoredTopics[i].Score == scoredTopics[j].Score {
			return i < j
		}
		return scoredTopics[i].Score > scoredTopics[j].Score
	})
	seen := make(map[string]struct{}, len(scoredTopics))
	result := make([]*biz.RawTopic, 0, limit)
	for _, item := range scoredTopics {
		if item.Topic == nil {
			continue
		}
		titleKey := strings.ToLower(strings.TrimSpace(item.Topic.Title))
		urlKey := strings.ToLower(strings.TrimSpace(item.Topic.URL))
		dedupKey := titleKey + "|" + urlKey
		if dedupKey == "|" {
			continue
		}
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}
		result = append(result, item.Topic)
		if len(result) >= limit {
			break
		}
	}
	return result
}

func dedupRawTopics(topics []*biz.RawTopic) []*biz.RawTopic {
	seen := make(map[string]struct{})
	result := make([]*biz.RawTopic, 0, len(topics))
	for _, topic := range topics {
		if topic == nil {
			continue
		}
		titleKey := strings.ToLower(strings.TrimSpace(topic.Title))
		urlKey := strings.ToLower(strings.TrimSpace(topic.URL))
		dedupKey := titleKey + "|" + urlKey
		if dedupKey == "|" {
			continue
		}
		if _, exists := seen[dedupKey]; exists {
			continue
		}
		seen[dedupKey] = struct{}{}
		result = append(result, topic)
	}
	return result
}

func mergeTopicsRoundRobin(buckets [][]*biz.RawTopic, limit int) []*biz.RawTopic {
	if len(buckets) == 0 || limit <= 0 {
		return nil
	}
	indexes := make([]int, len(buckets))
	result := make([]*biz.RawTopic, 0, limit)

	for len(result) < limit {
		progress := false
		for i := range buckets {
			if len(result) >= limit {
				break
			}
			if indexes[i] >= len(buckets[i]) {
				continue
			}
			result = append(result, buckets[i][indexes[i]])
			indexes[i]++
			progress = true
		}
		if !progress {
			break
		}
	}
	return result
}

func diversifyRawTopicsByDomain(topics []*biz.RawTopic, maxPerDomain int) []*biz.RawTopic {
	if maxPerDomain <= 0 {
		return topics
	}
	domainCounts := make(map[string]int)
	result := make([]*biz.RawTopic, 0, len(topics))
	for _, topic := range topics {
		if topic == nil {
			continue
		}
		domain := extractDomain(topic.URL)
		if domain == "" {
			result = append(result, topic)
			continue
		}
		if domainCounts[domain] >= maxPerDomain {
			continue
		}
		domainCounts[domain]++
		result = append(result, topic)
	}
	return result
}

func extractDomain(rawURL string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsedURL.Hostname())
}

func looksLikeExpandedKeyword(keyword string) bool {
	k := strings.TrimSpace(keyword)
	return strings.Contains(k, "实际应用") || strings.Contains(k, "行业") || strings.Contains(k, "场景")
}

func buildBingQuery(keyword string) string {
	if looksLikeExpandedKeyword(keyword) {
		return keyword
	}
	return fmt.Sprintf("%s AI 创业 最新", keyword)
}

func buildBaiduQuery(keyword string) string {
	if looksLikeExpandedKeyword(keyword) {
		return keyword
	}
	return fmt.Sprintf("%s AI 创业 机会", keyword)
}

func (p *KeywordSearchPlugin) searchBaidu(ctx context.Context, keyword string, limit int) []*biz.RawTopic {
	query := buildBaiduQuery(keyword)
	searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d", url.QueryEscape(query), limit)
	htmlCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	body, err := fetchHTMLWithRetry(htmlCtx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://www.baidu.com/",
		MaxRetries:     0,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("Baidu HTML search error for '%s': %v", keyword, err)
		return p.searchBaiduSugrec(ctx, query, keyword, limit)
	}
	topics := parseBaiduToTopics(body, keyword, limit)
	if len(topics) > 0 {
		return topics
	}
	p.log.Warnf("Baidu HTML search got 0 topics for '%s', switch to Sugrec", keyword)
	return p.searchBaiduSugrec(ctx, query, keyword, limit)
}

func (p *KeywordSearchPlugin) searchBingRSS(ctx context.Context, query, keyword string, limit int) []*biz.RawTopic {
	rssCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	searchURL := fmt.Sprintf(
		"https://www.bing.com/search?q=%s&format=rss&count=%d&setlang=zh-cn",
		url.QueryEscape(query),
		limit,
	)
	body, err := fetchHTMLWithRetry(rssCtx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     0,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("Bing RSS search error for '%s': %v", keyword, err)
		return nil
	}
	topics := parseBingRSSItemsToTopics(body, keyword, limit)
	if len(topics) == 0 {
		p.log.Warnf("Bing RSS search got 0 topics for '%s'", keyword)
	}
	return topics
}

func (p *KeywordSearchPlugin) searchBaiduSugrec(ctx context.Context, query, keyword string, limit int) []*biz.RawTopic {
	sugrecCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	searchURL := fmt.Sprintf("https://www.baidu.com/sugrec?prod=pc&wd=%s", url.QueryEscape(query))
	body, err := fetchHTMLWithRetry(sugrecCtx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://www.baidu.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("Baidu Sugrec search error for '%s': %v", keyword, err)
		return nil
	}
	topics := parseBaiduSugrecToTopics(body, keyword, limit)
	if len(topics) == 0 {
		p.log.Warnf("Baidu Sugrec got 0 topics for '%s'", keyword)
	}
	return topics
}

func parseBingToTopics(html, keyword string, limit int) []*biz.RawTopic {
	var topics []*biz.RawTopic

	liRe := regexp.MustCompile(`<li class="b_algo"[^>]*>(.*?)</li>`)
	titleRe := regexp.MustCompile(`<h2><a[^>]*href="([^"]*)"[^>]*>(.*?)</a></h2>`)
	snippetRe := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)

	matches := liRe.FindAllStringSubmatch(html, limit*2)
	for _, m := range matches {
		if len(topics) >= limit {
			break
		}
		block := m[1]
		titleMatch := titleRe.FindStringSubmatch(block)
		if titleMatch == nil {
			continue
		}
		href := titleMatch[1]
		title := stripHTML(titleMatch[2])

		snippet := ""
		snippetMatch := snippetRe.FindStringSubmatch(block)
		if snippetMatch != nil {
			snippet = stripHTML(snippetMatch[1])
		}

		if title != "" && href != "" {
			topics = append(topics, &biz.RawTopic{
				Title:   title,
				URL:     href,
				Source:  "bing_keyword",
				Snippet: fmt.Sprintf("[关键词: %s] %s", keyword, snippet),
			})
		}
	}
	return topics
}

func parseBaiduToTopics(html, keyword string, limit int) []*biz.RawTopic {
	var topics []*biz.RawTopic

	resultRe := regexp.MustCompile(`<div class="result[^"]*"[^>]*>(.*?)</div>\s*</div>`)
	titleRe := regexp.MustCompile(`<h3[^>]*><a[^>]*href="([^"]*)"[^>]*>(.*?)</a></h3>`)
	snippetRe := regexp.MustCompile(`<span class="content-right_[^"]*">(.*?)</span>`)

	blocks := resultRe.FindAllStringSubmatch(html, limit*2)
	for _, block := range blocks {
		if len(topics) >= limit {
			break
		}
		content := block[1]
		titleMatch := titleRe.FindStringSubmatch(content)
		if titleMatch == nil {
			continue
		}
		href := titleMatch[1]
		title := stripHTML(titleMatch[2])

		snippet := ""
		snippetMatch := snippetRe.FindStringSubmatch(content)
		if snippetMatch != nil {
			snippet = stripHTML(snippetMatch[1])
		}

		if title != "" {
			topics = append(topics, &biz.RawTopic{
				Title:   title,
				URL:     href,
				Source:  "baidu_keyword",
				Snippet: fmt.Sprintf("[关键词: %s] %s", keyword, truncateStr(snippet, 200)),
			})
		}
	}
	return topics
}

type bingRSSDoc struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

func parseBingRSSItemsToTopics(content, keyword string, limit int) []*biz.RawTopic {
	var doc bingRSSDoc
	if err := xml.Unmarshal([]byte(content), &doc); err != nil {
		return nil
	}
	var topics []*biz.RawTopic
	seen := make(map[string]struct{})
	for _, item := range doc.Channel.Items {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(html.UnescapeString(stripHTML(item.Title)))
		link := strings.TrimSpace(html.UnescapeString(item.Link))
		desc := strings.TrimSpace(html.UnescapeString(stripHTML(item.Description)))
		if title == "" || link == "" {
			continue
		}
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			continue
		}
		key := strings.ToLower(title) + "|" + link
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		topics = append(topics, &biz.RawTopic{
			Title:   title,
			URL:     link,
			Source:  "bing_keyword",
			Snippet: fmt.Sprintf("[关键词: %s] %s", keyword, truncateStr(desc, 200)),
		})
	}
	return topics
}

type baiduSugrecResp struct {
	G []struct {
		Q string `json:"q"`
	} `json:"g"`
}

func parseBaiduSugrecToTopics(content, keyword string, limit int) []*biz.RawTopic {
	var resp baiduSugrecResp
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil
	}
	var topics []*biz.RawTopic
	seen := make(map[string]struct{})
	for _, item := range resp.G {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(item.Q)
		if title == "" {
			continue
		}
		dedupKey := strings.ToLower(title)
		if _, ok := seen[dedupKey]; ok {
			continue
		}
		seen[dedupKey] = struct{}{}
		searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s", url.QueryEscape(title))
		topics = append(topics, &biz.RawTopic{
			Title:   title,
			URL:     searchURL,
			Source:  "baidu_keyword",
			Snippet: fmt.Sprintf("[关键词: %s] 百度搜索联想词：%s", keyword, truncateStr(title, 120)),
		})
	}
	return topics
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
