// Package versioning 负责目录版本的创建、冻结与替代。
package versioning

import (
	"fmt"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

// Manager 封装版本发布业务逻辑。
type Manager struct {
	db *store.DB
}

// NewManager 构造版本管理器。
func NewManager(db *store.DB) *Manager { return &Manager{db: db} }

// CreateDraft 创建目录版本草稿，绑定当前归属快照哈希。
func (m *Manager) CreateDraft(catalogID int64, label string) (*model.Version, error) {
	c, err := m.db.GetCatalog(catalogID)
	if err != nil {
		return nil, err
	}
	if c.Status == model.CatalogArchived {
		return nil, model.ErrArchived
	}
	hash, err := Snapshot(m.db, catalogID)
	if err != nil {
		return nil, err
	}
	num, err := m.db.NextVersionNumber(catalogID)
	if err != nil {
		return nil, err
	}
	v := &model.Version{
		CatalogID:    catalogID,
		Number:       num,
		Label:        label,
		Status:       model.VersionDraft,
		SnapshotHash: hash,
	}
	if _, err := m.db.InsertVersion(v); err != nil {
		return nil, err
	}
	return v, nil
}

// Publish 冻结版本：draft → published，并把目录其它版本标记为 superseded。
func (m *Manager) Publish(versionID int64) (*model.Version, error) {
	v, err := m.db.GetVersion(versionID)
	if err != nil {
		return nil, err
	}
	switch v.Status {
	case model.VersionPublished:
		return v, nil
	case model.VersionDraft:
		// ok
	default:
		return nil, fmt.Errorf("%w: version in status %s", model.ErrConflict, v.Status)
	}
	if err := m.db.PublishVersion(versionID); err != nil {
		return nil, err
	}
	if err := m.db.SupersedeVersions(v.CatalogID, 0, versionID); err != nil {
		return nil, err
	}
	return m.db.GetVersion(versionID)
}

// Get 读取版本。
func (m *Manager) Get(id int64) (*model.Version, error) {
	return m.db.GetVersion(id)
}

// List 列出目录版本。
func (m *Manager) List(catalogID int64) ([]model.Version, error) {
	return m.db.ListVersionsByCatalog(catalogID)
}

// ActivePublished 返回目录当前活动（已发布且未被替代）版本。
func (m *Manager) ActivePublished(catalogID int64) (*model.Version, error) {
	return m.db.GetActivePublishedVersion(catalogID)
}
