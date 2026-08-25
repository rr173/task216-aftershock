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

// TestCreateEventPreservesOriginTime 验证通过 HTTP 导入的 RFC3339 发震时间
// 原样写入持久化记录（含秒级与小数秒精度），不会变为零值或被截断。
func TestCreateEventPreservesOriginTime(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "event.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(httpapi.New(service.New(db)).Handler())
	defer server.Close()

	body := bytes.NewBufferString(`{"name":"catalog-e","region":"tohoku"}`)
	res, err := http.Post(server.URL+"/api/catalogs", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create catalog status = %d", res.StatusCode)
	}
	var catalog model.Catalog
	if err := json.NewDecoder(res.Body).Decode(&catalog); err != nil {
		res.Body.Close()
		t.Fatal(err)
	}
	res.Body.Close()

	// 含秒与亚秒精度的 RFC3339 时间，确保不被按分钟截断。
	const rfc3339 = "2024-03-11T06:25:43.123456789Z"
	want, err := time.Parse(time.RFC3339Nano, rfc3339)
	if err != nil {
		t.Fatal(err)
	}

	reqBody := bytes.NewBufferString(fmt.Sprintf(
		`{"origin_time":%q,"latitude":38.297,"longitude":142.373,"depth_km":24.0,"magnitude":7.1,"loc_error_h_km":3.0,"loc_error_z_km":4.0}`,
		rfc3339))
	postRes, err := http.Post(
		server.URL+"/api/catalogs/"+formatID(catalog.ID)+"/events",
		"application/json", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	defer postRes.Body.Close()
	if postRes.StatusCode != http.StatusCreated {
		t.Fatalf("create event status = %d", postRes.StatusCode)
	}
	var created model.Event
	if err := json.NewDecoder(postRes.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if !created.OriginTime.Equal(want) {
		t.Fatalf("response origin_time = %v, want %v", created.OriginTime, want)
	}

	// 关闭并重新打开同一数据库，验证从持久化层读回的时间仍然完整。
	db.Close()
	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := service.New(reopened).Event.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.OriginTime.Equal(want) {
		t.Fatalf("persisted origin_time = %v, want %v (raw %q)",
			got.OriginTime, want, got.OriginTime.Format(time.RFC3339Nano))
	}
	// 额外断言：亚秒精度未被按分钟或按秒截断。
	if got.OriginTime.Nanosecond() != want.Nanosecond() {
		t.Fatalf("sub-second precision lost: got ns=%d, want ns=%d",
			got.OriginTime.Nanosecond(), want.Nanosecond())
	}
}
