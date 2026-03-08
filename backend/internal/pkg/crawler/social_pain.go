package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.CrawlerPluginInterface = (*SocialPainPlugin)(nil)

// ── 痛点/吐槽关键词（用于从通用内容中筛选痛点话题）──

var painKeywords = []string{
	"痛点", "吐槽", "差评", "难用", "坑", "不好用", "退款", "失望",
	"bug", "问题", "槽点", "不满", "投诉", "体验差", "垃圾", "骗",
	"frustrat", "pain", "problem", "issue", "hate", "terrible",
	"broken", "waste", "disappoint", "annoying", "useless",
}

func isPainRelated(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range painKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// SocialPainPlugin 社交平台痛点抓取（HN Algolia + 百度热搜痛点 + Bing RSS降级）
type SocialPainPlugin struct {
	client         *http.Client
	log            *log.Helper
	getConstraints func() []string
}

func NewSocialPainPlugin(logger log.Logger, getConstraints func() []string) *SocialPainPlugin {
	return &SocialPainPlugin{
		client:         newCrawlerHTTPClient(15 * time.Second),
		log:            log.NewHelper(logger),
		getConstraints: getConstraints,
	}
}

func (p *SocialPainPlugin) Name() string { return "social_pain" }
func (p *SocialPainPlugin) Label() string {
	return "社交平台痛点抓取（HN讨论/百度热搜/Bing搜索）"
}

func (p *SocialPainPlugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	if limit <= 0 {
		limit = 10
	}

	perSource := limit / 3
	if perSource < 2 {
		perSource = 2
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

	// 来源 1: HN Algolia — 英文 AI/科技痛点讨论
	addTopics(p.fetchHNPain(ctx, perSource))

	// 来源 2: 百度热搜 — 中文 AI/科技痛点热点
	addTopics(p.fetchBaiduPain(ctx, perSource))

	// 来源 3: 小红书 — 直接抓取 explore 页面推荐笔记 + 百度移动端搜索回退
	if len(allTopics) < limit {
		remaining := limit - len(allTopics)
		addTopics(p.fetchXHSPain(ctx, remaining))
	}

	p.log.Infof("[SocialPain] Fetched %d social pain topics", len(allTopics))
	return allTopics, nil
}

// ── 来源 1: HN Algolia Search ──

type hnAlgoliaResp struct {
	Hits []struct {
		Title    string `json:"title"`
		URL      string `json:"url"`
		ObjectID string `json:"objectID"`
		Points   int    `json:"points"`
		Author   string `json:"author"`
	} `json:"hits"`
}

func (p *SocialPainPlugin) fetchHNPain(ctx context.Context, limit int) []*biz.RawTopic {
	queries := []string{
		"AI tool frustration problem",
		"AI product pain point startup",
		"LLM limitation issue workflow",
	}

	var topics []*biz.RawTopic
	for _, q := range queries {
		if len(topics) >= limit {
			break
		}
		apiURL := fmt.Sprintf(
			"https://hn.algolia.com/api/v1/search?query=%s&tags=story&hitsPerPage=%d",
			url.QueryEscape(q), limit,
		)
		body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
			AcceptLanguage: "en-US,en;q=0.9",
			MaxRetries:     1,
			DetectAntiBot:  false,
		})
		if err != nil {
			p.log.Warnf("[SocialPain] HN Algolia error for '%s': %v", q, err)
			continue
		}

		var resp hnAlgoliaResp
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			p.log.Warnf("[SocialPain] HN Algolia parse error: %v", err)
			continue
		}

		for _, hit := range resp.Hits {
			if len(topics) >= limit {
				break
			}
			title := strings.TrimSpace(hit.Title)
			if title == "" {
				continue
			}
			sourceURL := hit.URL
			if sourceURL == "" {
				sourceURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
			}
			topics = append(topics, &biz.RawTopic{
				Title:      title,
				URL:        sourceURL,
				Source:     "hn_pain",
				Popularity: hit.Points,
				Snippet:    fmt.Sprintf("[HN痛点] score: %d, by: %s", hit.Points, hit.Author),
			})
		}
	}

	if len(topics) == 0 {
		p.log.Warnf("[SocialPain] HN Algolia got 0, fallback to Bing RSS")
		return p.searchBingRSS(ctx, "AI tool frustration pain point site:news.ycombinator.com", limit)
	}

	p.log.Infof("[SocialPain] HN Algolia: %d pain topics", len(topics))
	return topics
}

// ── 来源 2: 百度热搜痛点过滤 ──

