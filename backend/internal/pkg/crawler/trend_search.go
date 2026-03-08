package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// ── AI/科技相关关键词过滤（用于从通用热搜中筛选科技话题）──

var trendAIKeywords = []string{
	"ai", "人工智能", "大模型", "chatgpt", "gpt", "llm", "agent",
	"机器人", "自动驾驶", "芯片", "半导体", "算力", "数据",
	"科技", "互联网", "软件", "创业", "融资", "开源",
	"deepseek", "openai", "google", "microsoft", "apple", "meta",
	"算法", "模型", "智能", "数字", "云计算", "区块链",
	"saas", "startup", "developer", "machine learning", "deep learning",
	"neural", "transformer", "diffusion", "robotics", "autonomous",
	// 扩展：中文科技热词、公司名、产品名
	"华为", "小米", "字节", "腾讯", "阿里", "百度", "比亚迪",
	"特斯拉", "英伟达", "nvidia", "tesla", "spacex",
	"手机", "电动", "新能源", "量子", "卫星", "火箭", "航天",
	"编程", "程序员", "代码", "开发者", "技术", "工程师",
	"网络安全", "黑客", "漏洞", "隐私", "加密",
	"电商", "直播", "短视频", "平台", "app",
	"5g", "6g", "物联网", "iot", "vr", "ar", "元宇宙",
}

// TrendSearchPlugin 热点趋势抓取插件 — 从百度热搜/GitHub Trending/Hacker News 获取真实热点
type TrendSearchPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string // 获取约束标签（保持接口兼容）
}

var _ biz.CrawlerPluginInterface = (*TrendSearchPlugin)(nil)

