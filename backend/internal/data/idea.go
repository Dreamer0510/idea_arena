package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type ideaRepo struct {
	data *Data
	log  *log.Helper
}

// NewIdeaRepo 创建 IdeaRepo 实现
func NewIdeaRepo(data *Data, logger log.Logger) biz.IdeaRepo {
	return &ideaRepo{data: data, log: log.NewHelper(logger)}
}

func (r *ideaRepo) Create(ctx context.Context, idea *biz.Idea) (*biz.Idea, error) {
	model := toBizIdeaModel(idea)
	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toIdeaBiz(model), nil
}

func (r *ideaRepo) GetByID(ctx context.Context, id int64) (*biz.Idea, error) {
	var model IdeaModel
	if err := r.data.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	return toIdeaBiz(&model), nil
}

func (r *ideaRepo) Update(ctx context.Context, idea *biz.Idea) error {
	updates := make(map[string]interface{})
	if idea.Topic != "" {
		updates["topic"] = idea.Topic
	}
	if idea.Status != "" {
		updates["status"] = idea.Status
	}
	if idea.Proposal != "" {
		updates["proposal"] = idea.Proposal
	}
	if idea.DebateLog != "" {
		updates["debate_log"] = idea.DebateLog
	}
	if idea.FinalReport != "" {
		updates["final_report"] = idea.FinalReport
	}
	if idea.ProductName != "" {
		updates["product_name"] = idea.ProductName
	}
	if idea.OneLiner != "" {
		updates["one_liner"] = idea.OneLiner
	}
	if idea.SearchData != "" {
		updates["search_data"] = idea.SearchData
	}
	if idea.JudgeRefined != "" {
		updates["judge_refined"] = idea.JudgeRefined
	}
	if idea.VoteResult != "" {
		updates["vote_result"] = idea.VoteResult
	}
	if idea.TechStack != "" {
		updates["tech_stack"] = idea.TechStack
	}
	if idea.DevPrompt != "" {
		updates["dev_prompt"] = idea.DevPrompt
	}
	if len(idea.Tags) > 0 {
		if b, err := json.Marshal(idea.Tags); err == nil {
			updates["tags"] = string(b)
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return r.data.db.WithContext(ctx).Model(&IdeaModel{}).Where("id = ?", idea.ID).Updates(updates).Error
}

func (r *ideaRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.WithContext(ctx).Delete(&IdeaModel{}, id).Error
}

func (r *ideaRepo) List(ctx context.Context, query *biz.IdeaListQuery) (*biz.IdeaListResult, error) {
	var models []IdeaModel
	var total int64

	db := r.data.db.WithContext(ctx).Model(&IdeaModel{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.MinScore > 0 {
		db = db.Where("score_overall >= ?", query.MinScore)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	sortBy := "created_at"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	sortOrder := "desc"
	if strings.EqualFold(query.SortOrder, "asc") {
		sortOrder = "asc"
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Offset(offset).Limit(query.PageSize).
		Find(&models).Error; err != nil {
		return nil, err
	}

	items := make([]*biz.Idea, len(models))
	for i, m := range models {
		items[i] = toIdeaBiz(&m)
	}

	return &biz.IdeaListResult{
		Items:    items,
		Total:    int(total),
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (r *ideaRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.data.db.WithContext(ctx).Model(&IdeaModel{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ideaRepo) UpdateScores(ctx context.Context, id int64, feasibility, economics, profit, overall float64) error {
	return r.data.db.WithContext(ctx).Model(&IdeaModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"score_feasibility": feasibility,
		"score_economics":   economics,
		"score_profit":      profit,
		"score_overall":     overall,
	}).Error
}

func (r *ideaRepo) UpdateDebateLog(ctx context.Context, id int64, debateLog string, roundCount int) error {
	return r.data.db.WithContext(ctx).Model(&IdeaModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"debate_log":  debateLog,
		"round_count": roundCount,
	}).Error
}

func (r *ideaRepo) GetPending(ctx context.Context, limit int) ([]*biz.Idea, error) {
	var models []IdeaModel
	if err := r.data.db.WithContext(ctx).Where("status = ?", "pending").
		Order("created_at ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]*biz.Idea, len(models))
	for i, m := range models {
		items[i] = toIdeaBiz(&m)
	}
	return items, nil
}

func (r *ideaRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	if err := r.data.db.WithContext(ctx).Model(&IdeaModel{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// -------- 转换函数 --------

func toBizIdeaModel(idea *biz.Idea) *IdeaModel {
	tagsJSON := "[]"
	if len(idea.Tags) > 0 {
		if b, err := json.Marshal(idea.Tags); err == nil {
			tagsJSON = string(b)
		}
	}
	return &IdeaModel{
		ID:               idea.ID,
		Topic:            idea.Topic,
		Status:           idea.Status,
		Proposal:         idea.Proposal,
		DebateLog:        idea.DebateLog,
		FinalReport:      idea.FinalReport,
		ScoreFeasibility: idea.ScoreFeasibility,
		ScoreEconomics:   idea.ScoreEconomics,
		ScoreProfit:      idea.ScoreProfit,
		ScoreOverall:     idea.ScoreOverall,
		RoundCount:       idea.RoundCount,
		TechStack:        idea.TechStack,
		DevPrompt:        idea.DevPrompt,
		Tags:             tagsJSON,
		ProductName:      idea.ProductName,
		OneLiner:         idea.OneLiner,
		SearchData:       idea.SearchData,
		JudgeRefined:     idea.JudgeRefined,
		VoteResult:       idea.VoteResult,
	}
}

func toIdeaBiz(m *IdeaModel) *biz.Idea {
	var tags []string
	if m.Tags != "" {
		_ = json.Unmarshal([]byte(m.Tags), &tags)
	}
	return &biz.Idea{
		ID:               m.ID,
		Topic:            m.Topic,
		Status:           m.Status,
		Proposal:         m.Proposal,
		DebateLog:        m.DebateLog,
		FinalReport:      m.FinalReport,
		ScoreFeasibility: m.ScoreFeasibility,
		ScoreEconomics:   m.ScoreEconomics,
		ScoreProfit:      m.ScoreProfit,
		ScoreOverall:     m.ScoreOverall,
		RoundCount:       m.RoundCount,
		TechStack:        m.TechStack,
		DevPrompt:        m.DevPrompt,
		Tags:             tags,
		ProductName:      m.ProductName,
		OneLiner:         m.OneLiner,
		SearchData:       m.SearchData,
		JudgeRefined:     m.JudgeRefined,
		VoteResult:       m.VoteResult,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// GetStats 获取聚合统计数据
func (r *ideaRepo) GetStats(ctx context.Context) (*biz.IdeaStats, error) {
	stats := &biz.IdeaStats{
		StatusCounts: make(map[string]int64),
		AvgScores:    make(map[string]float64),
		ScoreBuckets: make(map[string]int64),
	}

	// 总数
	r.data.db.WithContext(ctx).Model(&IdeaModel{}).Count(&stats.Total)

	// 按状态统计
	type statusRow struct {
		Status string
		Cnt    int64
	}
	var statusRows []statusRow
	r.data.db.WithContext(ctx).Model(&IdeaModel{}).
		Select("status, COUNT(*) as cnt").
		Group("status").
		Scan(&statusRows)
	for _, row := range statusRows {
		stats.StatusCounts[row.Status] = row.Cnt
	}

	// 平均分 (仅已评分)
	type avgRow struct {
		AvgFeasibility float64
		AvgEconomics   float64
		AvgProfit      float64
		AvgOverall     float64
	}
	var avg avgRow
	r.data.db.WithContext(ctx).Model(&IdeaModel{}).
		Where("score_overall > 0").
		Select("AVG(score_feasibility) as avg_feasibility, AVG(score_economics) as avg_economics, AVG(score_profit) as avg_profit, AVG(score_overall) as avg_overall").
		Scan(&avg)
	stats.AvgScores["feasibility"] = avg.AvgFeasibility
	stats.AvgScores["economics"] = avg.AvgEconomics
	stats.AvgScores["profit"] = avg.AvgProfit
	stats.AvgScores["overall"] = avg.AvgOverall

	// 总辩论轮次
	r.data.db.WithContext(ctx).Model(&IdeaModel{}).
		Select("COALESCE(SUM(round_count), 0)").
		Scan(&stats.TotalRounds)

	// 评分分布
	type bucketRow struct {
		Bucket string
		Cnt    int64
	}
	var bucketRows []bucketRow
	r.data.db.WithContext(ctx).Model(&IdeaModel{}).
		Where("score_overall > 0").
		Select(`CASE 
			WHEN score_overall < 2 THEN '0-2'
			WHEN score_overall < 4 THEN '2-4'
			WHEN score_overall < 6 THEN '4-6'
			WHEN score_overall < 8 THEN '6-8'
			ELSE '8-10' END as bucket, COUNT(*) as cnt`).
		Group("bucket").
		Scan(&bucketRows)
	for _, row := range bucketRows {
		stats.ScoreBuckets[row.Bucket] = row.Cnt
	}

	// 投票统计
	var totalVoted int64
	r.data.db.WithContext(ctx).Model(&IdeaModel{}).
		Where("vote_result IS NOT NULL AND vote_result != ''").
		Count(&totalVoted)

	if totalVoted > 0 {
		vs := &biz.VoteStats{TotalVoted: totalVoted}
		// 统计各最终判定的数量
		type verdictRow struct {
			Cnt int64
		}
		var yesCnt, noCnt, condCnt verdictRow
		r.data.db.WithContext(ctx).Model(&IdeaModel{}).
			Where("vote_result LIKE '%\"final_verdict\":\"YES\"%'").
			Select("COUNT(*) as cnt").Scan(&yesCnt)
		r.data.db.WithContext(ctx).Model(&IdeaModel{}).
			Where("vote_result LIKE '%\"final_verdict\":\"NO\"%'").
			Select("COUNT(*) as cnt").Scan(&noCnt)
		r.data.db.WithContext(ctx).Model(&IdeaModel{}).
			Where("vote_result LIKE '%\"final_verdict\":\"CONDITIONAL\"%'").
			Select("COUNT(*) as cnt").Scan(&condCnt)

		vs.YesRate = float64(yesCnt.Cnt) / float64(totalVoted) * 100
		vs.NoRate = float64(noCnt.Cnt) / float64(totalVoted) * 100
		vs.CondRate = float64(condCnt.Cnt) / float64(totalVoted) * 100
		stats.VoteStats = vs
	}

	return stats, nil
}
