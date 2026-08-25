package store

import (
	"time"

	"task216-aftershock/internal/model"
)

// InsertConflict 记录一条边界冲突（同目录同事件同簇对唯一，重复忽略）。
func (db *DB) InsertConflict(c *model.Conflict) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT OR IGNORE INTO conflicts (catalog_id, event_id, cluster_a_id, cluster_b_id, status, resolution, created_at, resolved_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.CatalogID, c.EventID, c.ClusterAID, c.ClusterBID, c.Status, c.Resolution, nowText(), formatTime(c.ResolvedAt),
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

// GetConflict 按 ID 读取冲突。
func (db *DB) GetConflict(id int64) (*model.Conflict, error) {
	var c model.Conflict
	var createdAt, resolvedAt string
	err := db.conn.QueryRow(
		`SELECT id, catalog_id, event_id, cluster_a_id, cluster_b_id, status, resolution, created_at, resolved_at
		 FROM conflicts WHERE id = ?`, id,
	).Scan(&c.ID, &c.CatalogID, &c.EventID, &c.ClusterAID, &c.ClusterBID,
		&c.Status, &c.Resolution, &createdAt, &resolvedAt)
	if err != nil {
		return nil, mapSQLError(err)
	}
	c.CreatedAt, _ = parseTime(createdAt)
	c.ResolvedAt, _ = parseTime(resolvedAt)
	return &c, nil
}

// ListConflictsByCatalog 列出目录下全部冲突。
func (db *DB) ListConflictsByCatalog(catalogID int64) ([]model.Conflict, error) {
	rows, err := db.conn.Query(
		`SELECT id, catalog_id, event_id, cluster_a_id, cluster_b_id, status, resolution, created_at, resolved_at
		 FROM conflicts WHERE catalog_id = ? ORDER BY id ASC`, catalogID)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []model.Conflict
	for rows.Next() {
		var c model.Conflict
		var createdAt, resolvedAt string
		if err := rows.Scan(&c.ID, &c.CatalogID, &c.EventID, &c.ClusterAID, &c.ClusterBID,
			&c.Status, &c.Resolution, &createdAt, &resolvedAt); err != nil {
			return nil, mapSQLError(err)
		}
		c.CreatedAt, _ = parseTime(createdAt)
		c.ResolvedAt, _ = parseTime(resolvedAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

// ResolveConflict 标记冲突为已裁决并记录裁决结果。
func (db *DB) ResolveConflict(id int64, resolution string) error {
	_, err := db.conn.Exec(
		`UPDATE conflicts SET status = ?, resolution = ?, resolved_at = ? WHERE id = ?`,
		model.ConflictResolved, resolution, nowText(), id)
	return mapSQLError(err)
}

// DeleteConflictsByCatalog 清空目录下的冲突记录。
func (db *DB) DeleteConflictsByCatalog(catalogID int64) error {
	_, err := db.conn.Exec(`DELETE FROM conflicts WHERE catalog_id = ?`, catalogID)
	return mapSQLError(err)
}
