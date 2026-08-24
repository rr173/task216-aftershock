package cluster

import (
	"math"
	"testing"
	"time"

	"task216-aftershock/internal/model"
)

func mkEvent(id int64, t time.Time, lat, lon, mag float64) model.Event {
	return model.Event{
		ID: id, OriginTime: t, Latitude: lat, Longitude: lon,
		Magnitude: mag, Status: model.EventValid, LocErrorH: 2, LocErrorZ: 2,
	}
}

func TestHaversine(t *testing.T) {
	// 赤道上经度差 1 度约 111.19 km。
	d := haversineKm(0, 0, 0, 1)
	if math.Abs(d-111.19) > 1.0 {
		t.Fatalf("expected ~111.19 km, got %f", d)
	}
}

func TestUtsuDistance(t *testing.T) {
	// M5 对应半径约 10^(0.5*5-1.8)=10^0.7≈5.01 km。
	r := utsuDistanceKm(5.0)
	if math.Abs(r-5.01) > 0.5 {
		t.Fatalf("expected ~5.01 km, got %f", r)
	}
}

func TestIsAftershockOf(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	p := Params{
		MinMainshockMag: 3.0, TimeWindowDays: 30, DeltaMagnitude: 1.2,
		UseUtsuDistance: false, DistanceWindowKm: 50,
	}
	ms := mkEvent(1, base, 36.0, 140.0, 5.5)

	// 时间、距离、震级均满足。
	ok := mkEvent(2, base.Add(24*time.Hour), 36.1, 140.1, 4.5)
	if !p.isAftershockOf(ms, ok) {
		t.Fatal("expected aftershock match")
	}
	// 震级低于下限。
	lowMag := mkEvent(3, base.Add(24*time.Hour), 36.1, 140.1, 3.0)
	if p.isAftershockOf(ms, lowMag) {
		t.Fatal("low magnitude should not match")
	}
	// 时间早于主震。
	early := mkEvent(4, base.Add(-24*time.Hour), 36.1, 140.1, 4.5)
	if p.isAftershockOf(ms, early) {
		t.Fatal("earlier event should not match")
	}
	// 距离超出窗口。
	far := mkEvent(5, base.Add(24*time.Hour), 40.0, 140.0, 4.5)
	if p.isAftershockOf(ms, far) {
		t.Fatal("far event should not match")
	}
}

func TestBuild_DetectsConflict(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []model.Event{
		mkEvent(1, base, 36.00, 140.00, 5.5),                    // 主震 A
		mkEvent(2, base.Add(3*24*time.Hour), 36.10, 140.10, 5.0), // 主震 B
		mkEvent(3, base.Add(5*24*time.Hour), 36.05, 140.05, 4.5), // 重叠事件 X
	}
	p := Params{
		MinMainshockMag: 3.0, TimeWindowDays: 30, DeltaMagnitude: 1.2,
		UseUtsuDistance: false, DistanceWindowKm: 50,
	}
	res, err := Build(events, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Clusters) < 2 {
		t.Fatalf("expected >=2 clusters, got %d", len(res.Clusters))
	}
	if len(res.Conflicts) < 1 {
		t.Fatalf("expected >=1 conflict, got %d", len(res.Conflicts))
	}
}

func TestBuild_IsolatedEvents(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []model.Event{
		mkEvent(1, base, 36.00, 140.00, 5.5),                    // 主震
		mkEvent(2, base.Add(200*24*time.Hour), 30.00, 130.00, 2.0), // 孤立：时间/距离都远离
	}
	p := Params{
		MinMainshockMag: 3.0, TimeWindowDays: 30, DeltaMagnitude: 1.2,
		UseUtsuDistance: false, DistanceWindowKm: 50,
	}
	res, err := Build(events, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Isolated) < 1 {
		t.Fatal("expected isolated event")
	}
}
