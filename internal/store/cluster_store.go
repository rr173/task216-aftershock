package store

import (
	"time"

	"task216-aftershock/internal/model"
)

// InsertCluster 创建余震簇。
func (db *DB) InsertCluster(c *model.Cluster) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO clusters (catalog_id, mainshock_id, status, confidence, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		c.CatalogID, c.MainshockID, c.Status, c.Confidence, nowText(),
	)
	if err != nil {
		return 0, mapSQLError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, mapSQLError(err)
	}
	c.ID = id
	c.CreatedAt = time.Now().UTC()
	return id, nil
}

// GetCluster 按 ID 读取簇。
func (db *DB) GetCluster(id int64) (*model.Cluster, error) {
	var c model.Cluster
	var createdAt string
	err := db.conn.QueryRow(
		`SELECT id, catalog_id, mainshock_id, status, confidence, created_at
		 FROM clusters WHERE id = ?`, id,
	).Scan(&c.ID, &c.CatalogID, &c.MainshockID, &c.Status, &c.Confidence, &createdAt)
	if err != nil {
		return nil, mapSQLError(err)
	}
	c.CreatedAt, _ = parseTime(createdAt)
	return &c, nil
}

// ListClustersByCatalog 列出目录下的全部簇。
func (db *DB) ListClustersByCatalog(catalogID int64) ([]model.Cluster, error) {
	rows, err := db.conn.Query(
		`SELECT id, catalog_id, mainshock_id, status, confidence, created_at
		 FROM clusters WHERE catalog_id = ? ORDER BY id ASC`, catalogID)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []model.Cluster
	for rows.Next() {
		var c model.Cluster
		var createdAt string
		if err := rows.Scan(&c.ID, &c.CatalogID, &c.MainshockID, &c.Status, &c.Confidence, &createdAt); err != nil {
			return nil, mapSQLError(err)
		}
		c.CreatedAt, _ = parseTime(createdAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateCluster 更新簇的主震、状态与置信度。
func (db *DB) UpdateCluster(id, mainshockID int64, status string, confidence float64) error {
	_, err := db.conn.Exec(
		`UPDATE clusters SET mainshock_id = ?, status = ?, confidence = ? WHERE id = ?`,
		mainshockID, status, confidence, id)
	return mapSQLError(err)
}

// AddMembership 建立簇-事件关联（幂等，重复插入忽略）。
func (db *DB) AddMembership(clusterID, eventID int64, role string) error {
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO cluster_memberships (cluster_id, event_id, role) VALUES (?, ?, ?)`,
		clusterID, eventID, role)
	return mapSQLError(err)
}

// ListMembersByCluster 列出簇内全部事件关联。
func (db *DB) ListMembersByCluster(clusterID int64) ([]model.ClusterMembership, error) {
	rows, err := db.conn.Query(
		`SELECT cluster_id, event_id, role FROM cluster_memberships WHERE cluster_id = ? ORDER BY event_id ASC`, clusterID)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []model.ClusterMembership
	for rows.Next() {
		var m model.ClusterMembership
		if err := rows.Scan(&m.ClusterID, &m.EventID, &m.Role); err != nil {
			return nil, mapSQLError(err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteMembership 移除簇-事件关联（拆分/合并时重组归属）。
func (db *DB) DeleteMembership(clusterID, eventID int64) error {
	_, err := db.conn.Exec(
		`DELETE FROM cluster_memberships WHERE cluster_id = ? AND event_id = ?`, clusterID, eventID)
	return mapSQLError(err)
}

// ListEventClusters 列出事件归属的全部簇（重叠冲突判定用）。
func (db *DB) ListEventClusters(eventID int64) ([]int64, error) {
	rows, err := db.conn.Query(
		`SELECT cluster_id FROM cluster_memberships WHERE event_id = ? ORDER BY cluster_id ASC`, eventID)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var cid int64
		if err := rows.Scan(&cid); err != nil {
			return nil, mapSQLError(err)
		}
		out = append(out, cid)
	}
	return out, rows.Err()
}

// DeleteClustersByCatalog 清空目录下的簇（重新识别时）。
func (db *DB) DeleteClustersByCatalog(catalogID int64) error {
	if _, err := db.conn.Exec(`DELETE FROM cluster_memberships WHERE cluster_id IN (SELECT id FROM clusters WHERE catalog_id = ?)`, catalogID); err != nil {
		return mapSQLError(err)
	}
	if _, err := db.conn.Exec(`DELETE FROM clusters WHERE catalog_id = ?`, catalogID); err != nil {
		return mapSQLError(err)
	}
	return nil
}
