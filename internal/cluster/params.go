package cluster

import (
	"fmt"

	"task216-aftershock/internal/model"
)

// Params 聚类参数：主震触发窗口的阈值配置。
type Params struct {
	MinMainshockMag  float64
	TimeWindowDays   float64
	DeltaMagnitude   float64
	UseUtsuDistance  bool
	DistanceWindowKm float64
}

// DefaultParams 返回默认聚类参数。
func DefaultParams() Params {
	return Params{
		MinMainshockMag:  model.DefaultMinMainshockMag,
		TimeWindowDays:   model.DefaultTimeWindowDays,
		DeltaMagnitude:   model.DefaultDeltaMagnitude,
		UseUtsuDistance:  true,
		DistanceWindowKm: 0,
	}
}

// Validate 校验参数合法性。
func (p Params) Validate() error {
	if p.MinMainshockMag <= 0 {
		return fmt.Errorf("%w: min_mainshock_mag must be positive", model.ErrInvalid)
	}
	if p.TimeWindowDays <= 0 || p.TimeWindowDays > 3650 {
		return fmt.Errorf("%w: time_window_days out of range", model.ErrInvalid)
	}
	if p.DeltaMagnitude <= 0 || p.DeltaMagnitude > 5 {
		return fmt.Errorf("%w: delta_magnitude out of range", model.ErrInvalid)
	}
	if !p.UseUtsuDistance && p.DistanceWindowKm <= 0 {
		return fmt.Errorf("%w: distance_window_km must be positive when utsu disabled", model.ErrInvalid)
	}
	return nil
}

// FromInput 从外部输入构造参数（未提供的字段用默认值补齐）。
func FromInput(in model.ClusterInput) Params {
	p := DefaultParams()
	if in.MinMainshockMag > 0 {
		p.MinMainshockMag = in.MinMainshockMag
	}
	if in.TimeWindowDays > 0 {
		p.TimeWindowDays = in.TimeWindowDays
	}
	if in.DeltaMagnitude > 0 {
		p.DeltaMagnitude = in.DeltaMagnitude
	}
	if !in.UseUtsuDistance && in.DistanceWindowKm > 0 {
		p.UseUtsuDistance = false
		p.DistanceWindowKm = in.DistanceWindowKm
	}
	if in.UseUtsuDistance {
		p.UseUtsuDistance = true
	}
	return p
}
