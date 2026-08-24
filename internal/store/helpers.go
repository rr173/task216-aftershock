package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"task216-aftershock/internal/model"
)

// mapSQLError 把底层 SQL 错误映射为业务错误。
func mapSQLError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	return err
}

// isUniqueViolation 判断是否为 UNIQUE 约束冲突。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// formatTime 将时间序列化为可存储的 RFC3339Nano 文本。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTime 解析存储的时间文本；空串返回零值。
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, s)
}

// nowText 返回当前时间的 RFC3339Nano 文本。
func nowText() string { return time.Now().UTC().Format(time.RFC3339Nano) }
