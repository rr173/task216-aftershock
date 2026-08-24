package review

import (
	"fmt"

	"task216-aftershock/internal/model"
)

// SplitCluster 把簇内的部分事件拆分为新簇。
// 原簇保留主震与其余成员，拆分出的事件归入新簇（状态 split）。
// 返回 (新簇, 错误)。
func (m *Manager) SplitCluster(clusterID int64, eventIDs []int64) (*model.Cluster, error) {
	if len(eventIDs) == 0 {
		return nil, fmt.Errorf("%w: no events to split", model.ErrInvalid)
	}
	c, err := m.db.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}
	if c.Status == model.ClusterMerged || c.Status == model.ClusterSplit {
		return nil, fmt.Errorf("%w: cluster %d not splittable", model.ErrConflict, clusterID)
	}

	members, err := m.db.ListMembersByCluster(clusterID)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[int64]bool)
	for _, mem := range members {
		memberSet[mem.EventID] = true
	}

	// 主震不能被拆分出去。
	splitSet := make(map[int64]bool)
	for _, id := range eventIDs {
		if id == c.MainshockID {
			return nil, fmt.Errorf("%w: mainshock cannot be split out", model.ErrInvalid)
		}
		if !memberSet[id] {
			return nil, fmt.Errorf("%w: event %d not in cluster %d", model.ErrInvalid, id, clusterID)
		}
		splitSet[id] = true
	}

	// 创建新簇承接拆分事件。
	newCluster := &model.Cluster{
		CatalogID:   c.CatalogID,
		MainshockID: 0,
		Status:      model.ClusterSplit,
		Confidence:  0,
	}
	newID, err := m.db.InsertCluster(newCluster)
	if err != nil {
		return nil, err
	}

	// 迁移成员关系。
	for _, mem := range members {
		if !splitSet[mem.EventID] {
			continue
		}
		if err := m.db.DeleteMembership(clusterID, mem.EventID); err != nil {
			return nil, err
		}
		role := model.RoleAftershock
		if err := m.db.AddMembership(newID, mem.EventID, role); err != nil {
			return nil, err
		}
	}

	// 原簇标记为 split。
	if err := m.db.UpdateCluster(clusterID, c.MainshockID, model.ClusterSplit, c.Confidence); err != nil {
		return nil, err
	}
	return m.db.GetCluster(newID)
}
