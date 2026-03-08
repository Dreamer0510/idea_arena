package crawler

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func newTestTrendPlugin() *TrendSearchPlugin {
	return NewTrendSearchPlugin(log.DefaultLogger, nil, nil)
}

// TestFetchBaiduHotSearch 测试百度热搜 API 主路径
// 验证：能从 top.baidu.com JSON API 获取实时热搜，筛选出 AI/科技相关话题
func TestFetchBaiduHotSearch(t *testing.T) {
	p := newTestTrendPlugin()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	topics := p.fetchBaiduHotSearch(ctx, 5)

	t.Logf("[BaiduHotSearch] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d title=%q source=%s url=%s popularity=%d snippet=%s",
			i+1, topic.Title, topic.Source, topic.URL, topic.Popularity, truncateStr(topic.Snippet, 80))
	}

	if len(topics) == 0 {
		t.Logf("[BaiduHotSearch] WARNING: 主路径返回 0 条，可能是当前热搜中没有 AI/科技相关话题，或百度 API 被限制")
		t.Logf("[BaiduHotSearch] 尝试降级路径...")

		fallbackTopics := p.fetchBaiduHotSearchFallback(ctx, 5)
		t.Logf("[BaiduHotSearch Fallback] got %d topics", len(fallbackTopics))
		for i, topic := range fallbackTopics {
			t.Logf("  #%d title=%q source=%s url=%s", i+1, topic.Title, topic.Source, topic.URL)
		}

		if len(fallbackTopics) == 0 {
			t.Error("[BaiduHotSearch] 主路径和降级路径都返回 0 条，需要排查")
		}
		return
	}

	// 验证数据质量
	for _, topic := range topics {
		if topic.Title == "" {
			t.Error("百度热搜话题标题为空")
		}
		if topic.Source != "baidu_hot" {
			t.Errorf("百度热搜 source 应为 baidu_hot，实际为 %s", topic.Source)
		}
		if topic.URL == "" {
			t.Error("百度热搜话题 URL 为空")
		}
	}
}

// TestFetchGitHubTrending 测试 GitHub Trending 主路径
// 验证：能从 github.com/trending 解析出每日热门仓库
func TestFetchGitHubTrending(t *testing.T) {
	p := newTestTrendPlugin()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	topics := p.fetchGitHubTrending(ctx, 5)

	t.Logf("[GitHubTrending] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d title=%q source=%s url=%s popularity=%d snippet=%s",
			i+1, topic.Title, topic.Source, topic.URL, topic.Popularity, truncateStr(topic.Snippet, 80))
	}

	if len(topics) == 0 {
		t.Logf("[GitHubTrending] WARNING: 主路径返回 0 条，可能是 GitHub 反爬或网络问题")
		t.Logf("[GitHubTrending] 尝试降级路径...")

		fallbackTopics := p.fetchGitHubTrendingFallback(ctx, 5)
		t.Logf("[GitHubTrending Fallback] got %d topics", len(fallbackTopics))
		for i, topic := range fallbackTopics {
			t.Logf("  #%d title=%q source=%s url=%s", i+1, topic.Title, topic.Source, topic.URL)
		}

		if len(fallbackTopics) == 0 {
			t.Error("[GitHubTrending] 主路径和降级路径都返回 0 条，需要排查")
		}
		return
	}

	// 验证数据质量
	for _, topic := range topics {
		if topic.Title == "" {
			t.Error("GitHub Trending 仓库名为空")
		}
		if topic.Source != "github_trending" {
			t.Errorf("GitHub Trending source 应为 github_trending，实际为 %s", topic.Source)
		}
		if topic.URL == "" || !contains(topic.URL, "github.com") {
			t.Errorf("GitHub Trending URL 不合法: %s", topic.URL)
		}
	}
}

// TestFetchHackerNews 测试 Hacker News Firebase API 主路径
// 验证：能从 HN API 获取热门科技文章
func TestFetchHackerNews(t *testing.T) {
	p := newTestTrendPlugin()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchHackerNews(ctx, 5)

	t.Logf("[HackerNews] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d title=%q source=%s url=%s popularity=%d snippet=%s",
			i+1, topic.Title, topic.Source, topic.URL, topic.Popularity, truncateStr(topic.Snippet, 80))
	}

	if len(topics) == 0 {
		t.Logf("[HackerNews] WARNING: 主路径返回 0 条，可能是 Firebase API 被限制")
		t.Logf("[HackerNews] 尝试降级路径...")

		fallbackTopics := p.fetchHackerNewsFallback(ctx, 5)
		t.Logf("[HackerNews Fallback] got %d topics", len(fallbackTopics))
		for i, topic := range fallbackTopics {
			t.Logf("  #%d title=%q source=%s url=%s", i+1, topic.Title, topic.Source, topic.URL)
		}

		if len(fallbackTopics) == 0 {
			t.Error("[HackerNews] 主路径和降级路径都返回 0 条，需要排查")
		}
		return
	}

	// 验证数据质量
	for _, topic := range topics {
		if topic.Title == "" {
			t.Error("Hacker News 标题为空")
		}
		if topic.Source != "hackernews_top" {
			t.Errorf("Hacker News source 应为 hackernews_top，实际为 %s", topic.Source)
		}
		if topic.URL == "" {
			t.Error("Hacker News URL 为空")
		}
		if topic.Popularity <= 0 {
			t.Logf("Hacker News score=%d for %q (score 为 0 可能是新帖)", topic.Popularity, topic.Title)
		}
	}
}

