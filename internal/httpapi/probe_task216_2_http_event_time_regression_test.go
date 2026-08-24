package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"task216-aftershock/internal/httpapi"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/service"
	"task216-aftershock/internal/store"
)

func TestHTTPEventPreservesOriginTime(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "event.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	catalog, err := service.New(db).Catalog.Create("catalog", "region")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(httpapi.New(service.New(db)).Handler())
	defer server.Close()
	want := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	payload := map[string]any{"origin_time": want.Format(time.RFC3339), "latitude": 36, "longitude": 140, "depth_km": 10, "magnitude": 4.5}
	data, _ := json.Marshal(payload)
	res, err := http.Post(server.URL+"/api/catalogs/"+idText(catalog.ID)+"/events", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create event status = %d", res.StatusCode)
	}
	var got model.Event
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.OriginTime.Equal(want) {
		t.Fatalf("origin time = %s, want %s", got.OriginTime, want)
	}
}

func idText(id int64) string { return fmt.Sprintf("%d", id) }
