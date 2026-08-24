package store

import (
	"time"

	"task216-aftershock/internal/model"
)

// InsertEvent 创建地震事件（catalog_id + fingerprint 唯一，重复则返回 ErrDuplicate）。
func (db *DB) InsertEvent(e *model.Event) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO events (catalog_id, origin_time, latitude, longitude, depth_km, magnitude,
		   loc_error_h, loc_error_z, fingerprint, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.CatalogID, formatTime(e.OriginTime), e.Latitude, e.Longitude, e.DepthKm, e.Magnitude,
		e.LocErrorH, e.LocErrorZ, e.Fingerprint, e.Status, nowText(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, model.ErrDuplicate
		}
		return 0, mapSQLError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, mapSQLError(err)
	}
	e.ID = id
	e.CreatedAt = time.Now().UTC()
	return id, nil
}

// GetEvent 按 ID 读取事件。
func (db *DB) GetEvent(id int64) (*model.Event, error) {
	var e model.Event
	var originTime, createdAt string
	err := db.conn.QueryRow(
		`SELECT id, catalog_id, origin_time, latitude, longitude, depth_km, magnitude,
		   loc_error_h, loc_error_z, fingerprint, status, created_at
		 FROM events WHERE id = ?`, id,
	).Scan(&e.ID, &e.CatalogID, &originTime, &e.Latitude, &e.Longitude, &e.DepthKm, &e.Magnitude,
		&e.LocErrorH, &e.LocErrorZ, &e.Fingerprint, &e.Status, &createdAt)
	if err != nil {
		return nil, mapSQLError(err)
	}
	e.OriginTime, _ = parseTime(originTime)
	e.CreatedAt, _ = parseTime(createdAt)
	return &e, nil
}

// ListEventsByCatalog 列出目录下的全部事件（按发震时间升序）。
func (db *DB) ListEventsByCatalog(catalogID int64) ([]model.Event, error) {
	rows, err := db.conn.Query(
		`SELECT id, catalog_id, origin_time, latitude, longitude, depth_km, magnitude,
		   loc_error_h, loc_error_z, fingerprint, status, created_at
		 FROM events WHERE catalog_id = ? ORDER BY origin_time ASC, id ASC`, catalogID)
	if err != nil {
		return nil, mapSQLError(err)
	}
	defer rows.Close()

	var out []model.Event
	for rows.Next() {
		var e model.Event
		var originTime, createdAt string
		if err := rows.Scan(&e.ID, &e.CatalogID, &originTime, &e.Latitude, &e.Longitude, &e.DepthKm, &e.Magnitude,
			&e.LocErrorH, &e.LocErrorZ, &e.Fingerprint, &e.Status, &createdAt); err != nil {
			return nil, mapSQLError(err)
		}
		e.OriginTime, _ = parseTime(originTime)
		e.CreatedAt, _ = parseTime(createdAt)
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateEventStatus 更新事件状态（定位不稳、去重等）。
func (db *DB) UpdateEventStatus(id int64, status string) error {
	_, err := db.conn.Exec(`UPDATE events SET status = ? WHERE id = ?`, status, id)
	return mapSQLError(err)
}

// FindEventByFingerprint 按目录 + 指纹查找事件，用于幂等判定。
func (db *DB) FindEventByFingerprint(catalogID int64, fp string) (*model.Event, error) {
	var e model.Event
	var originTime, createdAt string
	err := db.conn.QueryRow(
		`SELECT id, catalog_id, origin_time, latitude, longitude, depth_km, magnitude,
		   loc_error_h, loc_error_z, fingerprint, status, created_at
		 FROM events WHERE catalog_id = ? AND fingerprint = ?`, catalogID, fp,
	).Scan(&e.ID, &e.CatalogID, &originTime, &e.Latitude, &e.Longitude, &e.DepthKm, &e.Magnitude,
		&e.LocErrorH, &e.LocErrorZ, &e.Fingerprint, &e.Status, &createdAt)
	if err != nil {
		return nil, mapSQLError(err)
	}
	e.OriginTime, _ = parseTime(originTime)
	e.CreatedAt, _ = parseTime(createdAt)
	return &e, nil
}
