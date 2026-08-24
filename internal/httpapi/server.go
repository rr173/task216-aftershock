// Package httpapi 提供地震余震簇识别服务的 HTTP 接口（路由前缀 /api）。
package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"task216-aftershock/internal/model"
	"task216-aftershock/internal/service"
)

// Server 封装 HTTP 路由与处理器。
type Server struct {
	app *service.Service
	mux *http.ServeMux
}

// New 构造 HTTP 服务并注册全部路由。
func New(app *service.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回底层 http.Handler。
func (s *Server) Handler() http.Handler {
	return logMiddleware(s.mux)
}

// routes 注册全部 API 路由（前缀 /api）。
func (s *Server) routes() {
	m := s.mux

	// 目录。
	m.HandleFunc("POST /api/catalogs", s.createCatalog)
	m.HandleFunc("GET /api/catalogs", s.listCatalogs)
	m.HandleFunc("GET /api/catalogs/{id}", s.getCatalog)
	m.HandleFunc("POST /api/catalogs/{id}/analyzing", s.markAnalyzing)
	m.HandleFunc("POST /api/catalogs/{id}/publish", s.publishCatalog)
	m.HandleFunc("POST /api/catalogs/{id}/archive", s.archiveCatalog)

	// 事件。
	m.HandleFunc("POST /api/catalogs/{id}/events", s.createEvent)
	m.HandleFunc("GET /api/catalogs/{id}/events", s.listEvents)
	m.HandleFunc("GET /api/events/{id}", s.getEvent)
	m.HandleFunc("POST /api/events/{id}/mark-unstable", s.markUnstable)
	m.HandleFunc("POST /api/events/{id}/mark-valid", s.markValid)

	// 簇识别与复核。
	m.HandleFunc("POST /api/catalogs/{id}/cluster", s.identifyClusters)
	m.HandleFunc("GET /api/catalogs/{id}/clusters", s.listClusters)
	m.HandleFunc("GET /api/clusters/{id}", s.getCluster)
	m.HandleFunc("GET /api/clusters/{id}/members", s.listClusterMembers)
	m.HandleFunc("POST /api/clusters/{id}/lock-mainshock", s.lockMainshock)
	m.HandleFunc("POST /api/clusters/{id}/split", s.splitCluster)
	m.HandleFunc("POST /api/clusters/{id}/merge", s.mergeClusters)

	// 冲突。
	m.HandleFunc("GET /api/catalogs/{id}/conflicts", s.listConflicts)
	m.HandleFunc("POST /api/conflicts/{id}/resolve", s.resolveConflict)

	// 版本。
	m.HandleFunc("POST /api/versions", s.createVersion)
	m.HandleFunc("GET /api/versions", s.listVersions)
	m.HandleFunc("GET /api/versions/{id}", s.getVersion)
	m.HandleFunc("POST /api/versions/{id}/publish", s.publishVersion)

	// 统计与健康。
	m.HandleFunc("GET /api/stats", s.stats)
	m.HandleFunc("GET /api/health", s.health)
}

// writeJSON 序列化 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr 序列化错误响应并映射状态码。
func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrConflict), errors.Is(err, model.ErrDuplicate):
		status = http.StatusConflict
	case errors.Is(err, model.ErrArchived):
		status = http.StatusGone
	case errors.Is(err, model.ErrUnstable):
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeBody 解析 JSON 请求体。
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		writeErr(w, err)
		return false
	}
	return true
}

// logMiddleware 记录请求方法、路径与状态。
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		log.Printf("%s %s", r.Method, r.URL.Path)
	})
}
