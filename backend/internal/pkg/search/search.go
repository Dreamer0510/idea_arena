package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"idea_arena/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

// Result 搜索结果
type Result struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
	Source  string `json:"source"`
}

// AggregatedSearch 聚合搜索结果
type AggregatedSearch struct {
	Query   string   `json:"query"`
	Results []Result `json:"results"`
	Source  string   `json:"source"`
}

// Aggregator 多源搜索聚合器
type Aggregator struct {
	conf   *conf.Search
	client *http.Client
	log    *log.Helper
}

// NewAggregator 创建搜索聚合器
func NewAggregator(c *conf.Search, logger log.Logger) *Aggregator {
	return &Aggregator{
		conf: c,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		log: log.NewHelper(logger),
	}
}

// MultiSourceSearch 执行多源搜索
func (a *Aggregator) MultiSourceSearch(ctx context.Context, topic string) ([]AggregatedSearch, string) {
	var allSearches []AggregatedSearch

	queriesCN := []string{
		fmt.Sprintf("%s AI创业 最新 2026", topic),
		fmt.Sprintf("%s AI 市场机会 融资", topic),
	}
	queriesEN := []string{
		fmt.Sprintf("%s AI startup latest 2026", topic),
	}

	// Bing 搜索（免费 scraping）
	if a.conf.Bing != nil && a.conf.Bing.Enabled {
		for _, q := range queriesCN {
			results := a.searchBing(ctx, q, 5)
			if len(results) > 0 {
				allSearches = append(allSearches, AggregatedSearch{
					Query: q, Results: results, Source: "bing",
				})
			}
		}
		for _, q := range queriesEN {
			results := a.searchBing(ctx, q, 5)
			if len(results) > 0 {
				allSearches = append(allSearches, AggregatedSearch{
					Query: q, Results: results, Source: "bing",
				})
			}
		}
	}

	// 百度搜索（免费 scraping）
	if a.conf.Baidu != nil && a.conf.Baidu.Enabled {
		results := a.searchBaidu(ctx, queriesCN[0], 5)
		if len(results) > 0 {
			allSearches = append(allSearches, AggregatedSearch{
				Query: queriesCN[0], Results: results, Source: "baidu",
			})
		}
	}

	// 格式化为 LLM 可消费的上下文
	contextStr := formatSearchContext(allSearches)
	return allSearches, contextStr
}

// searchBing 通过 Bing CN 免费搜索（scraping）
func (a *Aggregator) searchBing(ctx context.Context, query string, limit int) []Result {
	searchURL := fmt.Sprintf("https://cn.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		a.log.Warnf("Bing search request error: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Warnf("Bing search error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return parseBingResults(string(body), limit)
}

// searchBaidu 通过百度免费搜索（scraping）
func (a *Aggregator) searchBaidu(ctx context.Context, query string, limit int) []Result {
	searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d", url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		a.log.Warnf("Baidu search request error: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Warnf("Baidu search error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return parseBaiduResults(string(body), limit)
}

// parseBingResults 解析 Bing 搜索结果 HTML
func parseBingResults(html string, limit int) []Result {
	var results []Result

	// 匹配 Bing 搜索结果块
	liRe := regexp.MustCompile(`<li class="b_algo"[^>]*>(.*?)</li>`)
	titleRe := regexp.MustCompile(`<h2><a[^>]*href="([^"]*)"[^>]*>(.*?)</a></h2>`)
	snippetRe := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)

	matches := liRe.FindAllStringSubmatch(html, limit*2)
	for _, m := range matches {
		if len(results) >= limit {
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
			results = append(results, Result{
				Title:   title,
				Snippet: snippet,
				URL:     href,
				Source:  "bing",
			})
		}
	}

	return results
}

// parseBaiduResults 解析百度搜索结果 HTML
func parseBaiduResults(html string, limit int) []Result {
	var results []Result

	// 匹配百度搜索结果
	resultRe := regexp.MustCompile(`<div class="result[^"]*"[^>]*>(.*?)</div>\s*</div>`)
	titleRe := regexp.MustCompile(`<h3[^>]*><a[^>]*href="([^"]*)"[^>]*>(.*?)</a></h3>`)
	snippetRe := regexp.MustCompile(`<span class="content-right_[^"]*">(.*?)</span>`)

	blocks := resultRe.FindAllStringSubmatch(html, limit*2)
	for _, block := range blocks {
		if len(results) >= limit {
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
			results = append(results, Result{
				Title:   title,
				Snippet: snippet,
				URL:     href,
				Source:  "baidu",
			})
		}
	}

	return results
}

// stripHTML 移除 HTML 标签
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	// 解码常见 HTML 实体
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}

// formatSearchContext 将搜索结果格式化为 LLM 可消费的上下文
func formatSearchContext(searches []AggregatedSearch) string {
	if len(searches) == 0 {
		return "=== 未获取到搜索结果 ===\n"
	}

	var sb strings.Builder
	sb.WriteString("=== 多源最新信息搜索结果 ===\n")
	sb.WriteString(fmt.Sprintf("来源：Bing China / 百度\n"))
	sb.WriteString(fmt.Sprintf("搜索时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	seenURLs := make(map[string]bool)
	totalResults := 0

	for _, s := range searches {
		var unique []Result
		for _, r := range s.Results {
			if seenURLs[r.URL] {
				continue
			}
			seenURLs[r.URL] = true
			unique = append(unique, r)
		}
		if len(unique) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("【%s | \"%s\"】\n", strings.ToUpper(s.Source), s.Query))
		for _, r := range unique {
			sb.WriteString(fmt.Sprintf("  • %s\n    %s\n    %s\n", r.Title, r.Snippet, r.URL))
			totalResults++
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n共 %d 条不重复结果，来自 %d 次搜索\n", totalResults, len(searches)))
	return sb.String()
}
