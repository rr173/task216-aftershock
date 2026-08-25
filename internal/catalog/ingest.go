package catalog

import (
	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
)

// IngestResult 一次批量导入的结果汇总。
type IngestResult struct {
	Accepted   int     `json:"accepted"`
	Duplicates int     `json:"duplicates"`
	Unstable   int     `json:"unstable"`
	EventIDs   []int64 `json:"event_ids"`
}

// IngestEvent 向目录导入单条事件：校验目录可写、事件合法、幂等去重。
// 导入不推进目录状态，目录需经显式 MarkAnalyzing 进入分析阶段。
// 返回 (事件, 是否为新事件, 错误)。
func (m *Manager) IngestEvent(catalogID int64, in event.EventInput) (*model.Event, bool, error) {
	if err := m.IsWritable(catalogID); err != nil {
		return nil, false, err
	}
	return event.NewManager(m.db).Ingest(catalogID, in)
}

// IngestBatch 向目录批量导入事件，逐条去重并汇总结果。
func (m *Manager) IngestBatch(catalogID int64, inputs []event.EventInput) (*IngestResult, error) {
	if err := m.IsWritable(catalogID); err != nil {
		return nil, err
	}
	em := event.NewManager(m.db)
	res := &IngestResult{}
	for _, in := range inputs {
		e, isNew, err := em.Ingest(catalogID, in)
		if err != nil {
			return nil, err
		}
		if !isNew {
			res.Duplicates++
			continue
		}
		res.Accepted++
		if e.Status == model.EventUnstable {
			res.Unstable++
		}
		res.EventIDs = append(res.EventIDs, e.ID)
	}
	return res, nil
}
