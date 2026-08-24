// Package catalog 负责地震目录的生命周期管理（创建、发布、封存）。
package catalog

import (
	"fmt"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

// Manager 封装目录业务逻辑。
type Manager struct {
	db *store.DB
}

// NewManager 构造目录管理器。
func NewManager(db *store.DB) *Manager { return &Manager{db: db} }

// Create 创建地震目录（初始状态 importing）。
func (m *Manager) Create(name, region string) (*model.Catalog, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: catalog name is required", model.ErrInvalid)
	}
	c := &model.Catalog{
		Name:   name,
		Region: region,
		Status: model.CatalogImporting,
	}
	if _, err := m.db.InsertCatalog(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Get 读取目录。
func (m *Manager) Get(id int64) (*model.Catalog, error) {
	return m.db.GetCatalog(id)
}

// List 列出全部目录。
func (m *Manager) List() ([]model.Catalog, error) {
	return m.db.ListCatalogs()
}

// MarkAnalyzing 把目录推进到可分析状态（导入完成）。
func (m *Manager) MarkAnalyzing(id int64) error {
	c, err := m.db.GetCatalog(id)
	if err != nil {
		return err
	}
	switch c.Status {
	case model.CatalogImporting, model.CatalogAnalyzing:
		return m.db.UpdateCatalogStatus(id, model.CatalogAnalyzing, false, false)
	case model.CatalogArchived:
		return model.ErrArchived
	default:
		return model.ErrConflict
	}
}

// Publish 发布目录（analyzing → published）。
func (m *Manager) Publish(id int64) error {
	c, err := m.db.GetCatalog(id)
	if err != nil {
		return err
	}
	switch c.Status {
	case model.CatalogAnalyzing, model.CatalogPublished:
		return m.db.UpdateCatalogStatus(id, model.CatalogPublished, true, false)
	case model.CatalogArchived:
		return model.ErrArchived
	default:
		return model.ErrConflict
	}
}

// Archive 封存目录（published → archived，只读）。
func (m *Manager) Archive(id int64) error {
	c, err := m.db.GetCatalog(id)
	if err != nil {
		return err
	}
	switch c.Status {
	case model.CatalogPublished, model.CatalogArchived:
		return m.db.UpdateCatalogStatus(id, model.CatalogArchived, false, true)
	default:
		return model.ErrConflict
	}
}

// IsWritable 判断目录是否可写（未封存）。
func (m *Manager) IsWritable(id int64) error {
	c, err := m.db.GetCatalog(id)
	if err != nil {
		return err
	}
	if c.Status == model.CatalogArchived {
		return nil
	}
	return nil
}
