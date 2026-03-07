package biz

import (
	"context"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeDiscoveryRepo struct {
	settings       map[string]string
	plugins        []*CrawlerPlugin
	hashes         map[string]bool
	savedTopics    []*DiscoveredTopic
	sourceFeedback []*SourceFeedbackStat
	feedbackErr    error
}

func (r *fakeDiscoveryRepo) GetSetting(ctx context.Context, key string) (string, error) {
	if r.settings == nil {
		return "", errors.New("not found")
	}
	v, ok := r.settings[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (r *fakeDiscoveryRepo) SetSetting(ctx context.Context, key, value string) error { return nil }
func (r *fakeDiscoveryRepo) GetAllSettings(ctx context.Context) (map[string]string, error) {
	if r.settings == nil {
		return map[string]string{}, nil
	}
	return r.settings, nil
}
func (r *fakeDiscoveryRepo) ListKeywords(ctx context.Context) ([]*SearchKeyword, error) { return nil, nil }
func (r *fakeDiscoveryRepo) CreateKeyword(ctx context.Context, keyword string) (*SearchKeyword, error) {
	return nil, nil
}
func (r *fakeDiscoveryRepo) UpdateKeyword(ctx context.Context, id int64, keyword string, enabled bool) error {
	return nil
}
func (r *fakeDiscoveryRepo) DeleteKeyword(ctx context.Context, id int64) error { return nil }
func (r *fakeDiscoveryRepo) ListPlugins(ctx context.Context) ([]*CrawlerPlugin, error) {
	return r.plugins, nil
}
func (r *fakeDiscoveryRepo) GetPlugin(ctx context.Context, name string) (*CrawlerPlugin, error) {
	for _, plugin := range r.plugins {
		if plugin.Name == name {
			return plugin, nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeDiscoveryRepo) UpsertPlugin(ctx context.Context, plugin *CrawlerPlugin) error {
	r.plugins = append(r.plugins, plugin)
	return nil
}
func (r *fakeDiscoveryRepo) TogglePlugin(ctx context.Context, name string, enabled bool) error { return nil }
func (r *fakeDiscoveryRepo) ListTags(ctx context.Context) ([]*DiscoveryTag, error)             { return nil, nil }
func (r *fakeDiscoveryRepo) ListTagsByCategory(ctx context.Context, category string) ([]*DiscoveryTag, error) {
	return nil, nil
}
func (r *fakeDiscoveryRepo) CreateTag(ctx context.Context, tag string, category string) (*DiscoveryTag, error) {
	return nil, nil
}
func (r *fakeDiscoveryRepo) UpdateTag(ctx context.Context, id int64, tag string, category string, enabled bool) error {
	return nil
}
func (r *fakeDiscoveryRepo) DeleteTag(ctx context.Context, id int64) error { return nil }
func (r *fakeDiscoveryRepo) SaveTopics(ctx context.Context, topics []*DiscoveredTopic) error {
	r.savedTopics = append(r.savedTopics, topics...)
	for index := range topics {
		topics[index].ID = int64(index + 1)
	}
	return nil
}
func (r *fakeDiscoveryRepo) ListTopics(ctx context.Context, status string, page, pageSize int) ([]*DiscoveredTopic, int, error) {
	return nil, 0, nil
}
func (r *fakeDiscoveryRepo) ListTopicSourceStats(ctx context.Context, days int) ([]*TopicSourceStat, error) {
	return nil, nil
}
func (r *fakeDiscoveryRepo) ListSourceFeedbackStats(ctx context.Context, days int) ([]*SourceFeedbackStat, error) {
	if r.feedbackErr != nil {
		return nil, r.feedbackErr
	}
	return r.sourceFeedback, nil
}
func (r *fakeDiscoveryRepo) GetTopic(ctx context.Context, id int64) (*DiscoveredTopic, error) { return nil, nil }
func (r *fakeDiscoveryRepo) UpdateTopicStatus(ctx context.Context, id int64, status string) error {
	return nil
}
func (r *fakeDiscoveryRepo) UpdateTopicAnalysis(ctx context.Context, id int64, recommendation string, score float64) error {
	return nil
}
func (r *fakeDiscoveryRepo) SetTopicIdeaID(ctx context.Context, id int64, ideaID int64) error { return nil }
func (r *fakeDiscoveryRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	if r.hashes == nil {
		return false, nil
	}
	return r.hashes[hash], nil
}

type fakeCrawlerPlugin struct {
	name   string
	label  string
	topics []*RawTopic
}

func (p *fakeCrawlerPlugin) Name() string  { return p.name }
func (p *fakeCrawlerPlugin) Label() string { return p.label }
func (p *fakeCrawlerPlugin) Fetch(ctx context.Context, limit int) ([]*RawTopic, error) {
	if len(p.topics) <= limit {
		return p.topics, nil
	}
	return p.topics[:limit], nil
}

func TestBuildSourceFeedbackFactors(t *testing.T) {
	repo := &fakeDiscoveryRepo{sourceFeedback: []*SourceFeedbackStat{
		{Source: "alpha", SampleSize: 5, SuccessRate: 1.0},
		{Source: "beta", SampleSize: 5, SuccessRate: 0.0},
		{Source: "gamma", SampleSize: 4, SuccessRate: 1.0},
		{Source: "delta", SampleSize: 10, SuccessRate: 0.5},
	}}
	uc := NewDiscoveryUsecase(repo, nil, nil, log.NewStdLogger(io.Discard))

	factors := uc.buildSourceFeedbackFactors(context.Background())

	if len(factors) != 3 {
		t.Fatalf("expected 3 factors, got %d", len(factors))
	}
	if math.Abs(factors["alpha"]-1.15) > 0.0001 {
		t.Fatalf("alpha factor mismatch: %.4f", factors["alpha"])
	}
	if math.Abs(factors["beta"]-0.85) > 0.0001 {
		t.Fatalf("beta factor mismatch: %.4f", factors["beta"])
	}
	if _, exists := factors["gamma"]; exists {
		t.Fatalf("gamma should be skipped because sample < %d", sourceFeedbackMinSample)
	}
	if math.Abs(factors["delta"]-1.0) > 0.0001 {
		t.Fatalf("delta factor mismatch: %.4f", factors["delta"])
	}
}

func TestApplySourceFeedbackFactor(t *testing.T) {
	cases := []struct {
		name     string
		score    float64
		factor   float64
		expected float64
	}{
		{name: "normal boost", score: 8.0, factor: 1.10, expected: 8.8},
		{name: "clamp top", score: 9.5, factor: 1.20, expected: 10.0},
		{name: "fallback no factor", score: 8.0, factor: 0, expected: 8.0},
		{name: "down weight", score: 8.0, factor: 0.85, expected: 6.8},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := applySourceFeedbackFactor(testCase.score, testCase.factor)
			if math.Abs(actual-testCase.expected) > 0.0001 {
				t.Fatalf("expected %.4f, got %.4f", testCase.expected, actual)
			}
		})
	}
}

func TestFetchAndDedupApplySourceFeedback(t *testing.T) {
	rawTopic := &RawTopic{
		Title:      "AI 自动化客服流程优化",
		URL:        "https://example.com/topic",
		Source:     "unit_source",
		Popularity: 300,
		Replies:    30,
		Snippet:    "企业反馈效率低、重复劳动重，愿意为降本增效付费",
	}

	repo := &fakeDiscoveryRepo{
		settings: map[string]string{
			"topics_per_source": "10",
		},
		plugins: []*CrawlerPlugin{
			{Name: "unit_source", Label: "Unit Source", Enabled: true},
		},
		sourceFeedback: []*SourceFeedbackStat{
			{Source: "unit_source", SampleSize: 10, SuccessRate: 1.0},
		},
	}

	uc := NewDiscoveryUsecase(repo, nil, nil, log.NewStdLogger(io.Discard))
	uc.plugins["unit_source"] = &fakeCrawlerPlugin{
		name:   "unit_source",
		label:  "Unit Source",
		topics: []*RawTopic{rawTopic},
	}

	topics, err := uc.fetchAndDedup(context.Background())
	if err != nil {
		t.Fatalf("fetchAndDedup error: %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(topics))
	}

	_, _, _, _, _, ruleTotal := calculateRuleScores(rawTopic)
	expected := applySourceFeedbackFactor(ruleTotal, 1.15)
	if math.Abs(topics[0].RecommendScore-expected) > 0.0001 {
		t.Fatalf("recommend score mismatch, expected %.4f, got %.4f", expected, topics[0].RecommendScore)
	}
}