func NewTrendSearchPlugin(
	logger log.Logger,
	getConstraints func() []string,
	_ func() (cn []string, en []string), // 保持构造函数签名兼容，不再使用
) *TrendSearchPlugin {
	return &TrendSearchPlugin{
		client:         newCrawlerHTTPClient(15 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *TrendSearchPlugin) Name() string { return "trend_search" }
func (p *TrendSearchPlugin) Label() string {
	return "热点趋势抓取（百度热搜/GitHub Trending/Hacker News）"
}

func (p *TrendSearchPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	if limit <= 0 {
		limit = 10
	}

	perSource := limit / 3
	if perSource < 3 {
		perSource = 3
	}

	var (
		mu        sync.Mutex
		allTopics []*biz.RawTopic
		wg        sync.WaitGroup
	)

	// 并发抓取三个来源
	wg.Add(3)

	go func() {
		defer wg.Done()
		topics := p.fetchBaiduHotSearch(ctx, perSource)
		mu.Lock()
		allTopics = append(allTopics, topics...)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		topics := p.fetchGitHubTrending(ctx, perSource)
		mu.Lock()
		allTopics = append(allTopics, topics...)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		topics := p.fetchHackerNews(ctx, perSource)
		mu.Lock()
		allTopics = append(allTopics, topics...)
		mu.Unlock()
	}()

	wg.Wait()

	if len(allTopics) > limit {
		allTopics = allTopics[:limit]
	}

	p.log.Infof("[TrendSearch] Fetched %d real trending topics (baidu_hot + github_trending + hackernews_top)", len(allTopics))
	return allTopics, nil
}

// ── 百度热搜 ──

type baiduHotBoardResp struct {
	Data struct {
		Cards []struct {
			Content []struct {
				Content []struct {
					Word   string `json:"word"`
					Desc   string `json:"desc"`
					URL    string `json:"url"`
					HotTag string `json:"hotTag"` // 0=普通, 1=新, 2=沸, 3=热
					Index  int    `json:"index"`
				} `json:"content"`
			} `json:"content"`
		} `json:"cards"`
	} `json:"data"`
}

func (p *TrendSearchPlugin) fetchBaiduHotSearch(ctx context.Context, limit int) []*biz.RawTopic {
	apiURL := "https://top.baidu.com/api/board?platform=wise&tab=realtime"
	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://top.baidu.com/",
		MaxRetries:     2,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[TrendSearch] Baidu hot search API error: %v", err)
		return p.fetchBaiduHotSearchFallback(ctx, limit)
	}

	var resp baiduHotBoardResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		p.log.Warnf("[TrendSearch] Baidu hot search JSON parse error: %v", err)
		return p.fetchBaiduHotSearchFallback(ctx, limit)
	}

	// 展开嵌套结构：cards[].content[].content[] → flat items
	type baiduHotItem struct {
		Word   string
		Desc   string
		URL    string
		HotTag string
		Index  int
	}
	var allItems []baiduHotItem
	for _, card := range resp.Data.Cards {
		for _, outer := range card.Content {
			for _, item := range outer.Content {
				allItems = append(allItems, baiduHotItem{
					Word: item.Word, Desc: item.Desc, URL: item.URL, HotTag: item.HotTag, Index: item.Index,
				})
			}
		}
	}

	// hotTag 映射为热度指标：3=热(1000), 2=沸(800), 1=新(600), 0=普通(按排名)
	hotTagScore := func(tag string, index int) int {
		switch tag {
		case "3":
			return 1000
		case "2":
			return 800
		case "1":
			return 600
		default:
			// 按排名位置给分，排名越前越高
			if index <= 0 {
				return 500
			}
			return max(100, 500-index*10)
		}
	}

	var topics []*biz.RawTopic
	for _, item := range allItems {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(item.Word)
		if title == "" {
			continue
		}
		// 筛选 AI/科技相关话题
		if !isTechRelated(title + " " + item.Desc) {
			continue
		}
		sourceURL := item.URL
		if sourceURL == "" {
			sourceURL = fmt.Sprintf("https://www.baidu.com/s?wd=%s", url.QueryEscape(title))
		}
		desc := item.Desc
		if desc == "" {
			desc = title
		}
		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        sourceURL,
			Source:     "baidu_hot",
			Popularity: hotTagScore(item.HotTag, item.Index),
			Snippet:    fmt.Sprintf("[百度热搜] %s", truncateStr(desc, 220)),
		})
	}

	// 如果科技过滤后为 0，取热搜前 N 条作为保底（热搜本身有发现价值）
	if len(topics) == 0 && len(allItems) > 0 {
		p.log.Warnf("[TrendSearch] Baidu hot search got 0 AI/tech topics, taking top %d from all hot topics", limit)
		for _, item := range allItems {
			if len(topics) >= limit {
				break
			}
			title := strings.TrimSpace(item.Word)
			if title == "" {
				continue
			}
			sourceURL := item.URL
			if sourceURL == "" {
				sourceURL = fmt.Sprintf("https://www.baidu.com/s?wd=%s", url.QueryEscape(title))
			}
			desc := item.Desc
			if desc == "" {
				desc = title
			}
			topics = append(topics, &biz.RawTopic{
				Title:      title,
				URL:        sourceURL,
				Source:     "baidu_hot",
				Popularity: hotTagScore(item.HotTag, item.Index),
				Snippet:    fmt.Sprintf("[百度热搜] %s", truncateStr(desc, 220)),
			})
		}
	}

	if len(topics) == 0 {
		p.log.Warnf("[TrendSearch] Baidu hot search API returned empty data, trying fallback")
		return p.fetchBaiduHotSearchFallback(ctx, limit)
	}

	p.log.Infof("[TrendSearch] Baidu hot search: %d AI/tech topics", len(topics))
	return topics
}

// fetchBaiduHotSearchFallback 百度热搜降级：用 Bing 搜索"百度热搜 AI"
func (p *TrendSearchPlugin) fetchBaiduHotSearchFallback(ctx context.Context, limit int) []*biz.RawTopic {
	query := "AI 科技 热点 最新动态"
	searchURL := fmt.Sprintf(
		"https://www.bing.com/search?q=%s&format=rss&count=%d&setlang=zh-cn",
		url.QueryEscape(query), limit,
	)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[TrendSearch] Baidu hot fallback (Bing RSS) error: %v", err)
		return nil
	}
	topics := parseBingRSSItemsToTopics(body, query, limit)
	for _, t := range topics {
		t.Source = "baidu_hot"
		t.Snippet = fmt.Sprintf("[热点趋势] %s", stripKeywordPrefix(t.Snippet))
	}
	return topics
}