// TestTrendSearchPluginFetchIntegration 集成测试：完整 Fetch 流程
func TestTrendSearchPluginFetchIntegration(t *testing.T) {
	p := newTestTrendPlugin()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 10)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("[TrendSearch Integration] total=%d topics", len(topics))

	sourceCount := make(map[string]int)
	for _, topic := range topics {
		sourceCount[topic.Source]++
	}

	t.Logf("Source distribution:")
	for source, count := range sourceCount {
		t.Logf("  %s: %d", source, count)
	}

	if len(topics) == 0 {
		t.Error("Fetch 返回 0 条话题，三个来源全部失败")
	}

	// 验证至少有 2 个不同来源有数据
	if len(sourceCount) < 2 {
		t.Logf("WARNING: 只有 %d 个来源返回了数据，预期至少 2 个", len(sourceCount))
	}
}

// TestParseGitHubTrending 测试 GitHub Trending HTML 解析逻辑
func TestParseGitHubTrending(t *testing.T) {
	// 模拟 GitHub Trending 的 HTML 片段
	mockHTML := `
<article class="Box-row">
  <h2 class="h3 lh-condensed">
    <a href="/testowner/testrepo" data-hydro-click="">
      <span>testowner</span> /
      <span class="text-normal">testrepo</span>
    </a>
  </h2>
  <p class="col-9 color-fg-muted my-1 pr-4">A test repository for AI things</p>
  <div class="f6 color-fg-muted mt-2">
    <span class="d-inline-block float-sm-right">128 stars today</span>
  </div>
</article>
<article class="Box-row">
  <h2 class="h3 lh-condensed">
    <a href="/another/repo2" data-hydro-click="">
      <span>another</span> /
      <span class="text-normal">repo2</span>
    </a>
  </h2>
  <p class="col-9 color-fg-muted my-1 pr-4">Another cool project</p>
  <div class="f6 color-fg-muted mt-2">
    <span class="d-inline-block float-sm-right">42 stars today</span>
  </div>
</article>
`

	p := newTestTrendPlugin()
	topics := p.parseGitHubTrending(mockHTML, 10)

	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}

	if topics[0].Title != "testowner/testrepo" {
		t.Errorf("expected title 'testowner/testrepo', got %q", topics[0].Title)
	}
	if topics[0].URL != "https://github.com/testowner/testrepo" {
		t.Errorf("expected URL 'https://github.com/testowner/testrepo', got %q", topics[0].URL)
	}
	if topics[0].Source != "github_trending" {
		t.Errorf("expected source 'github_trending', got %q", topics[0].Source)
	}
	if topics[0].Popularity != 128 {
		t.Errorf("expected 128 stars, got %d", topics[0].Popularity)
	}

	if topics[1].Title != "another/repo2" {
		t.Errorf("expected title 'another/repo2', got %q", topics[1].Title)
	}
	if topics[1].Popularity != 42 {
		t.Errorf("expected 42 stars, got %d", topics[1].Popularity)
	}

	t.Logf("parseGitHubTrending OK: %d topics parsed correctly", len(topics))
}

// TestIsTechRelated 测试 AI/科技关键词过滤
func TestIsTechRelated(t *testing.T) {
	tests := []struct {
		text   string
		expect bool
	}{
		{"ChatGPT 发布新版本", true},
		{"OpenAI 完成新一轮融资", true},
		{"人工智能赋能制造业", true},
		{"今日天气预报", false},
		{"NBA 季后赛赛程", false},
		{"AI Agent 自动化工具", true},
		{"大模型在医疗领域的应用", true},
		{"明星八卦新闻", false},
		{"Google DeepMind 新突破", true},
		{"区块链技术应用", true},
	}

	for _, tc := range tests {
		result := isTechRelated(tc.text)
		if result != tc.expect {
			t.Errorf("isTechRelated(%q) = %v, want %v", tc.text, result, tc.expect)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
