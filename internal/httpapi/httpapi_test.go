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
