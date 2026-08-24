package httpapi

import (
	"net/http"
	"strconv"

	"task216-aftershock/internal/model"
)

type createVersionReq struct {
	CatalogID int64  `json:"catalog_id"`
	Label     string `json:"label"`
}

func (s *Server) createVersion(w http.ResponseWriter, r *http.Request) {
	var req createVersionReq
	if !decodeBody(w, r, &req) {
		return
	}
	v, err := s.app.Versioning.CreateDraft(req.CatalogID, req.Label)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	catalogID, err := strconv.ParseInt(r.URL.Query().Get("catalog_id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	vs, err := s.app.Versioning.List(catalogID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vs)
}

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	v, err := s.app.Versioning.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) publishVersion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	v, err := s.app.Versioning.Publish(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
