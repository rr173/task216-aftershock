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

	// 确定合并后主震：比较两个主震的震级，取震级更大者。
	newMainshockID := a.MainshockID
	am, errA := m.db.GetEvent(a.MainshockID)
	bm, errB := m.db.GetEvent(b.MainshockID)
	if errA == nil && errB == nil && bm.Magnitude > am.Magnitude {
		newMainshockID = b.MainshockID
	}

	// 迁移 B 的成员到 A，并按新主震同步角色。
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
			// B 的原主震若不再担任主震，降为余震。
			role = model.RoleAftershock
		}
		if err := m.db.AddMembership(clusterAID, mem.EventID, role); err != nil {
			return nil, err
		}
	}

	// 若主震发生变更，重写 A 中成员角色：新主震升为主震，原主震降为余震。
	if newMainshockID != a.MainshockID {
		membersA, err := m.db.ListMembersByCluster(clusterAID)
		if err != nil {
			return nil, err
		}
		for _, mem := range membersA {
			if mem.EventID == newMainshockID {
				if mem.Role != model.RoleMainshock {
					if err := m.db.DeleteMembership(clusterAID, mem.EventID); err != nil {
						return nil, err
					}
					if err := m.db.AddMembership(clusterAID, mem.EventID, model.RoleMainshock); err != nil {
						return nil, err
					}
				}
			} else if mem.EventID == a.MainshockID && mem.Role == model.RoleMainshock {
				if err := m.db.DeleteMembership(clusterAID, mem.EventID); err != nil {
					return nil, err
				}
				if err := m.db.AddMembership(clusterAID, mem.EventID, model.RoleAftershock); err != nil {
					return nil, err
				}
			}
		}
	}

	// 更新置信度（合并取较高者）与状态，并把新主震写入簇 A。
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
