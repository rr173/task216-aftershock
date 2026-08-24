// Package review 负责研究人员对余震簇的人工复核：锁定主震、拆分、合并。
package review

import (
	"fmt"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/store"
)

// Manager 封装簇复核业务逻辑。
type Manager struct {
	db *store.DB
}

// NewManager 构造复核管理器。
func NewManager(db *store.DB) *Manager { return &Manager{db: db} }

// LockMainshock 把某簇的主震锁定为指定事件（该事件须已在该簇内）。
// 锁定后簇状态转为 confirmed，主震角色写入成员关系。
func (m *Manager) LockMainshock(clusterID, eventID int64) (*model.Cluster, error) {
	c, err := m.db.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}
	if c.Status == model.ClusterSplit || c.Status == model.ClusterMerged {
		return nil, fmt.Errorf("%w: cluster %d already %s", model.ErrConflict, clusterID, c.Status)
	}

	ev, err := m.db.GetEvent(eventID)
	if err != nil {
		return nil, err
	}
	if ev.CatalogID != c.CatalogID {
		return nil, fmt.Errorf("%w: event %d not in same catalog", model.ErrInvalid, eventID)
	}

	// 校验事件已归属该簇（或本身是该簇当前主震）。
	members, err := m.db.ListMembersByCluster(clusterID)
	if err != nil {
		return nil, err
	}
	inCluster := false
	for _, mem := range members {
		if mem.EventID == eventID {
			inCluster = true
			break
		}
	}
	if !inCluster {
		return nil, fmt.Errorf("%w: event %d not a member of cluster %d", model.ErrInvalid, eventID, clusterID)
	}

	// 更新主震角色：新主震 role=mainshock，原主震（若有）降为 aftershock。
	for _, mem := range members {
		if mem.EventID == eventID {
			if err := m.db.DeleteMembership(clusterID, eventID); err != nil {
				return nil, err
			}
			if err := m.db.AddMembership(clusterID, eventID, model.RoleMainshock); err != nil {
				return nil, err
			}
		} else if mem.Role == model.RoleMainshock {
			if err := m.db.DeleteMembership(clusterID, mem.EventID); err != nil {
				return nil, err
			}
			if err := m.db.AddMembership(clusterID, mem.EventID, model.RoleAftershock); err != nil {
				return nil, err
			}
		}
	}

	if err := m.db.UpdateCluster(clusterID, c.MainshockID, model.ClusterConfirmed, c.Confidence); err != nil {
		return nil, err
	}
	return m.db.GetCluster(clusterID)
}

// ListClusters 列出目录下的全部簇。
func (m *Manager) ListClusters(catalogID int64) ([]model.Cluster, error) {
	return m.db.ListClustersByCatalog(catalogID)
}

// GetCluster 读取簇。
func (m *Manager) GetCluster(id int64) (*model.Cluster, error) {
	return m.db.GetCluster(id)
}

// ListMembers 列出簇内成员关系。
func (m *Manager) ListMembers(clusterID int64) ([]model.ClusterMembership, error) {
	return m.db.ListMembersByCluster(clusterID)
}
