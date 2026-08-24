package httpapi

import (
	"net/http"
	"strconv"

	"task216-aftershock/internal/model"
)

type identifyReq struct {
	MinMainshockMag  float64 `json:"min_mainshock_mag"`
	TimeWindowDays   float64 `json:"time_window_days"`
	DeltaMagnitude   float64 `json:"delta_magnitude"`
	UseUtsuDistance  bool    `json:"use_utsu_distance"`
	DistanceWindowKm float64 `json:"distance_window_km"`
}

func (s *Server) identifyClusters(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	var req identifyReq
	if !decodeBody(w, r, &req) {
		return
	}
	res, err := s.app.IdentifyClusters(catalogID, model.ClusterInput{
		MinMainshockMag:  req.MinMainshockMag,
		TimeWindowDays:   req.TimeWindowDays,
		DeltaMagnitude:   req.DeltaMagnitude,
		UseUtsuDistance:  req.UseUtsuDistance,
		DistanceWindowKm: req.DistanceWindowKm,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) listClusters(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	cs, err := s.app.Review.ListClusters(catalogID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) getCluster(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	c, err := s.app.Review.GetCluster(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) listClusterMembers(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	ms, err := s.app.Review.ListMembers(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

type lockMainshockReq struct {
	EventID int64 `json:"event_id"`
}

func (s *Server) lockMainshock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	var req lockMainshockReq
	if !decodeBody(w, r, &req) {
		return
	}
	c, err := s.app.Review.LockMainshock(id, req.EventID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type splitReq struct {
	EventIDs []int64 `json:"event_ids"`
}

func (s *Server) splitCluster(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	var req splitReq
	if !decodeBody(w, r, &req) {
		return
	}
	nc, err := s.app.Review.SplitCluster(id, req.EventIDs)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nc)
}

type mergeReq struct {
	OtherClusterID int64 `json:"other_cluster_id"`
}

func (s *Server) mergeClusters(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	var req mergeReq
	if !decodeBody(w, r, &req) {
		return
	}
	c, err := s.app.Review.MergeClusters(id, req.OtherClusterID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) listConflicts(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	confs, err := s.app.DB().ListConflictsByCatalog(catalogID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, confs)
}

type resolveReq struct {
	Resolution string `json:"resolution"`
}

func (s *Server) resolveConflict(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	var req resolveReq
	if !decodeBody(w, r, &req) {
		return
	}
	if err := s.app.ResolveConflict(id, req.Resolution); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}
