package data

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListSourceFeedbackStats(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := autoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := &discoveryRepo{data: &Data{db: db}}
	now := time.Now()
	oldTime := now.AddDate(0, 0, -40)

	ideas := []*IdeaModel{
		{ID: 1, Topic: "a", Status: "graduated"},
		{ID: 2, Topic: "b", Status: "failed"},
		{ID: 3, Topic: "c", Status: "promising"},
		{ID: 4, Topic: "d", Status: "graduated"},
	}
	for _, idea := range ideas {
		if err := db.Create(idea).Error; err != nil {
			t.Fatalf("seed idea: %v", err)
		}
	}

	topics := []*DiscoveredTopicModel{
		{Source: "source_a", Title: "t1", IdeaID: 1, DiscoveredAt: now},
		{Source: "source_a", Title: "t2", IdeaID: 2, DiscoveredAt: now},
		{Source: "source_b", Title: "t3", IdeaID: 3, DiscoveredAt: now},
		{Source: "source_b", Title: "t4", IdeaID: 0, DiscoveredAt: now}, // 未入辩论，不计入样本
		{Source: "source_c", Title: "t5", IdeaID: 4, DiscoveredAt: oldTime},
	}
	for _, topic := range topics {
		if err := db.Create(topic).Error; err != nil {
			t.Fatalf("seed topic: %v", err)
		}
	}

	stats, err := repo.ListSourceFeedbackStats(context.Background(), 30)
	if err != nil {
		t.Fatalf("ListSourceFeedbackStats(30): %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 sources in 30 days, got %d", len(stats))
	}

	check := map[string]struct {
		sample  int64
		rate    float64
	}{
		"source_a": {sample: 2, rate: 0.5},
		"source_b": {sample: 1, rate: 1.0},
	}
	for _, stat := range stats {
		expected, ok := check[stat.Source]
		if !ok {
			t.Fatalf("unexpected source %s", stat.Source)
		}
		if stat.SampleSize != expected.sample {
			t.Fatalf("%s sample mismatch, expected %d got %d", stat.Source, expected.sample, stat.SampleSize)
		}
		if diff := stat.SuccessRate - expected.rate; diff > 0.0001 || diff < -0.0001 {
			t.Fatalf("%s success rate mismatch, expected %.2f got %.2f", stat.Source, expected.rate, stat.SuccessRate)
		}
	}

	statsAll, err := repo.ListSourceFeedbackStats(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListSourceFeedbackStats(0): %v", err)
	}
	if len(statsAll) != 3 {
		t.Fatalf("expected 3 sources without day filter, got %d", len(statsAll))
	}
}
