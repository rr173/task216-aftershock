package httpapi

import (
	"fmt"
	"time"

	"task216-aftershock/internal/model"
)

// parseTimeInput 解析 HTTP 层接收的 RFC3339 时间字符串。
func parseTimeInput(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: origin_time must be RFC3339", model.ErrInvalid)
	}
	return t.Truncate(time.Minute), nil
}