func (p *SocialPainPlugin) fetchBaiduPain(ctx context.Context, limit int) []*biz.RawTopic {
	apiURL := "https://top.baidu.com/api/board?platform=wise&tab=realtime"
	body, err := fetchHTMLWithRetry(ctx, p.client, apiURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://top.baidu.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[SocialPain] Baidu hot API error: %v, fallback to Bing RSS", err)
		return p.searchBingRSS(ctx, "AI 工具 痛点 吐槽 问题", limit)
	}

	var resp baiduHotBoardResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		p.log.Warnf("[SocialPain] Baidu hot parse error: %v", err)
		return p.searchBingRSS(ctx, "AI 工具 痛点 吐槽 问题", limit)
	}

	// 展开嵌套结构
	type hotItem struct {
		Word   string
		Desc   string
		URL    string
		HotTag string
		Index  int
	}
	var allItems []hotItem
	for _, card := range resp.Data.Cards {
		for _, outer := range card.Content {
			for _, item := range outer.Content {
				allItems = append(allItems, hotItem{
					Word: item.Word, Desc: item.Desc, URL: item.URL, HotTag: item.HotTag, Index: item.Index,
				})
			}
		}
	}

	// 不限领域筛选：通用生活痛点同样有价值（后续由 AI 跨域分析判断是否有 AI 赋能可能性）
	var topics []*biz.RawTopic
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
		pop := 500
		if item.HotTag == "3" {
			pop = 1000
		} else if item.HotTag == "2" {
			pop = 800
		} else if item.HotTag == "1" {
			pop = 600
		} else if item.Index > 0 {
			pop = max(100, 500-item.Index*10)
		}
		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        sourceURL,
			Source:     "baidu_social_pain",
			Popularity: pop,
			Snippet:    fmt.Sprintf("[百度热搜痛点] %s", truncateStr(desc, 220)),
		})
	}

	if len(topics) == 0 {
		p.log.Warnf("[SocialPain] Baidu hot got 0 topics, fallback to Bing RSS")
		return p.searchBingRSS(ctx, "生活 痛点 吐槽 问题 体验差", limit)
	}

	p.log.Infof("[SocialPain] Baidu hot: %d topics", len(topics))
	return topics
}

// ── 来源 3: 小红书 explore 页面抓取 + 百度移动端搜索回退 ──

var xhsInitialStateRe = regexp.MustCompile(`__INITIAL_STATE__\s*=\s*(\{.+?)\s*</script>`)
var xhsNoteBlockRe = regexp.MustCompile(`"id"\s*:\s*"([a-f0-9]{24})"\s*,\s*"modelType"\s*:\s*"note"\s*,\s*"noteCard"`)
var xhsDisplayTitleRe = regexp.MustCompile(`"displayTitle"\s*:\s*"([^"]*)"`)
var xhsLikedCountRe = regexp.MustCompile(`"likedCount"\s*:\s*"([^"]*)"`)
var xhsNicknameRe = regexp.MustCompile(`"nickname"\s*:\s*"([^"]*)"`)

func (p *SocialPainPlugin) fetchXHSPain(ctx context.Context, limit int) []*biz.RawTopic {
	topics := p.fetchXHSExplore(ctx, limit)
	if len(topics) > 0 {
		return topics
	}
	p.log.Warnf("[SocialPain] XHS explore got 0, fallback to Baidu mobile search")
	return p.fetchBaiduMobileXHS(ctx, limit)
}

func (p *SocialPainPlugin) fetchXHSExplore(ctx context.Context, limit int) []*biz.RawTopic {
	body, err := fetchHTMLWithRetry(ctx, p.client, "https://www.xiaohongshu.com/explore", crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://www.xiaohongshu.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[SocialPain] XHS explore fetch error: %v", err)
		return nil
	}

	// 提取 __INITIAL_STATE__ JSON
	m := xhsInitialStateRe.FindStringSubmatch(body)
	if len(m) < 2 {
		p.log.Warnf("[SocialPain] XHS explore: __INITIAL_STATE__ not found")
		return nil
	}
	stateJSON := strings.ReplaceAll(m[1], "undefined", "null")

	// 定位所有 "id":"<24hex>","modelType":"note","noteCard" 块
	blocks := xhsNoteBlockRe.FindAllStringSubmatchIndex(stateJSON, -1)
	p.log.Infof("[SocialPain] XHS explore found %d note blocks", len(blocks))

	type noteEntry struct {
		noteID    string
		title     string
		nickname  string
		likedCount string
	}
	var entries []noteEntry

	for _, loc := range blocks {
		// loc[2]:loc[3] = noteId 捕获组
		noteID := stateJSON[loc[2]:loc[3]]
		// 从 noteCard 开始向后取 2000 字符提取字段
		start := loc[1]
		end := start + 2000
		if end > len(stateJSON) {
			end = len(stateJSON)
		}
		chunk := stateJSON[start:end]

		titleMatch := xhsDisplayTitleRe.FindStringSubmatch(chunk)
		if len(titleMatch) < 2 || titleMatch[1] == "" {
			continue
		}

		entry := noteEntry{noteID: noteID, title: titleMatch[1]}

		if nm := xhsNicknameRe.FindStringSubmatch(chunk); len(nm) >= 2 {
			entry.nickname = nm[1]
		}
		if lm := xhsLikedCountRe.FindStringSubmatch(chunk); len(lm) >= 2 {
			entry.likedCount = lm[1]
		}
		entries = append(entries, entry)
	}

	p.log.Infof("[SocialPain] XHS explore parsed %d notes with titles", len(entries))

	var topics []*biz.RawTopic
	for _, e := range entries {
		if len(topics) >= limit {
			break
		}
		title := strings.TrimSpace(e.title)
		if title == "" {
			continue
		}
		noteURL := fmt.Sprintf("https://www.xiaohongshu.com/explore/%s", e.noteID)
		pop := 100
		if e.likedCount != "" {
			if liked := parseXHSCount(e.likedCount); liked > 0 {
				pop = liked
			}
		}
		snippet := fmt.Sprintf("[小红书] %s", truncateStr(title, 200))
		if e.nickname != "" {
			snippet = fmt.Sprintf("[小红书] by %s: %s", e.nickname, truncateStr(title, 180))
		}
		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        noteURL,
			Source:     "xhs_explore",
			Popularity: pop,
			Snippet:    snippet,
		})
	}
	return topics
}

