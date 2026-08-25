package httpapi

import (
	"net/http"
	"strconv"

	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
)

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	// 直接解码到 EventInput：其 OriginTime 为 time.Time，encoding/json 经
	// time.Time.UnmarshalJSON 按 RFC3339（含可选小数秒）解析并原样保留精度，
	// 避免字符串中转造成的截断或丢失。
	var in event.EventInput
	if !decodeBody(w, r, &in) {
		return
	}
	e, isNew, err := s.app.Catalog.IngestEvent(catalogID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	status := http.StatusCreated
	if !isNew {
		status = http.StatusOK // 幂等重复，返回已有事件
	}
	writeJSON(w, status, e)
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	es, err := s.app.Event.List(catalogID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, es)
}

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	e, err := s.app.Event.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) markUnstable(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	if err := s.app.Event.MarkUnstable(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unstable"})
}

func (s *Server) markValid(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	if err := s.app.Event.MarkValid(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "valid"})
}
