package cluster

import (
	"math"
	"time"

	"task216-aftershock/internal/model"
)

// earthRadiusKm 地球平均半径（公里）。
const earthRadiusKm = 6371.0

// haversineKm 计算两点间大圆距离（公里）。
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// distanceKm 计算两个事件间的震中距离（公里），忽略深度，仅用水平坐标。
func distanceKm(a, b model.Event) float64 {
	return haversineKm(a.Latitude, a.Longitude, b.Latitude, b.Longitude)
}

// utsuDistanceKm 用宇津-关（Utsu-Seki）经验公式估计主震震级对应的余震区半径（公里）。
//
//	log10(R) = 0.5*M - 1.8  （近似，M 为主震震级）
//
// 这是余震区半径随震级增长的经典经验关系。
func utsuDistanceKm(magnitude float64) float64 {
	return math.Pow(10, 0.5*magnitude-1.8)
}

// windowDistanceKm 返回主震震级对应的距离窗（公里）。
// 若启用 Utsu 公式则用经验半径，否则用显式参数。
func (p Params) windowDistanceKm(mainshockMag float64) float64 {
	if p.UseUtsuDistance {
		return utsuDistanceKm(mainshockMag)
	}
	return p.DistanceWindowKm
}

// inTimeWindow 判断事件是否落在主震触发时间窗内（主震后 timeWindowDays 天，含主震本身）。
func (p Params) inTimeWindow(mainshock, e model.Event) bool {
	if e.OriginTime.Before(mainshock.OriginTime) {
		return false
	}
	elapsed := e.OriginTime.Sub(mainshock.OriginTime)
	return elapsed <= time.Duration(p.TimeWindowDays*24*float64(time.Hour))
}

// inMagnitudeWindow 判断事件震级是否满足余震震级下限（主震震级 - deltaMagnitude）。
func (p Params) inMagnitudeWindow(mainshock, e model.Event) bool {
	return e.Magnitude <= mainshock.Magnitude && e.Magnitude >= mainshock.Magnitude-p.DeltaMagnitude
}

// inSpatialWindow 判断事件是否落在主震距离窗内。
func (p Params) inSpatialWindow(mainshock, e model.Event) bool {
	return distanceKm(mainshock, e) <= p.windowDistanceKm(mainshock.Magnitude)
}

// isAftershockOf 判断事件是否属于某主震的余震（时间、距离、震级三个窗口同时满足）。
func (p Params) isAftershockOf(mainshock, e model.Event) bool {
	if mainshock.ID == e.ID {
		return false // 主震本身不属于自己的余震
	}
	return p.inTimeWindow(mainshock, e) &&
		p.inSpatialWindow(mainshock, e) &&
		p.inMagnitudeWindow(mainshock, e)
}

// windowMatch 描述一次窗口匹配：事件落入哪个主震窗口、震中距多少。
type windowMatch struct {
	Mainshock model.Event
	Distance  float64
}
