// Package model 定义地震余震簇时空窗识别服务的核心实体、状态机与业务错误。
package model

import "time"

// Catalog 地震目录：一批带定位误差的地震事件集合，是分析与发布的顶层单元。
type Catalog struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Region      string    `json:"region"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	ArchivedAt  time.Time `json:"archived_at,omitempty"`
}

// Event 地震事件：一条带定位误差与震级的地震记录。
// Fingerprint 用于幂等去重，由时间、坐标、深度、震级与目录共同派生。
type Event struct {
	ID          int64     `json:"id"`
	CatalogID   int64     `json:"catalog_id"`
	OriginTime  time.Time `json:"origin_time"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	DepthKm     float64   `json:"depth_km"`
	Magnitude   float64   `json:"magnitude"`
	LocErrorH   float64   `json:"loc_error_h_km"`
	LocErrorZ   float64   `json:"loc_error_z_km"`
	Fingerprint string    `json:"fingerprint"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Cluster 余震簇：围绕一个主震（可锁定或候选）的时空窗归属集合。
type Cluster struct {
	ID          int64     `json:"id"`
	CatalogID   int64     `json:"catalog_id"`
	MainshockID int64     `json:"mainshock_id"`
	Status      string    `json:"status"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
}

// ClusterMembership 簇与事件的关联，记录事件在簇内的角色。
type ClusterMembership struct {
	ClusterID int64 `json:"cluster_id"`
	EventID   int64 `json:"event_id"`
	Role      string `json:"role"`
}

// Conflict 边界冲突：一个事件同时落入多个候选簇的时空窗，需要研究人员裁决。
type Conflict struct {
	ID         int64     `json:"id"`
	CatalogID  int64     `json:"catalog_id"`
	EventID    int64     `json:"event_id"`
	ClusterAID int64     `json:"cluster_a_id"`
	ClusterBID int64     `json:"cluster_b_id"`
	Status     string    `json:"status"`
	Resolution string    `json:"resolution"`
	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
}

// Version 目录版本：一次归属结果的不可变快照，供引用与追溯。
type Version struct {
	ID           int64     `json:"id"`
	CatalogID    int64     `json:"catalog_id"`
	Number       int       `json:"number"`
	Label        string    `json:"label"`
	Status       string    `json:"status"`
	SnapshotHash string    `json:"snapshot_hash"`
	CreatedAt    time.Time `json:"created_at"`
	FrozenAt     time.Time `json:"frozen_at,omitempty"`
	SupersededBy int64     `json:"superseded_by,omitempty"`
}

// ClusterInput 是构建/识别余震簇时的参数输入。
type ClusterInput struct {
	MinMainshockMag  float64 `json:"min_mainshock_mag"`
	TimeWindowDays   float64 `json:"time_window_days"`
	DeltaMagnitude   float64 `json:"delta_magnitude"`
	UseUtsuDistance  bool    `json:"use_utsu_distance"`
	DistanceWindowKm float64 `json:"distance_window_km"`
}
