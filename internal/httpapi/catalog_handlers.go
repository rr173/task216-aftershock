package httpapi

import (
	"net/http"
	"strconv"

	"task216-aftershock/internal/model"
)

type createCatalogReq struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

func (s *Server) createCatalog(w http.ResponseWriter, r *http.Request) {
	var req createCatalogReq
	if !decodeBody(w, r, &req) {
		return
	}
	c, err := s.app.Catalog.Create(req.Name, req.Region)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) listCatalogs(w http.ResponseWriter, r *http.Request) {
	cs, err := s.app.Catalog.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) getCatalog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	c, err := s.app.Catalog.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) markAnalyzing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	if err := s.app.Catalog.MarkAnalyzing(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "analyzing"})
}

func (s *Server) publishCatalog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	if c, getErr := s.app.Catalog.Get(id); getErr == nil && c.Status == model.CatalogImporting {
		_ = s.app.Catalog.MarkAnalyzing(id)
	}
	if err := s.app.Catalog.Publish(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (s *Server) archiveCatalog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErr(w, model.ErrInvalid)
		return
	}
	if err := s.app.Catalog.Archive(id); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "archived"})
}
