package httpapi_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task216-aftershock/internal/httpapi"
	"task216-aftershock/internal/service"
	"task216-aftershock/internal/store"
)

func TestArchivedCatalogRejectsHTTPEventWrite(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "archived.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := service.New(db)
	c, err := app.Catalog.Create("catalog", "region")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Catalog.MarkAnalyzing(c.ID); err != nil {
		t.Fatal(err)
	}
	if err := app.Catalog.Publish(c.ID); err != nil {
		t.Fatal(err)
	}
	if err := app.Catalog.Archive(c.ID); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.New(app).Handler())
	defer server.Close()
	body := bytes.NewBufferString(`{"origin_time":"2024-01-01T00:00:00Z","latitude":36,"longitude":140,"depth_km":10,"magnitude":3}`)
	res, err := http.Post(server.URL+"/api/catalogs/"+fmt.Sprintf("%d", c.ID)+"/events", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusGone {
		t.Fatalf("archived write status = %d, want 410", res.StatusCode)
	}
}
