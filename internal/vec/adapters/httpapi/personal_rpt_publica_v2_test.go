package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestRPTPublicaV2FiltraCategoriaYCentroSoloEnPuestos(t *testing.T) {
	f, err := leerFiltroRPTPublicaV2("vista=puestos&q=&limit=25&offset=0&categoria_clave=administrativo&centro_codigo=101")
	if err != nil || f.CategoriaClave != "administrativo" || f.CentroCodigo != "101" {
		t.Fatalf("filtro exacto=%+v err=%v", f, err)
	}
	if _, err := leerFiltroRPTPublicaV2("vista=categorias&q=&limit=25&offset=0&categoria_clave=administrativo"); err == nil {
		t.Fatal("se admitió filtro de puestos sobre categorías")
	}
}

func TestRPTPublicaV2AdmiteUnicodeLegalYRechazaQueryExcesiva(t *testing.T) {
	consulta := &consultaRPTV2Prueba{}
	auditor := &auditorRPTV2Prueba{}
	h, err := NewHandlerRPTPublicaV2(autoridadRPTV2Prueba{ErrAutenticacionRutaExactaRequerida}, consulta, auditor)
	if err != nil {
		t.Fatal(err)
	}
	categoria := strings.Repeat("a", personaldomain.LongitudMaximaClaveCategoriaRPTPublicaV2)
	centro := strings.Repeat("1", 64)
	for _, q := range []string{strings.Repeat("á", 100), strings.Repeat("😀", 100)} {
		raw := url.Values{"vista": {"puestos"}, "q": {q}, "limit": {"25"}, "offset": {"0"}, "categoria_clave": {categoria}, "centro_codigo": {centro}}.Encode()
		if len(raw) <= 512 || len(raw) > maximoQueryRPTPublicaV2 {
			t.Fatalf("presupuesto no cubre filtro Unicode válido: %d", len(raw))
		}
		f, err := leerFiltroRPTPublicaV2(raw)
		if err != nil || f.Q != q {
			t.Fatalf("filtro Unicode=%d err=%v", len(raw), err)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaRPTPublicaPersonalV2+"?"+raw, nil))
		if w.Code != http.StatusUnauthorized || consulta.llamada {
			t.Fatalf("Unicode legal: estado=%d consulta=%t", w.Code, consulta.llamada)
		}
	}
	sobreclave := url.Values{"vista": {"puestos"}, "q": {""}, "limit": {"25"}, "offset": {"0"}, "categoria_clave": {categoria + "a"}}.Encode()
	if _, err := leerFiltroRPTPublicaV2(sobreclave); err == nil {
		t.Fatal("se admitió clave de categoría mayor que la fuente candidata")
	}
	antes := len(auditor.llamadas)
	excesiva := strings.Repeat("x", maximoQueryRPTPublicaV2+1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaRPTPublicaPersonalV2+"?"+excesiva, nil))
	if w.Code != http.StatusBadRequest || len(auditor.llamadas) != antes || consulta.llamada {
		t.Fatalf("query excesiva: estado=%d auditorias=%d", w.Code, len(auditor.llamadas))
	}
}
