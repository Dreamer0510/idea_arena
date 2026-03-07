package data

import (
	"context"
	"fmt"
	"strings"

	"idea_arena/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
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

	if c == nil || c.Database == nil {
		return nil, nil, fmt.Errorf("database config is required")
	}

	driver := strings.ToLower(strings.TrimSpace(c.Database.Driver))
	if driver == "" {
		driver = "sqlite"
	}
	source := strings.TrimSpace(c.Database.Source)
	if source == "" {
		if driver == "sqlite" || driver == "sqlite3" {
			source = "data/idea_arena.db"
		} else {
			return nil, nil, fmt.Errorf("database source is required for driver %s", driver)
		}
	}

	var dialector gorm.Dialector
	switch driver {
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(source)
	case "mysql":
		dialector = mysql.Open(source)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver: %s (supported: sqlite, mysql)", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, err
	}
	helper.Infof("database connected (driver=%s)", driver)

	// 自动迁移
	if err := autoMigrate(db); err != nil {
		return nil, nil, err
	}

	// 初始化 Redis
	var rdb *redis.Client
	if c.Redis != nil && strings.TrimSpace(c.Redis.Addr) != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     c.Redis.Addr,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		})

		if err := rdb.Ping(context.Background()).Err(); err != nil {
			helper.Warnf("Redis connection failed: %v, running without cache", err)
			rdb = nil
		}
	} else {
		helper.Info("Redis disabled: data.redis.addr is empty")
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
