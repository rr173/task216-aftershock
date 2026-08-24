package service

import (
	"fmt"

	"task216-aftershock/internal/cluster"
	"task216-aftershock/internal/model"
)

// IdentifyResult 一次簇识别的落库结果。
type IdentifyResult struct {
	ClusterCount  int     `json:"cluster_count"`
	ConflictCount int     `json:"conflict_count"`
	IsolatedCount int     `json:"isolated_count"`
	ClusterIDs    []int64 `json:"cluster_ids"`
}

// IdentifyClusters 执行余震簇识别并落库：
// 读取目录事件 → 纯计算时空窗归属 → 清空旧簇 → 写入簇、成员与冲突。
func (s *Service) IdentifyClusters(catalogID int64, in model.ClusterInput) (*IdentifyResult, error) {
	c, err := s.db.GetCatalog(catalogID)
	if err != nil {
		return nil, err
	}
	if c.Status == model.CatalogArchived {
		return nil, model.ErrArchived
	}

	params := cluster.FromInput(in)
	events, err := s.db.ListEventsByCatalog(catalogID)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("%w: catalog has no events", model.ErrInvalid)
	}

	result, err := cluster.Build(events, params)
	if err != nil {
		return nil, err
	}

	// 清空旧簇与冲突，重建。
	if err := s.db.DeleteClustersByCatalog(catalogID); err != nil {
		return nil, err
	}
	if err := s.db.DeleteConflictsByCatalog(catalogID); err != nil {
		return nil, err
	}

	res := &IdentifyResult{}
	for _, draft := range result.Clusters {
		cl := &model.Cluster{
			CatalogID:   catalogID,
			MainshockID: draft.Mainshock.ID,
			Status:      model.ClusterCandidate,
			Confidence:  draft.Confidence,
		}
		cid, err := s.db.InsertCluster(cl)
		if err != nil {
			return nil, err
		}
		res.ClusterIDs = append(res.ClusterIDs, cid)

		for _, e := range draft.Members {
			role := model.RoleAftershock
			if e.ID == draft.Mainshock.ID {
				role = model.RoleMainshock
			}
			if err := s.db.AddMembership(cid, e.ID, role); err != nil {
				return nil, err
			}
		}
	}
	res.ClusterCount = len(result.Clusters)

	// 落库冲突：每个冲突事件与它的多簇归属。
	for _, cd := range result.Conflicts {
		if len(cd.ClusterRef) < 2 {
			continue
		}
		aID := res.ClusterIDs[cd.ClusterRef[0]]
		bID := res.ClusterIDs[cd.ClusterRef[1]]
		conf := &model.Conflict{
			CatalogID:  catalogID,
			EventID:    cd.Event.ID,
			ClusterAID: aID,
			ClusterBID: bID,
			Status:     model.ConflictOpen,
		}
		if _, err := s.db.InsertConflict(conf); err != nil {
			return nil, err
		}
		res.ConflictCount++
	}

	// 涉及冲突的簇标记 overlapping。
	overlapClusterSet := make(map[int64]bool)
	for _, cd := range result.Conflicts {
		for _, ref := range cd.ClusterRef {
			overlapClusterSet[res.ClusterIDs[ref]] = true
		}
	}
	for cid := range overlapClusterSet {
		cl, err := s.db.GetCluster(cid)
		if err != nil {
			return nil, err
		}
		if err := s.db.UpdateCluster(cid, cl.MainshockID, model.ClusterOverlapping, cl.Confidence); err != nil {
			return nil, err
		}
	}

	res.IsolatedCount = len(result.Isolated)
	return res, nil
}

// ResolveConflict 裁决一条边界冲突（assign_a / assign_b）。
// assign_a：事件只保留在簇 A，从簇 B 移除；assign_b 相反。
func (s *Service) ResolveConflict(conflictID int64, resolution string) error {
	if resolution != model.ResolutionAssignA && resolution != model.ResolutionAssignB {
		return fmt.Errorf("%w: unknown resolution %s", model.ErrInvalid, resolution)
	}
	target, err := s.db.GetConflict(conflictID)
	if err != nil {
		return err
	}
	if target.Status == model.ConflictResolved {
		return fmt.Errorf("%w: conflict already resolved", model.ErrConflict)
	}

	dropCluster := target.ClusterBID
	if resolution == model.ResolutionAssignB {
		dropCluster = target.ClusterAID
	}
	if err := s.db.DeleteMembership(dropCluster, target.EventID); err != nil {
		return err
	}
	return s.db.ResolveConflict(conflictID, resolution)
}