// ── GitHub Trending ──

func (p *TrendSearchPlugin) fetchGitHubTrending(ctx context.Context, limit int) []*biz.RawTopic {
	trendingURL := "https://github.com/trending?since=daily"
	body, err := fetchHTMLWithRetry(ctx, p.client, trendingURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://github.com/",
		MaxRetries:     2,
		DetectAntiBot:  true,
	})
	if err != nil {
		p.log.Warnf("[TrendSearch] GitHub Trending fetch error: %v", err)
		return p.fetchGitHubTrendingFallback(ctx, limit)
	}

	topics := p.parseGitHubTrending(body, limit)
	if len(topics) == 0 {
		p.log.Warnf("[TrendSearch] GitHub Trending parsed 0 repos, trying fallback")
		return p.fetchGitHubTrendingFallback(ctx, limit)
	}

	p.log.Infof("[TrendSearch] GitHub Trending: %d repos", len(topics))
	return topics
}

func (p *TrendSearchPlugin) parseGitHubTrending(html string, limit int) []*biz.RawTopic {
	var topics []*biz.RawTopic

	// GitHub Trending 每个仓库在 <article class="Box-row"> 中
	articleRe := regexp.MustCompile(`(?s)<article class="Box-row">(.*?)</article>`)
	// 仓库名：<h2 ...><a href="/owner/repo">
	repoRe := regexp.MustCompile(`(?s)<h2[^>]*>.*?<a[^>]*href="(/[^"]+)"[^>]*>`)
	// 描述：<p class="col-9 ...">描述</p>
	descRe := regexp.MustCompile(`(?s)<p[^>]*class="[^"]*col-9[^"]*"[^>]*>(.*?)</p>`)
	// 星标数：<span ...class="d-inline-block float-sm-right">... ★ 数字
	starsRe := regexp.MustCompile(`(?s)(\d[\d,]*)\s*stars\s*today`)

	articles := articleRe.FindAllStringSubmatch(html, limit*2)
	for _, article := range articles {
		if len(topics) >= limit {
			break
		}
		block := article[1]

		repoMatch := repoRe.FindStringSubmatch(block)
		if repoMatch == nil {
			continue
		}
		repoPath := strings.TrimSpace(repoMatch[1])
		repoName := strings.TrimPrefix(repoPath, "/")
		if repoName == "" {
			continue
		}

		desc := ""
		descMatch := descRe.FindStringSubmatch(block)
		if descMatch != nil {
			desc = strings.TrimSpace(stripHTML(descMatch[1]))
		}

		stars := 0
		starsMatch := starsRe.FindStringSubmatch(block)
		if starsMatch != nil {
			starsStr := strings.ReplaceAll(starsMatch[1], ",", "")
			stars, _ = strconv.Atoi(starsStr)
		}

		topics = append(topics, &biz.RawTopic{
			Title:      repoName,
			URL:        "https://github.com" + repoPath,
			Source:     "github_trending",
			Popularity: stars,
			Snippet:    fmt.Sprintf("[GitHub Trending] %s", truncateStr(desc, 220)),
		})
	}

	return topics
}

// fetchGitHubTrendingFallback GitHub Trending 降级：用 Bing RSS 搜索
func (p *TrendSearchPlugin) fetchGitHubTrendingFallback(ctx context.Context, limit int) []*biz.RawTopic {
	query := "site:github.com AI trending open source new repo"
	searchURL := fmt.Sprintf(
		"https://www.bing.com/search?q=%s&format=rss&count=%d",
		url.QueryEscape(query), limit,
	)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://www.bing.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[TrendSearch] GitHub Trending fallback (Bing RSS) error: %v", err)
		return nil
	}
	topics := parseBingRSSItemsToTopics(body, query, limit)
	for _, t := range topics {
		t.Source = "github_trending"
		t.Snippet = fmt.Sprintf("[GitHub Trending] %s", stripKeywordPrefix(t.Snippet))
	}
	return topics
}

// ── Hacker News ──

