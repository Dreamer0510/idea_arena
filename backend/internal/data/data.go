package data

import (
	"context"

	"idea_arena/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Data 数据层依赖
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewData 创建数据层实例
func NewData(c *conf.Data, l log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(l)

	// 初始化 SQLite
	db, err := gorm.Open(sqlite.Open(c.Database.Source), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, err
	}

	// 自动迁移
	if err := autoMigrate(db); err != nil {
		return nil, nil, err
	}

	// 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		helper.Warnf("Redis connection failed: %v, running without cache", err)
		rdb = nil
	}

	cleanup := func() {
		helper.Info("closing data resources")
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		if rdb != nil {
			rdb.Close()
		}
	}

	return &Data{db: db, rdb: rdb}, cleanup, nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&IdeaModel{},
		&UserModel{},
		&ManualQueueModel{},
		&SettingModel{},
		&SearchKeywordModel{},
		&DiscoveredTopicModel{},
		&CrawlerPluginModel{},
		&DiscoveryTagModel{},
		&SocialPostModel{},
		&AgentConfigModel{},
		&LLMProviderModel{},
		&LLMProviderModelListModel{},
	)
}
