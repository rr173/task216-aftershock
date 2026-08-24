package cluster

import (
	"sort"

	"task216-aftershock/internal/model"
)

// candidateMainshocks 从事件列表筛选主震候选：震级达到阈值、定位稳定（非 unstable）。
// 返回按发震时间升序排列的候选主震。
func candidateMainshocks(events []model.Event, p Params) []model.Event {
	var out []model.Event
	for _, e := range events {
		if e.Status == model.EventDuplicate || e.Status == model.EventUnstable {
			continue // 去重与定位不稳事件不参与主震判定
		}
		if e.Magnitude >= p.MinMainshockMag {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OriginTime.Equal(out[j].OriginTime) {
			return out[i].ID < out[j].ID
		}
		return out[i].OriginTime.Before(out[j].OriginTime)
	})
	return out
}

// mainshockScore 为主震候选计算置信度分数：震级越高、定位误差越小，分数越高。
func mainshockScore(e model.Event, p Params) float64 {
	magScore := (e.Magnitude - p.MinMainshockMag) / (p.MinMainshockMag + 1)
	if magScore < 0 {
		magScore = 0
	}
	if magScore > 1 {
		magScore = 1
	}
	// 定位精度贡献：误差越小越接近 1。
	locErr := e.LocErrorH
	if locErr < model.MinLocErrorKm {
		locErr = model.MinLocErrorKm
	}
	locScore := 1 / (1 + locErr/10)
	return 0.7*magScore + 0.3*locScore
}