func (p *TrendSearchPlugin) fetchHackerNews(ctx context.Context, limit int) []*biz.RawTopic {
	// Hacker News Firebase API: 获取 top story IDs
	apiURL := "https://hacker-news.firebaseio.com/v0/topstories.json"
	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://news.ycombinator.com/",
		MaxRetries:     2,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[TrendSearch] Hacker News top stories API error: %v", err)
		return p.fetchHackerNewsFallback(ctx, limit)
	}

	var storyIDs []int
	if err := json.Unmarshal([]byte(body), &storyIDs); err != nil {
		p.log.Warnf("[TrendSearch] Hacker News JSON parse error: %v", err)
		return p.fetchHackerNewsFallback(ctx, limit)
	}

	// 取前 N 个 story，并发获取详情（取多一些以便过滤）
	fetchCount := limit * 4
	if fetchCount > 50 {
		fetchCount = 50
	}
	if fetchCount > len(storyIDs) {
		fetchCount = len(storyIDs)
	}

	type hnItem struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Score int    `json:"score"`
		By    string `json:"by"`
	}

	var (
		mu      sync.Mutex
		items   []hnItem
		wg      sync.WaitGroup
		semCh   = make(chan struct{}, 5) // 限制并发数
	)

	for _, id := range storyIDs[:fetchCount] {
		wg.Add(1)
		go func(storyID int) {
			defer wg.Done()
			semCh <- struct{}{}
			defer func() { <-semCh }()

			itemURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyID)
			itemBody, fetchErr := fetchHTMLWithRetry(ctx, p.client, itemURL, crawlerFetchOptions{
				MaxRetries:    0,
				DetectAntiBot: false,
			})
			if fetchErr != nil {
				return
			}
			var item hnItem
			if json.Unmarshal([]byte(itemBody), &item) != nil {
				return
			}
			if item.Title == "" {
				return
			}
			mu.Lock()
			items = append(items, item)
			mu.Unlock()
		}(id)
	}

	wg.Wait()

	// 筛选 AI/科技相关（HN 本身就是科技社区，放宽过滤）
	var topics []*biz.RawTopic
	for _, item := range items {
		if len(topics) >= limit {
			break
		}
		sourceURL := item.URL
		if sourceURL == "" {
			sourceURL = fmt.Sprintf("https://news.ycombinator.com/item?id=0")
		}
		topics = append(topics, &biz.RawTopic{
			Title:      item.Title,
			URL:        sourceURL,
			Source:     "hackernews_top",
			Popularity: item.Score,
			Snippet:    fmt.Sprintf("[Hacker News] score: %d, by: %s", item.Score, item.By),
		})
	}

	if len(topics) == 0 {
		p.log.Warnf("[TrendSearch] Hacker News got 0 topics, trying fallback")
		return p.fetchHackerNewsFallback(ctx, limit)
	}

	p.log.Infof("[TrendSearch] Hacker News: %d topics", len(topics))
	return topics
}

// fetchHackerNewsFallback Hacker News 降级：用 Bing RSS 搜索
func (p *TrendSearchPlugin) fetchHackerNewsFallback(ctx context.Context, limit int) []*biz.RawTopic {
	query := "site:news.ycombinator.com AI startup trending"
	searchURL := fmt.Sprintf(
		"https://www.bing.com/search?q=%s&format=rss&count=%d",
		url.QueryEscape(query), limit,
	)
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "en-US,en;q=0.9",
		Referer:        "https://www.bing.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[TrendSearch] Hacker News fallback (Bing RSS) error: %v", err)
		return nil
	}
	topics := parseBingRSSItemsToTopics(body, query, limit)
	for _, t := range topics {
		t.Source = "hackernews_top"
		t.Snippet = fmt.Sprintf("[Hacker News] %s", stripKeywordPrefix(t.Snippet))
	}
	return topics
}

// ── 工具函数 ──

// isTechRelated 检查标题/描述是否包含科技/AI相关关键词
func isTechRelated(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range trendAIKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func shuffleStrings(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(dst), func(i, j int) { dst[i], dst[j] = dst[j], dst[i] })
	return dst
}
