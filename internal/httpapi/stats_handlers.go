package httpapi

import (
	"net/http"
)

// stats 返回服务全局统计：目录、事件、簇、冲突、版本数量。
func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	conn := s.app.DB().Conn()
	var catalogs, events, clusters, conflicts, versions int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM catalogs`).Scan(&catalogs)
	_ = conn.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&events)
	_ = conn.QueryRow(`SELECT COUNT(*) FROM clusters`).Scan(&clusters)
	_ = conn.QueryRow(`SELECT COUNT(*) FROM conflicts`).Scan(&conflicts)
	_ = conn.QueryRow(`SELECT COUNT(*) FROM versions`).Scan(&versions)

	writeJSON(w, http.StatusOK, map[string]int{
		"catalogs":  catalogs,
		"events":    events,
		"clusters":  clusters,
		"conflicts": conflicts,
		"versions":  versions,
	})
}

// health 返回健康检查结果。
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
