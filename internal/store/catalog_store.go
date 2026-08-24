package store

import (
	"time"

	"task216-aftershock/internal/model"
)

// InsertCatalog 创建地震目录，返回 ID。
func (db *DB) InsertCatalog(c *model.Catalog) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO catalogs (name, region, status, created_at, published_at, archived_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.Name, c.Region, c.Status, nowText(), formatTime(c.PublishedAt), formatTime(c.ArchivedAt),
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

// GetCatalog 按 ID 读取目录。
func (db *DB) GetCatalog(id int64) (*model.Catalog, error) {
	var c model.Catalog
	var createdAt, publishedAt, archivedAt string
	err := db.conn.QueryRow(
		`SELECT id, name, region, status, created_at, published_at, archived_at
		 FROM catalogs WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Region, &c.Status, &createdAt, &publishedAt, &archivedAt)
	if err != nil {
		return nil, mapSQLError(err)
	}
	c.CreatedAt, _ = parseTime(createdAt)
	c.PublishedAt, _ = parseTime(publishedAt)
	c.ArchivedAt, _ = parseTime(archivedAt)
	return &c, nil
}

// ListCatalogs 列出全部目录（按 ID 降序）。
func (db *DB) ListCatalogs() ([]model.Catalog, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, region, status, created_at, published_at, archived_at
		 FROM catalogs ORDER BY id DESC`)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []model.Catalog
	for rows.Next() {
		var c model.Catalog
		var createdAt, publishedAt, archivedAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.Region, &c.Status, &createdAt, &publishedAt, &archivedAt); err != nil {
			return nil, mapSQLError(err)
		}
		c.CreatedAt, _ = parseTime(createdAt)
		c.PublishedAt, _ = parseTime(publishedAt)
		c.ArchivedAt, _ = parseTime(archivedAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateCatalogStatus 更新目录状态（带状态机守卫由上层完成）。
func (db *DB) UpdateCatalogStatus(id int64, status string, publish, archive bool) error {
	publishedAt := ""
	archivedAt := ""
	if publish {
		publishedAt = nowText()
	}
	if archive {
		archivedAt = nowText()
	}
	_, err := db.conn.Exec(
		`UPDATE catalogs SET status = ?,
		   published_at = CASE WHEN ? THEN ? ELSE published_at END,
		   archived_at = CASE WHEN ? THEN ? ELSE archived_at END
		 WHERE id = ?`,
		status, publish, publishedAt, archive, archivedAt, id)
	return mapSQLError(err)
}
