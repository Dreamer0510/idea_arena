package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// compile-time check: KeywordSearchPlugin implements biz.CrawlerPluginInterface
var _ biz.CrawlerPluginInterface = (*KeywordSearchPlugin)(nil)

// KeywordSearchPlugin 搜索引擎关键词爬虫 — 用预设关键词从 Bing/百度搜索
type KeywordSearchPlugin struct {
	client   *http.Client
	log      *log.Helper
	keywords func() []string // 动态获取关键词列表
}

// NewKeywordSearchPlugin 创建搜索引擎关键词爬虫
func NewKeywordSearchPlugin(logger log.Logger, keywordsFn func() []string) *KeywordSearchPlugin {
	return &KeywordSearchPlugin{
		client:   &http.Client{Timeout: 15 * time.Second},
		log:      log.NewHelper(logger),
		keywords: keywordsFn,
	}
}

func (p *KeywordSearchPlugin) Name() string  { return "keyword_search" }
func (p *KeywordSearchPlugin) Label() string { return "搜索引擎关键词抓取（Bing/百度）" }

func (p *KeywordSearchPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	keywords := p.keywords()
	if len(keywords) == 0 {
		return nil, nil
	}

	var allTopics []*biz.RawTopic
	perKeyword := limit / len(keywords)
	if perKeyword < 2 {
		perKeyword = 2
	}

	for _, kw := range keywords {
		if len(allTopics) >= limit {
			break
		}

		// Bing 搜索
		bingTopics := p.searchBing(ctx, kw, perKeyword)
		allTopics = append(allTopics, bingTopics...)

		// 百度搜索
		baiduTopics := p.searchBaidu(ctx, kw, perKeyword)
		allTopics = append(allTopics, baiduTopics...)
	}

	// 截断到 limit
	if len(allTopics) > limit {
		allTopics = allTopics[:limit]
	}

	return allTopics, nil
}

func (p *KeywordSearchPlugin) searchBing(ctx context.Context, keyword string, limit int) []*biz.RawTopic {
	query := fmt.Sprintf("%s AI 创业 最新", keyword)
	searchURL := fmt.Sprintf("https://cn.bing.com/search?q=%s&count=%d", url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		p.log.Warnf("Bing keyword search request error: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warnf("Bing keyword search error for '%s': %v", keyword, err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return parseBingToTopics(string(body), keyword, limit)
}

func (p *KeywordSearchPlugin) searchBaidu(ctx context.Context, keyword string, limit int) []*biz.RawTopic {
	query := fmt.Sprintf("%s AI 创业 机会", keyword)
	searchURL := fmt.Sprintf("https://www.baidu.com/s?wd=%s&rn=%d", url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		p.log.Warnf("Baidu keyword search request error: %v", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warnf("Baidu keyword search error for '%s': %v", keyword, err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return parseBaiduToTopics(string(body), keyword, limit)
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

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
