package model

// 目录状态机：importing → analyzing → published → archived。
const (
	CatalogImporting = "importing"
	CatalogAnalyzing = "analyzing"
	CatalogPublished = "published"
	CatalogArchived  = "archived"
)

// 事件状态机：pending → valid / unstable / duplicate。
const (
	EventPending   = "pending"
	EventValid     = "valid"
	EventUnstable  = "unstable"
	EventDuplicate = "duplicate"
)

// 余震簇状态机：candidate → overlapping → confirmed / split / merged。
const (
	ClusterCandidate   = "candidate"
	ClusterOverlapping = "overlapping"
	ClusterConfirmed   = "confirmed"
	ClusterSplit       = "split"
	ClusterMerged      = "merged"
)

// 边界冲突状态机：open → resolved。
const (
	ConflictOpen     = "open"
	ConflictResolved = "resolved"
)

// 目录版本状态机：draft → published → superseded。
const (
	VersionDraft      = "draft"
	VersionPublished  = "published"
	VersionSuperseded = "superseded"
)

// 簇内角色。
const (
	RoleMainshock  = "mainshock"
	RoleAftershock = "aftershock"
)

// 冲突裁决结果。
const (
	ResolutionAssignA = "assign_a"
	ResolutionAssignB = "assign_b"
	ResolutionSplit   = "split"
	ResolutionMerge   = "merge"
)

// validCatalogStatus 目录合法状态集合。
var validCatalogStatus = map[string]bool{
	CatalogImporting: true,
	CatalogAnalyzing: true,
	CatalogPublished: true,
	CatalogArchived:  true,
}

// validEventStatus 事件合法状态集合。
var validEventStatus = map[string]bool{
	EventPending:   true,
	EventValid:     true,
	EventUnstable:  true,
	EventDuplicate: true,
}

// validClusterStatus 簇合法状态集合。
var validClusterStatus = map[string]bool{
	ClusterCandidate:   true,
	ClusterOverlapping: true,
	ClusterConfirmed:   true,
	ClusterSplit:       true,
	ClusterMerged:      true,
}

// validConflictStatus 冲突合法状态集合。
var validConflictStatus = map[string]bool{
	ConflictOpen:     true,
	ConflictResolved: true,
}

// validVersionStatus 版本合法状态集合。
var validVersionStatus = map[string]bool{
	VersionDraft:      true,
	VersionPublished:  true,
	VersionSuperseded: true,
}

// IsValidCatalogStatus 校验目录状态是否合法。
func IsValidCatalogStatus(s string) bool { return validCatalogStatus[s] }

// IsValidEventStatus 校验事件状态是否合法。
func IsValidEventStatus(s string) bool { return validEventStatus[s] }

// IsValidClusterStatus 校验簇状态是否合法。
func IsValidClusterStatus(s string) bool { return validClusterStatus[s] }

// IsValidConflictStatus 校验冲突状态是否合法。
func IsValidConflictStatus(s string) bool { return validConflictStatus[s] }

// IsValidVersionStatus 校验版本状态是否合法。
func IsValidVersionStatus(s string) bool { return validVersionStatus[s] }

// 领域常量。
const (
	// DefaultMinMainshockMag 默认主震震级下限。
	DefaultMinMainshockMag = 3.0
	// DefaultTimeWindowDays 默认主震触发时间窗（天）。
	DefaultTimeWindowDays = 7.0
	// DefaultDeltaMagnitude 默认余震震级下限差值（主震震级 - 余震震级上限）。
	DefaultDeltaMagnitude = 1.2
	// MinLocErrorKm 定位误差下限，防止除零。
	MinLocErrorKm = 0.01
)
