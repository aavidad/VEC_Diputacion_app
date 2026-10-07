package httpmodulos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/modulos"
)

type control struct {
	err       error
	consultas int
	id        string
}

func (c *control) ExigirHabilitado(_ context.Context, id string) error {
	c.consultas++
	c.id = id
	return c.err
}

func TestGuardNoInvocaOperacionDesactivadaNiDaPermiso(t *testing.T) {
	c := &control{}
	efectos := 0
	h, err := Proteger("vec.module.cronos", c, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { efectos++; w.WriteHeader(http.StatusForbidden) }))
	if err != nil {
		t.Fatal(err)
	}
	peticion := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/cronos/fichaje", nil))
		return w
	}
	if w := peticion(); w.Code != http.StatusForbidden || efectos != 1 {
		t.Fatal("guard ha concedido permiso")
	}
	c.err = domain.ErrDesactivado
	if w := peticion(); w.Code != http.StatusNotFound || efectos != 1 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, efectos)
	}
	c.err = errors.New("fuente inaccesible")
	if w := peticion(); w.Code != http.StatusServiceUnavailable || efectos != 1 {
		t.Fatal(w.Code, efectos)
	}
	if c.consultas != 3 || c.id != "vec.module.cronos" {
		t.Fatal(c)
	}
}
