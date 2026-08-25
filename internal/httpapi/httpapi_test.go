package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task216-aftershock/internal/httpapi"
	"task216-aftershock/internal/model"
	"task216-aftershock/internal/service"
	"task216-aftershock/internal/store"
)

func TestHTTPRoutesPersistCatalogLifecycle(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	server := httptest.NewServer(httpapi.New(service.New(db)).Handler())
	defer server.Close()

	body := bytes.NewBufferString(`{"name":"catalog-a","region":"tohoku"}`)
	res, err := http.Post(server.URL+"/api/catalogs", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		res.Body.Close()
		t.Fatalf("create catalog status = %d", res.StatusCode)
	}
	var catalog model.Catalog
	if err := json.NewDecoder(res.Body).Decode(&catalog); err != nil {
		res.Body.Close()
		t.Fatal(err)
	}
	res.Body.Close()
	if catalog.ID == 0 || catalog.Status != model.CatalogImporting {
		t.Fatalf("unexpected created catalog: %+v", catalog)
	}

	for _, path := range []string{
		"/api/catalogs/" + formatID(catalog.ID) + "/analyzing",
		"/api/catalogs/" + formatID(catalog.ID) + "/publish",
		"/api/catalogs/" + formatID(catalog.ID) + "/archive",
	} {
		request, err := http.NewRequest(http.MethodPost, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			t.Fatalf("POST %s status = %d", path, response.StatusCode)
		}
		response.Body.Close()
	}

	response, err := http.Get(server.URL + "/api/catalogs/" + formatID(catalog.ID))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get catalog status = %d", response.StatusCode)
	}
	var got model.Catalog
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Status != model.CatalogArchived {
		t.Fatalf("catalog status after lifecycle = %q", got.Status)
	}
}

func formatID(id int64) string {
	return fmt.Sprintf("%d", id)
}

// TestHTTPCreateEventRejectedOnArchivedCatalog 验证封存目录只读：
// 通过 HTTP 向已封存目录创建事件应返回 410 Gone，且不写入任何新事件。
func TestHTTPCreateEventRejectedOnArchivedCatalog(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "archived.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	server := httptest.NewServer(httpapi.New(service.New(db)).Handler())
	defer server.Close()

	// 创建目录并推进到封存。
	body := bytes.NewBufferString(`{"name":"catalog-archived","region":"tohoku"}`)
	res, err := http.Post(server.URL+"/api/catalogs", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	var catalog model.Catalog
	if err := json.NewDecoder(res.Body).Decode(&catalog); err != nil {
		res.Body.Close()
		t.Fatal(err)
	}
	res.Body.Close()

	for _, path := range []string{
		"/api/catalogs/" + formatID(catalog.ID) + "/analyzing",
		"/api/catalogs/" + formatID(catalog.ID) + "/publish",
		"/api/catalogs/" + formatID(catalog.ID) + "/archive",
	} {
		req, err := http.NewRequest(http.MethodPost, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("POST %s status = %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 封存后创建事件应被拒绝。
	body = bytes.NewBufferString(`{"origin_time":"2024-01-20T00:00:00Z","latitude":36.0,"longitude":140.0,"depth_km":10,"magnitude":3.0}`)
	res, err = http.Post(server.URL+"/api/catalogs/"+formatID(catalog.ID)+"/events", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusGone {
		t.Fatalf("create event on archived catalog status = %d, want 410 Gone", res.StatusCode)
	}

	// 事件列表应为空，确认未写入。
	listResp, err := http.Get(server.URL + "/api/catalogs/" + formatID(catalog.ID) + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var events []model.Event
	if err := json.NewDecoder(listResp.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("archived catalog wrote %d events, want 0", len(events))
	}
}
