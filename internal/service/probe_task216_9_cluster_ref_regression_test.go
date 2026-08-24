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

func TestIdentifyClustersRejectsInvalidConflictReference(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "cluster-ref.db"))
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
	if _, err := app.IdentifyClusters(c.ID, model.ClusterInput{MinMainshockMag: 3, TimeWindowDays: 30, DeltaMagnitude: 1.2, DistanceWindowKm: 50}); err != nil {
		t.Fatal(err)
	}
}
