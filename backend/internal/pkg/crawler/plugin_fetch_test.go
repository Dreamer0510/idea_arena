package crawler

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func noConstraints() []string { return nil }

// TestSocialPainPluginFetch 测试 social_pain 插件能否获取数据
func TestSocialPainPluginFetch(t *testing.T) {
	p := NewSocialPainPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 5)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("[SocialPain] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d source=%s title=%q url=%s", i+1, topic.Source, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[SocialPain] returned 0 topics — both Bing HTML and RSS failed")
	}
}

// TestAcademicFrontierPluginFetch 测试 academic_frontier 插件能否获取数据
func TestAcademicFrontierPluginFetch(t *testing.T) {
	p := NewAcademicFrontierPlugin(log.DefaultLogger, noConstraints)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topics, err := p.Fetch(ctx, 5)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("[AcademicFrontier] got %d topics", len(topics))
	for i, topic := range topics {
		t.Logf("  #%d source=%s title=%q url=%s", i+1, topic.Source, topic.Title, topic.URL)
	}

	if len(topics) == 0 {
		t.Error("[AcademicFrontier] returned 0 topics — both Bing HTML and RSS failed")
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
