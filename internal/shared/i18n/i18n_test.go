package i18n

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestLoadFSResolvesLocaleAndDefaultFallback(t *testing.T) {
	t.Parallel()

	catalog, err := LoadFS(fstest.MapFS{
		"locales/es.json": {Data: []byte(`{
			"api.error.unauthorized": "Autenticacion requerida",
			"api.error.forbidden": "Permisos insuficientes"
		}`)},
		"locales/en.json": {Data: []byte(`{
			"api.error.unauthorized": "Authentication required"
		}`)},
	}, "locales", WithDefaultLocale("es"))
	if err != nil {
		t.Fatalf("LoadFS() error = %v", err)
	}

	if got := catalog.T("es", "api.error.unauthorized"); got != "Autenticacion requerida" {
		t.Fatalf("T(es) = %q, want %q", got, "Autenticacion requerida")
	}
	if got := catalog.T("en", "api.error.unauthorized"); got != "Authentication required" {
		t.Fatalf("T(en) = %q, want %q", got, "Authentication required")
	}
	if got := catalog.T("en", "api.error.forbidden"); got != "Permisos insuficientes" {
		t.Fatalf("T(en fallback) = %q, want %q", got, "Permisos insuficientes")
	}
	if got := catalog.DefaultLocale(); got != "es" {
		t.Fatalf("DefaultLocale() = %q, want %q", got, "es")
	}
}

func TestCatalogMissingKeyFailsDeterministically(t *testing.T) {
	t.Parallel()

	catalog, err := New("es", map[string]map[string]string{
		"es": {
			"known": "valor",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	message, ok := catalog.Lookup("es", "missing")
	if ok {
		t.Fatalf("Lookup() ok = true, message = %q", message)
	}
	if got := catalog.T("es", "  missing  "); got != "missing" {
		t.Fatalf("T(missing) = %q, want %q", got, "missing")
	}
	if got := (*Catalog)(nil).T("es", "missing"); got != "missing" {
		t.Fatalf("nil Catalog T() = %q, want %q", got, "missing")
	}
}

func TestLoadFSErrorsAreDeterministic(t *testing.T) {
	t.Parallel()

	_, err := LoadFS(fstest.MapFS{}, "locales")
	if !errors.Is(err, ErrNoLocalesFound) {
		t.Fatalf("LoadFS(empty) error = %v, want %v", err, ErrNoLocalesFound)
	}

	_, err = LoadFS(fstest.MapFS{
		"locales/es.json": {Data: []byte(`{`)},
	}, "locales")
	if err == nil {
		t.Fatal("LoadFS(invalid json) error = nil, want error")
	}
}
