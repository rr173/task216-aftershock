// Package service 是业务编排层：组合目录、事件、簇、复核与版本等子模块。
package service

import (
	"task216-aftershock/internal/catalog"
	"task216-aftershock/internal/event"
	"task216-aftershock/internal/review"
	"task216-aftershock/internal/store"
	"task216-aftershock/internal/versioning"
)

// Service 聚合全部业务管理器，是 HTTP 层与 smoke 自检的统一入口。
type Service struct {
	Catalog   *catalog.Manager
	Event     *event.Manager
	Review    *review.Manager
	Versioning *versioning.Manager

	db *store.DB
}

// New 构造服务编排层。
func New(db *store.DB) *Service {
	return &Service{
		Catalog:    catalog.NewManager(db),
		Event:      event.NewManager(db),
		Review:     review.NewManager(db),
		Versioning: versioning.NewManager(db),
		db:         db,
	}
}

// DB 暴露底层存储（供 smoke 与统计读取）。
func (s *Service) DB() *store.DB { return s.db }
