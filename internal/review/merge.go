package review

import (
	"fmt"

	"task216-aftershock/internal/model"
)

// MergeClusters 把簇 B 合并进簇 A（B 的成员迁入 A，B 标记 merged）。
// 合并后 A 的主震保持为两者中震级更大的主震事件。
// 返回 (合并后的簇 A, 错误)。
func (m *Manager) MergeClusters(clusterAID, clusterBID int64) (*model.Cluster, error) {
	if clusterAID == clusterBID {
		return nil, fmt.Errorf("%w: cannot merge a cluster with itself", model.ErrInvalid)
	}
	a, err := m.db.GetCluster(clusterAID)
	if err != nil {
		return nil, err
	}
	b, err := m.db.GetCluster(clusterBID)
	if err != nil {
		return nil, err
	}
	if a.CatalogID != b.CatalogID {
		return nil, fmt.Errorf("%w: clusters belong to different catalogs", model.ErrInvalid)
	}
	if a.Status == model.ClusterMerged || b.Status == model.ClusterMerged {
		return nil, fmt.Errorf("%w: a merged cluster cannot be merged again", model.ErrConflict)
	}

	// 解析各自的主震事件：优先取簇持久化的 MainshockID，缺失时回退到成员关系中的 mainshock 角色。
	am, err := m.resolveMainshock(a)
	if err != nil {
		return nil, err
	}
	bm, err := m.resolveMainshock(b)
	if err != nil {
		return nil, err
	}

	// 确定合并后主震：比较两个主震的震级，取震级更大者（相等时保持 A）。
	newMainshockID := a.MainshockID
	if bm.Magnitude > am.Magnitude {
		newMainshockID = b.MainshockID
	}

	// 迁移 B 的成员到 A。
	membersB, err := m.db.ListMembersByCluster(clusterBID)
	if err != nil {
		return nil, err
	}
	for _, mem := range membersB {
		if err := m.db.DeleteMembership(clusterBID, mem.EventID); err != nil {
			return nil, err
		}
		role := mem.Role
		if mem.EventID == newMainshockID {
			role = model.RoleMainshock
		} else if mem.EventID == b.MainshockID && newMainshockID != b.MainshockID {
			role = model.RoleAftershock
		}
		if err := m.db.AddMembership(clusterAID, mem.EventID, role); err != nil {
			return nil, err
		}
	}

	// 若新主震原本在 A 中为 aftershock，修正角色。
	if newMainshockID != a.MainshockID {
		membersA, _ := m.db.ListMembersByCluster(clusterAID)
		for _, mem := range membersA {
			if mem.EventID == newMainshockID && mem.Role == model.RoleAftershock {
				if err := m.db.DeleteMembership(clusterAID, mem.EventID); err != nil {
					return nil, err
				}
				if err := m.db.AddMembership(clusterAID, mem.EventID, model.RoleMainshock); err != nil {
					return nil, err
				}
			}
		}
	}

	// 更新置信度（合并取较高者）与状态。
	conf := a.Confidence
	if b.Confidence > conf {
		conf = b.Confidence
	}
	if err := m.db.UpdateCluster(clusterAID, newMainshockID, model.ClusterConfirmed, conf); err != nil {
		return nil, err
	}
	if err := m.db.UpdateCluster(clusterBID, b.MainshockID, model.ClusterMerged, b.Confidence); err != nil {
		return nil, err
	}
	return m.db.GetCluster(clusterAID)
}

// resolveMainshock 解析簇的主震事件：优先用簇持久化的 MainshockID 加载事件，
// 该 ID 缺失（0 或事件已被删除）时回退到成员关系中 role=mainshock 的事件。
// 返回 (主震事件, 错误)；若找不到主震则返回 ErrInvalid，避免后续解引用空指针。
func (m *Manager) resolveMainshock(c *model.Cluster) (*model.Event, error) {
	if c.MainshockID != 0 {
		if ev, err := m.db.GetEvent(c.MainshockID); err == nil {
			return ev, nil
		}
	}
	members, err := m.db.ListMembersByCluster(c.ID)
	if err != nil {
		return nil, err
	}
	for _, mem := range members {
		if mem.Role == model.RoleMainshock {
			if ev, err := m.db.GetEvent(mem.EventID); err == nil {
				return ev, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: cluster %d has no resolvable mainshock", model.ErrInvalid, c.ID)
}