// parseXHSCount 解析小红书的计数字符串 "1.2万" -> 12000, "532" -> 532
func parseXHSCount(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	multiplier := 1
	if strings.HasSuffix(s, "万") {
		multiplier = 10000
		s = strings.TrimSuffix(s, "万")
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return 0
	}
	return int(f * float64(multiplier))
}

// fetchBaiduMobileXHS 百度移动端搜索小红书 AI 痛点内容（回退方案）
func (p *SocialPainPlugin) fetchBaiduMobileXHS(ctx context.Context, limit int) []*biz.RawTopic {
	query := "site:xiaohongshu.com AI 工具 产品 体验 测评"
	searchURL := fmt.Sprintf("https://m.baidu.com/s?word=%s", url.QueryEscape(query))
	body, err := fetchHTMLWithRetry(ctx, p.client, searchURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9",
		Referer:        "https://m.baidu.com/",
		MaxRetries:     2,
		DetectAntiBot:  true,
		CustomUA:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
	})
	if err != nil {
		p.log.Warnf("[SocialPain] Baidu mobile XHS error: %v, fallback to Bing RSS", err)
		return p.searchBingRSS(ctx, "小红书 AI 工具 产品 痛点 体验", limit)
	}
	topics := parseBaiduMobileToTopics(body, limit)
	if len(topics) == 0 {
		p.log.Warnf("[SocialPain] Baidu mobile XHS got 0, fallback to Bing RSS")
		return p.searchBingRSS(ctx, "小红书 AI 工具 产品 痛点 体验", limit)
	}
	for _, t := range topics {
		t.Source = "xhs_baidu"
		t.Snippet = fmt.Sprintf("[小红书via百度] %s", truncateStr(t.Snippet, 200))
	}
	p.log.Infof("[SocialPain] Baidu mobile XHS: %d topics", len(topics))
	return topics
}

// parseBaiduMobileToTopics 解析百度移动版搜索结果
func parseBaiduMobileToTopics(html string, limit int) []*biz.RawTopic {
	var topics []*biz.RawTopic
	// 百度移动版搜索结果在 <div class="c-result"> 或 <div class="result"> 内
	titleRe := regexp.MustCompile(`<a[^>]*class="[^"]*c-title-text[^"]*"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	matches := titleRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		// 尝试更宽松的匹配
		titleRe = regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>\s*<span[^>]*class="[^"]*title[^"]*"[^>]*>(.*?)</span>`)
		matches = titleRe.FindAllStringSubmatch(html, -1)
	}
	for _, m := range matches {
		if len(topics) >= limit {
			break
		}
		if len(m) < 3 {
			continue
		}
		link := strings.TrimSpace(m[1])
		title := strings.TrimSpace(stripHTMLTags(m[2]))
		if title == "" || link == "" {
			continue
		}
		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        link,
			Popularity: 100,
			Snippet:    title,
		})
	}
	return topics
}

// stripHTMLTags 移除 HTML 标签
func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

func (p *SocialPainPlugin) searchBingRSS(ctx context.Context, query string, limit int) []*biz.RawTopic {
	rssURL := fmt.Sprintf("https://www.bing.com/search?q=%s&format=rss&count=%d", url.QueryEscape(query), limit)
	body, err := fetchHTMLWithRetry(ctx, p.client, rssURL, crawlerFetchOptions{
		AcceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
		Referer:        "https://www.bing.com/",
		MaxRetries:     1,
		DetectAntiBot:  false,
	})
	if err != nil {
		p.log.Warnf("[SocialPain] Bing RSS error for '%s': %v", query, err)
		return nil
	}
	return parseBingRSSItemsToTopics(body, query, limit)
}
