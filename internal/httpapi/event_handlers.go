package httpapi

import (
	"net/http"
	"strconv"

	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
)

type createEventReq struct {
	OriginTime string  `json:"origin_time"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	DepthKm    float64 `json:"depth_km"`
	Magnitude  float64 `json:"magnitude"`
	LocErrorH  float64 `json:"loc_error_h_km"`
	LocErrorZ  float64 `json:"loc_error_z_km"`
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	var req createEventReq
	if !decodeBody(w, r, &req) {
		return
	}
	in := event.EventInput{
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		DepthKm:   req.DepthKm,
		Magnitude: req.Magnitude,
		LocErrorH: req.LocErrorH,
		LocErrorZ: req.LocErrorZ,
	}
	if req.OriginTime != "" {
		t, err := parseTimeInput(req.OriginTime)
		if err != nil {
			writeErr(w, err)
			return
		}
		in.OriginTime = t
	}
	e, isNew, err := s.app.Event.Ingest(catalogID, in)
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
