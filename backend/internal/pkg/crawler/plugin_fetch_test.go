package crawler

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func noConstraints() []string { return nil }

// TestSocialPainPluginFetch 测试 social_pain 插件完整 Fetch（limit=30 展示全貌）
func TestSocialPainPluginFetch(t *testing.T) {
	p := NewSocialPainPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 30)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	// 按来源统计
	sourceCounts := make(map[string]int)
	for _, topic := range topics {
		sourceCounts[topic.Source]++
	}
	t.Logf("[SocialPain] total: %d topics", len(topics))
	for src, cnt := range sourceCounts {
		t.Logf("  source=%s count=%d", src, cnt)
	}
	t.Log("--- 详细列表 ---")
	for i, topic := range topics {
		t.Logf("  #%d [%s] pop=%d title=%q url=%s", i+1, topic.Source, topic.Popularity, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[SocialPain] returned 0 topics")
	}
}

// TestSocialPainHNAlgolia 单独测试 HN Algolia 来源
func TestSocialPainHNAlgolia(t *testing.T) {
	p := NewSocialPainPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchHNPain(ctx, 10)
	t.Logf("[HN Algolia] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d pop=%d title=%q url=%s", i+1, topic.Popularity, topic.Title, topic.URL)
	}
	if len(topics) == 0 {
		t.Error("[HN Algolia] returned 0 topics")
	}
}

// TestSocialPainBaiduHot 单独测试百度热搜来源（不限领域）
func TestSocialPainBaiduHot(t *testing.T) {
	p := NewSocialPainPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchBaiduPain(ctx, 20)
	t.Logf("[Baidu Hot] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d pop=%d title=%q", i+1, topic.Popularity, topic.Title)
	}
	if len(topics) == 0 {
		t.Error("[Baidu Hot] returned 0 topics")
	}
}

// TestSocialPainXHSExplore 单独测试小红书 explore 来源
func TestSocialPainXHSExplore(t *testing.T) {
	p := NewSocialPainPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchXHSExplore(ctx, 30)
	t.Logf("[XHS Explore] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d pop=%d title=%q url=%s snippet=%s", i+1, topic.Popularity, topic.Title, topic.URL, topic.Snippet)
	}
	if len(topics) == 0 {
		t.Error("[XHS Explore] returned 0 topics")
	}
}

// TestAcademicFrontierPluginFetch 测试 academic_frontier 完整 Fetch
func TestAcademicFrontierPluginFetch(t *testing.T) {
	p := NewAcademicFrontierPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 15)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	sourceCounts := make(map[string]int)
	for _, topic := range topics {
		sourceCounts[topic.Source]++
	}
	t.Logf("[AcademicFrontier] total: %d topics", len(topics))
	for src, cnt := range sourceCounts {
		t.Logf("  source=%s count=%d", src, cnt)
	}
	t.Log("--- 详细列表 ---")
	for i, topic := range topics {
		t.Logf("  #%d [%s] pop=%d title=%q url=%s", i+1, topic.Source, topic.Popularity, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[AcademicFrontier] returned 0 topics")
	}
}

// TestAcademicArXiv 单独测试 arXiv API
func TestAcademicArXiv(t *testing.T) {
	p := NewAcademicFrontierPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchArXiv(ctx, 5)
	t.Logf("[arXiv] got %d papers", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d title=%q url=%s snippet=%s", i+1, topic.Title, topic.URL, topic.Snippet[:min(100, len(topic.Snippet))])
	}
	if len(topics) == 0 {
		t.Error("[arXiv] returned 0 papers")
	}
}

// TestAcademicHFPapers 单独测试 HuggingFace Daily Papers
func TestAcademicHFPapers(t *testing.T) {
	p := NewAcademicFrontierPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchHFDailyPapers(ctx, 5)
	t.Logf("[HF Papers] got %d papers", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d pop=%d title=%q url=%s", i+1, topic.Popularity, topic.Title, topic.URL)
	}
	if len(topics) == 0 {
		t.Error("[HF Papers] returned 0 papers")
	}
}

// TestAcademicPWC 单独测试 PapersWithCode
func TestAcademicPWC(t *testing.T) {
	p := NewAcademicFrontierPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics := p.fetchPapersWithCode(ctx, 5)
	t.Logf("[PapersWithCode] got %d papers", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d pop=%d title=%q url=%s", i+1, topic.Popularity, topic.Title, topic.URL)
	}
	if len(topics) == 0 {
		t.Error("[PapersWithCode] returned 0 papers")
	}
}

// TestFundingSignalPluginFetch 测试 funding_signal 插件能否获取数据
func TestFundingSignalPluginFetch(t *testing.T) {
	p := NewFundingSignalPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 5)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("[FundingSignal] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d source=%s title=%q url=%s", i+1, topic.Source, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[FundingSignal] returned 0 topics — both Bing HTML and RSS failed")
	}
}

// TestPolicySignalPluginFetch 测试 policy_signal 插件能否获取数据
func TestPolicySignalPluginFetch(t *testing.T) {
	p := NewPolicySignalPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 5)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("[PolicySignal] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d source=%s title=%q url=%s", i+1, topic.Source, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[PolicySignal] returned 0 topics — both Bing HTML and RSS failed")
	}
}

// TestDemandSignalPluginFetch 测试 demand_signal 插件能否获取数据
func TestDemandSignalPluginFetch(t *testing.T) {
	p := NewDemandSignalPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 5)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("[DemandSignal] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d source=%s title=%q url=%s", i+1, topic.Source, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[DemandSignal] returned 0 topics — both Bing HTML and RSS failed")
	}
}
