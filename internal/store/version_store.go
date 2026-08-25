package store

import (
	"time"

	"task216-aftershock/internal/model"
)

// InsertVersion 创建目录版本。
func (db *DB) InsertVersion(v *model.Version) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO versions (catalog_id, number, label, status, snapshot_hash, created_at, frozen_at, superseded_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		v.CatalogID, v.Number, v.Label, v.Status, v.SnapshotHash, nowText(), formatTime(v.FrozenAt), v.SupersededBy,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, model.ErrConflict
		}
		return 0, mapSQLError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, mapSQLError(err)
	}
	v.ID = id
	v.CreatedAt = time.Now().UTC()
	return id, nil
}

// GetVersion 按 ID 读取版本。
func (db *DB) GetVersion(id int64) (*model.Version, error) {
	var v model.Version
	var createdAt, frozenAt string
	err := db.conn.QueryRow(
		`SELECT id, catalog_id, number, label, status, snapshot_hash, created_at, frozen_at, superseded_by
		 FROM versions WHERE id = ?`, id,
	).Scan(&v.ID, &v.CatalogID, &v.Number, &v.Label, &v.Status, &v.SnapshotHash, &createdAt, &frozenAt, &v.SupersededBy)
	if err != nil {
		return nil, mapSQLError(err)
	}
	v.CreatedAt, _ = parseTime(createdAt)
	v.FrozenAt, _ = parseTime(frozenAt)
	return &v, nil
}

// ListVersionsByCatalog 列出目录下的全部版本。
func (db *DB) ListVersionsByCatalog(catalogID int64) ([]model.Version, error) {
	rows, err := db.conn.Query(
		`SELECT id, catalog_id, number, label, status, snapshot_hash, created_at, frozen_at, superseded_by
		 FROM versions WHERE catalog_id = ? ORDER BY number ASC`, catalogID)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []model.Version
	for rows.Next() {
		var v model.Version
		var createdAt, frozenAt string
		if err := rows.Scan(&v.ID, &v.CatalogID, &v.Number, &v.Label, &v.Status, &v.SnapshotHash, &createdAt, &frozenAt, &v.SupersededBy); err != nil {
			return nil, mapSQLError(err)
		}
		v.CreatedAt, _ = parseTime(createdAt)
		v.FrozenAt, _ = parseTime(frozenAt)
		out = append(out, v)
	}
	return out, rows.Err()
}

// NextVersionNumber 返回目录下一个版本号（现有最大 + 1）。
func (db *DB) NextVersionNumber(catalogID int64) (int, error) {
	var n int
	err := db.conn.QueryRow(
		`SELECT COALESCE(MAX(number), 0) FROM versions WHERE catalog_id = ?`, catalogID).Scan(&n)
	if err != nil {
		return 0, mapSQLError(err)
	}
	return n + 1, nil
}

// PublishVersion 冻结版本（draft → published）。
func (db *DB) PublishVersion(id int64) error {
	_, err := db.conn.Exec(
		`UPDATE versions SET status = ?, frozen_at = ? WHERE id = ?`,
		model.VersionPublished, nowText(), id)
	return mapSQLError(err)
}

// SupersedeVersions 把目录下除 exceptID 外的其它版本标记为已被 supersededBy 替代。
// exceptID 是新发布（保留为活动）的版本，其余版本（无论 draft/published）一律转为 superseded。
func (db *DB) SupersedeVersions(catalogID int64, exceptID int64, supersededBy int64) error {
	_, err := db.conn.Exec(
		`UPDATE versions SET status = ?, superseded_by = ? WHERE catalog_id = ? AND id != ?`,
		model.VersionSuperseded, supersededBy, catalogID, exceptID)
	return mapSQLError(err)
}

// GetActivePublishedVersion 返回目录当前活动（已发布且未被替代）版本。
// 同一目录最多存在一个 published 版本；按 number DESC 取最新，保证可追溯。
func (db *DB) GetActivePublishedVersion(catalogID int64) (*model.Version, error) {
	var v model.Version
	var createdAt, frozenAt string
	err := db.conn.QueryRow(
		`SELECT id, catalog_id, number, label, status, snapshot_hash, created_at, frozen_at, superseded_by
		 FROM versions WHERE catalog_id = ? AND status = ? ORDER BY number DESC LIMIT 1`,
		catalogID, model.VersionPublished,
	).Scan(&v.ID, &v.CatalogID, &v.Number, &v.Label, &v.Status, &v.SnapshotHash, &createdAt, &frozenAt, &v.SupersededBy)
	if err != nil {
		return nil, mapSQLError(err)
	}
	v.CreatedAt, _ = parseTime(createdAt)
	v.FrozenAt, _ = parseTime(frozenAt)
	return &v, nil
}
