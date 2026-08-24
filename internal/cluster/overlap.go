package cluster

import (
	"sort"
)

// OverlapPair 两个簇草稿之间的重叠关系。
type OverlapPair struct {
	ClusterA     int // 簇草稿下标
	ClusterB     int
	SharedEvents int
}

// overlapRatio 计算两个簇共享成员比例（Jaccard 相似度）。
func overlapRatio(a, b ClusterDraft) float64 {
	set := make(map[int64]bool)
	for _, e := range a.Members {
		set[e.ID] = true
	}
	shared := 0
	for _, e := range b.Members {
		if set[e.ID] {
			shared++
		}
	}
	union := len(a.Members) + len(b.Members) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}

// AnalyzeOverlaps 分析簇草稿集合的两两重叠，返回按共享成员数降序的重叠对。
func AnalyzeOverlaps(clusters []ClusterDraft) []OverlapPair {
	var pairs []OverlapPair
	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			shared := sharedMemberCount(clusters[i], clusters[j])
			if shared > 0 {
				pairs = append(pairs, OverlapPair{
					ClusterA:     i,
					ClusterB:     j,
					SharedEvents: shared,
				})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].SharedEvents < pairs[j].SharedEvents
	})
	return pairs
}

// sharedMemberCount 计算两个簇共享成员数。
func sharedMemberCount(a, b ClusterDraft) int {
	set := make(map[int64]bool)
	for _, e := range a.Members {
		set[e.ID] = true
	}
	shared := 0
	for _, e := range b.Members {
		if set[e.ID] {
			shared++
		}
	}
	return shared
}
