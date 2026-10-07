package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type autoridadRPTV2Prueba struct{ err error }

func (a autoridadRPTV2Prueba) ResolverContextoRPTPublicaV2(context.Context) (vecdomain.ContextoActor, error) {
	return vecdomain.ContextoActor{}, a.err
}

type consultaRPTV2Prueba struct{ llamada bool }

func (c *consultaRPTV2Prueba) Consultar(context.Context, vecdomain.ContextoActor, personaldomain.FiltroRPTPublicaV2) (personalports.PaginaRPTPublicaV2, error) {
	c.llamada = true
	return personalports.PaginaRPTPublicaV2{}, nil
}

type auditorRPTV2Prueba struct{ llamadas []DenegacionRPTPublicaV2 }

func (a *auditorRPTV2Prueba) RegistrarDenegacionRPTPublicaV2(_ context.Context, d DenegacionRPTPublicaV2) error {
	a.llamadas = append(a.llamadas, d)
	return nil
}

func TestRPTPublicaV2ExigeContextoYAuditaDenegacion(t *testing.T) {
	consulta := &consultaRPTV2Prueba{}
	auditor := &auditorRPTV2Prueba{}
	h, err := NewHandlerRPTPublicaV2(autoridadRPTV2Prueba{ErrAutenticacionRutaExactaRequerida}, consulta, auditor)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaRPTPublicaPersonalV2+"?vista=puestos&q=&limit=1&offset=0", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || consulta.llamada || len(auditor.llamadas) != 1 ||
		auditor.llamadas[0].Ruta != RutaRPTPublicaPersonalV2 || auditor.llamadas[0].ActorRef != "" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("estado=%d consulta=%t auditoria=%+v", w.Code, consulta.llamada, auditor.llamadas)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaRPTPublicaPersonalV2+"?vista=puestos&vista=categorias&limit=1&offset=0", nil))
	if w.Code != http.StatusBadRequest || consulta.llamada {
		t.Fatalf("filtro duplicado=%d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/vec/personal/rpt-publica/v2/otro?limit=1&offset=0", nil))
	if w.Code != http.StatusNotFound || consulta.llamada {
		t.Fatalf("ruta no canónica=%d", w.Code)
	}
}

func TestRPTPublicaV2RechazaDependenciasAusentes(t *testing.T) {
	if _, err := NewHandlerRPTPublicaV2(nil, &consultaRPTV2Prueba{}, &auditorRPTV2Prueba{}); !errors.Is(err, ErrHandlerRPTPublicaV2Invalido) {
		t.Fatalf("err=%v", err)
	}
}
