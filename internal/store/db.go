// Package store 提供基于 SQLite 的持久化层（modernc.org/sqlite，纯 Go 无 CGO）。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB 封装数据库连接与建表迁移。
type DB struct {
	conn *sql.DB
	path string
}

// Open 打开（必要时创建）SQLite 数据库并执行建表迁移。
func Open(path string) (*DB, error) {
	if path == "" {
		path = filepath.Join(".", "aftershock.db")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1) // 单写者，规避 SQLite 写锁竞争
	db := &DB{conn: conn, path: path}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close 关闭数据库连接。
func (db *DB) Close() error { return db.conn.Close() }

// Path 返回数据库文件路径。
func (db *DB) Path() string { return db.path }

// Conn 暴露底层连接，供需要事务或裸查询的场景使用。
func (db *DB) Conn() *sql.DB { return db.conn }

// migrate 执行全部建表语句（幂等）。
func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS catalogs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			region TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			published_at TEXT,
			archived_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_catalogs_status ON catalogs(status)`,

		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_id INTEGER NOT NULL,
			origin_time TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			depth_km REAL NOT NULL DEFAULT 0,
			magnitude REAL NOT NULL DEFAULT 0,
			loc_error_h REAL NOT NULL DEFAULT 0,
			loc_error_z REAL NOT NULL DEFAULT 0,
			fingerprint TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(catalog_id, fingerprint),
			FOREIGN KEY (catalog_id) REFERENCES catalogs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_catalog ON events(catalog_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_time ON events(catalog_id, origin_time)`,

		`CREATE TABLE IF NOT EXISTS clusters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_id INTEGER NOT NULL,
			mainshock_id INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			confidence REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY (catalog_id) REFERENCES catalogs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_clusters_catalog ON clusters(catalog_id)`,

		`CREATE TABLE IF NOT EXISTS cluster_memberships (
			cluster_id INTEGER NOT NULL,
			event_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			PRIMARY KEY (cluster_id, event_id),
			FOREIGN KEY (cluster_id) REFERENCES clusters(id),
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memberships_event ON cluster_memberships(event_id)`,

		`CREATE TABLE IF NOT EXISTS conflicts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_id INTEGER NOT NULL,
			event_id INTEGER NOT NULL,
			cluster_a_id INTEGER NOT NULL,
			cluster_b_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			resolution TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			resolved_at TEXT,
			UNIQUE(catalog_id, event_id, cluster_a_id, cluster_b_id),
			FOREIGN KEY (catalog_id) REFERENCES catalogs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conflicts_catalog ON conflicts(catalog_id)`,

		`CREATE TABLE IF NOT EXISTS versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			label TEXT NOT NULL,
			status TEXT NOT NULL,
			snapshot_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			frozen_at TEXT,
			superseded_by INTEGER NOT NULL DEFAULT 0,
			UNIQUE(catalog_id, number),
			FOREIGN KEY (catalog_id) REFERENCES catalogs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_catalog ON versions(catalog_id)`,
	}
	for _, s := range stmts {
		if _, err := db.conn.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
