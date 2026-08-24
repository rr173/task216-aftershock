package event

import (
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

// LocErrorThresholds 定位稳定性阈值（默认：水平 50 km、垂直 40 km 以上视为不稳）。
type LocErrorThresholds struct {
	MaxErrH float64
	MaxErrZ float64
}

// DefaultThresholds 返回默认定位稳定性阈值。
func DefaultThresholds() LocErrorThresholds {
	return LocErrorThresholds{MaxErrH: 50, MaxErrZ: 40}
}

// Manager 封装事件校验与持久化，供服务层调用。
type Manager struct {
	db *store.DB
}

// NewManager 构造事件管理器。
func NewManager(db *store.DB) *Manager { return &Manager{db: db} }

// Ingest 校验并写入一条事件到指定目录。
// 返回 (事件, 是否为新事件, 错误)。重复提交返回已有事件且 newEvent=false。
func (m *Manager) Ingest(catalogID int64, in EventInput) (*model.Event, bool, error) {
	if err := Validate(in); err != nil {
		return nil, false, err
	}

	fp := Fingerprint(in)
	if existing, err := m.db.FindEventByFingerprint(catalogID, fp); err == nil {
		return existing, false, nil
	} else if err != model.ErrNotFound {
		return nil, false, err
	}

	e := &model.Event{
		CatalogID:   catalogID,
		OriginTime:  in.OriginTime,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		DepthKm:     in.DepthKm,
		Magnitude:   in.Magnitude,
		LocErrorH:   in.LocErrorH,
		LocErrorZ:   in.LocErrorZ,
		Fingerprint: fp,
		Status:      model.EventPending,
	}

	th := DefaultThresholds()
	if LocUnstable(in, th.MaxErrH, th.MaxErrZ) {
		e.Status = model.EventUnstable
	} else {
		e.Status = model.EventValid
	}

	id, err := m.db.InsertEvent(e)
	if err != nil {
		return nil, false, err
	}
	e.ID = id
	return e, true, nil
}

// Get 读取事件。
func (m *Manager) Get(id int64) (*model.Event, error) {
	return m.db.GetEvent(id)
}

// List 列出目录下全部事件。
func (m *Manager) List(catalogID int64) ([]model.Event, error) {
	return m.db.ListEventsByCatalog(catalogID)
}

// MarkUnstable 把事件标记为定位不稳。
func (m *Manager) MarkUnstable(id int64) error {
	e, err := m.db.GetEvent(id)
	if err != nil {
		return err
	}
	if e.Status == model.EventValid || e.Status == model.EventPending {
		return m.db.UpdateEventStatus(id, model.EventUnstable)
	}
	return nil
}

// MarkValid 把定位不稳事件重新标记为有效（研究人员复核后）。
func (m *Manager) MarkValid(id int64) error {
	e, err := m.db.GetEvent(id)
	if err != nil {
		return err
	}
	if e.Status == model.EventUnstable {
		return m.db.UpdateEventStatus(id, model.EventValid)
	}
	return nil
}
