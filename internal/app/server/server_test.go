package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func TestServerHealthzIsJSON(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("/healthz json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("/healthz body = %#v, want status ok", body)
	}
}

func TestServerServesStaticUI(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	for _, tc := range []struct {
		path        string
		contentType string
		want        string
	}{
		{path: "/", contentType: "text/html", want: "VEC Diputacion"},
		{path: "/app.js", contentType: "text/javascript", want: `fetch("/api/demo"`},
		{path: "/styles.css", contentType: "text/css", want: ".listings"},
		{path: "/locales/es.json", contentType: "application/json", want: "api.candidate.created"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", tc.path, rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
			t.Fatalf("%s content-type = %q, want %q", tc.path, got, tc.contentType)
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("%s body missing %q", tc.path, tc.want)
		}
	}
}

func TestNewHTTPServerUsesConfigAndRoutesAPI(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	srv, err := NewHTTPServer(config.Config{Address: "127.0.0.1:0", ReadHeaderTimeout: time.Second}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	if srv.Addr != "127.0.0.1:0" || srv.ReadHeaderTimeout != time.Second {
		t.Fatalf("server config = %s/%s", srv.Addr, srv.ReadHeaderTimeout)
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/candidates", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("api status = %d, want 202", rec.Code)
	}
}

func TestNewHTTPServerRoutesAPIPrefixWithoutPanic(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/demo" {
			t.Fatalf("api path = %q, want /api/demo", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv, err := NewHTTPServer(config.Config{Address: "127.0.0.1:0", ReadHeaderTimeout: time.Second}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/demo", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("/api/demo status = %d, want 204", rec.Code)
	}
}

func TestNewHTTPServerRoutesProfessionalPortalEndpoint(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/portal" {
			t.Fatalf("api path = %q, want /api/portal", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	srv, err := NewHTTPServer(config.Config{Address: "127.0.0.1:0", ReadHeaderTimeout: time.Second}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/portal", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/portal status = %d, want 200", rec.Code)
	}
}
