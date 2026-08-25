package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/event"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/service"
	"task216-aftershock/internal/store"
)

func TestAssignBLeavesEventInClusterB(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := service.New(db)
	c, err := app.Catalog.Create("catalog", "region")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = app.Catalog.IngestBatch(c.ID, []event.EventInput{
		{OriginTime: base, Latitude: 36, Longitude: 140, Magnitude: 5.5},
		{OriginTime: base.Add(3 * 24 * time.Hour), Latitude: 36.1, Longitude: 140.1, Magnitude: 5},
		{OriginTime: base.Add(5 * 24 * time.Hour), Latitude: 36.05, Longitude: 140.05, Magnitude: 4.5},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := app.IdentifyClusters(c.ID, model.ClusterInput{MinMainshockMag: 3, TimeWindowDays: 30, DeltaMagnitude: 1.2, DistanceWindowKm: 50})
	if err != nil {
		t.Fatal(err)
	}
	conflicts, err := db.ListConflictsByCatalog(c.ID)
	if err != nil || len(conflicts) != 1 {
		t.Fatalf("conflicts = %d, err = %v", len(conflicts), err)
	}
	conflict := conflicts[0]
	if err := app.ResolveConflict(conflict.ID, model.ResolutionAssignB); err != nil {
		t.Fatal(err)
	}
	aMembers, err := db.ListMembersByCluster(result.ClusterIDs[0])
	if err != nil {
		t.Fatal(err)
	}
	bMembers, err := db.ListMembersByCluster(result.ClusterIDs[1])
	if err != nil {
		t.Fatal(err)
	}
	if containsEvent(aMembers, conflict.EventID) || !containsEvent(bMembers, conflict.EventID) {
		t.Fatalf("assign_b membership mismatch: A=%v B=%v event=%d", aMembers, bMembers, conflict.EventID)
	}
}

func containsEvent(ms []model.ClusterMembership, id int64) bool {
	for _, m := range ms {
		if m.EventID == id {
			return true
		}
	}
	return false
}
