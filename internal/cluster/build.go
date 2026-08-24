package cluster

import (
	"sort"

	"task216-aftershock/internal/model"
)

// ClusterDraft 一次识别得到的簇草稿：主震 + 归属事件列表。
type ClusterDraft struct {
	Mainshock  model.Event
	Members    []model.Event
	Confidence float64
}

// ConflictDraft 一次识别得到的边界冲突草稿。
type ConflictDraft struct {
	Event      model.Event
	ClusterRef []int // 冲突涉及的簇草稿下标
}

// BuildResult 是簇识别的纯计算结果，尚未落库。
type BuildResult struct {
	Clusters  []ClusterDraft
	Conflicts []ConflictDraft
	Isolated  []model.Event // 孤立事件：未落入任何主震窗口
}

// Build 执行余震簇识别：筛选主震候选，按时空窗归属事件，检测重叠冲突。
func Build(events []model.Event, p Params) (*BuildResult, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	mainshocks := candidateMainshocks(events, p)
	res := &BuildResult{}

	// 每个事件归属的簇草稿下标。
	eventToClusters := make(map[int64][]int)

	for mi, ms := range mainshocks {
		draft := ClusterDraft{
			Mainshock:  ms,
			Confidence: mainshockScore(ms, p),
			Members:    []model.Event{ms},
		}
		for _, e := range events {
			if e.ID == ms.ID {
				continue
			}
			if p.isAftershockOf(ms, e) {
				draft.Members = append(draft.Members, e)
				eventToClusters[e.ID] = append(eventToClusters[e.ID], mi)
			}
		}
		// 成员按发震时间排序，主震永远排第一。
		sort.Slice(draft.Members, func(i, j int) bool {
			return draft.Members[i].OriginTime.Before(draft.Members[j].OriginTime)
		})
		res.Clusters = append(res.Clusters, draft)
	}

	// 重叠冲突：一个事件落入多个簇窗口。
	for eid, refs := range eventToClusters {
		if len(refs) < 2 {
			continue
		}
		// 找该事件的完整记录。
		var ev model.Event
		found := false
		for _, e := range events {
			if e.ID == eid {
				ev = e
				found = true
				break
			}
		}
		if !found {
			continue
		}
		reversed := append([]int(nil), refs...)
		for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
			reversed[i], reversed[j] = reversed[j], reversed[i]
		}
		res.Conflicts = append(res.Conflicts, ConflictDraft{
			Event:      ev,
			ClusterRef: reversed,
		})
	}

	// 孤立事件：有效事件未落入任何簇（且自身不是主震）。
	inMainshock := make(map[int64]bool)
	for _, ms := range mainshocks {
		inMainshock[ms.ID] = true
	}
	for _, e := range events {
		if e.Status == model.EventDuplicate {
			continue
		}
		if inMainshock[e.ID] {
			continue
		}
		if refs, ok := eventToClusters[e.ID]; ok && len(refs) > 0 {
			continue
		}
		res.Isolated = append(res.Isolated, e)
	}

	return res, nil
}

// TotalMembers 返回簇草稿成员数。
func (d ClusterDraft) TotalMembers() int { return len(d.Members) }

// AftershockCount 返回簇内余震（不含主震）数量。
func (d ClusterDraft) AftershockCount() int {
	if len(d.Members) == 0 {
		return 0
	}
	return len(d.Members) - 1
}
