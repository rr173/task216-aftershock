// Package event 负责地震事件的校验与幂等去重。
package event

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"task216-aftershock/internal/model"
)

// EventInput 是外部提交事件时的原始输入。
type EventInput struct {
	OriginTime time.Time `json:"origin_time"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	DepthKm    float64   `json:"depth_km"`
	Magnitude  float64   `json:"magnitude"`
	LocErrorH  float64   `json:"loc_error_h_km"`
	LocErrorZ  float64   `json:"loc_error_z_km"`
}

// Validate 校验一条事件输入，返回错误说明（nil 表示合法）。
// 规则：坐标在合法经纬度范围、深度与震级在合理区间、时间非零、定位误差非负。
func Validate(in EventInput) error {
	if in.OriginTime.IsZero() {
		return fmt.Errorf("%w: origin_time is required", model.ErrInvalid)
	}
	if in.Latitude < -90 || in.Latitude > 90 {
		return fmt.Errorf("%w: latitude out of range", model.ErrInvalid)
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return fmt.Errorf("%w: longitude out of range", model.ErrInvalid)
	}
	if in.DepthKm < 0 || in.DepthKm > 800 {
		return fmt.Errorf("%w: depth out of range", model.ErrInvalid)
	}
	if in.Magnitude < -2 || in.Magnitude > 10 {
		return fmt.Errorf("%w: magnitude out of range", model.ErrInvalid)
	}
	if in.LocErrorH < 0 || in.LocErrorZ < 0 {
		return fmt.Errorf("%w: location error must be non-negative", model.ErrInvalid)
	}
	return nil
}

// LocUnstable 判断定位是否不稳：水平误差超过阈值或垂直误差异常。
func LocUnstable(in EventInput, maxErrH, maxErrZ float64) bool {
	return in.LocErrorH > maxErrH || in.LocErrorZ > maxErrZ
}

// Fingerprint 计算事件幂等指纹：时间取整秒、坐标取 4 位小数、深度/震级取 1 位小数。
func Fingerprint(in EventInput) string {
	t := in.OriginTime.UTC().Truncate(time.Second)
	lat := round(in.Latitude, 4)
	lon := round(in.Longitude, 4)
	depth := round(in.DepthKm, 1)
	mag := round(in.Magnitude, 1)
	raw := fmt.Sprintf("%s|%.4f|%.4f|%.1f|%.1f", t.Format(time.RFC3339), lat, lon, depth, mag)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// round 四舍五入到指定小数位，处理负零。
func round(v float64, decimals int) float64 {
	shift := math.Pow10(decimals)
	r := math.Round(v*shift) / shift
	if r == 0 {
		return 0
	}
	return r
}
