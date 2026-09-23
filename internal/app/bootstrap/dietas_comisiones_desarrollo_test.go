package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type autoridadExactaDelegadaPrueba struct{ llamadas int }

func (a *autoridadExactaDelegadaPrueba) AutorizarRutaExacta(context.Context, string) error {
	a.llamadas++
	return nil
}

func TestComisionesDietasSoloSeMontanConSelectorYMaterialNominal(t *testing.T) {
	if a, err := nuevasComisionesDietasDesarrollo(config.Config{DietasBorradoresEnabled: "false"}, nil, nil); err != nil || a != nil {
		t.Fatalf("Dietas inerte no debe abrir rutas: %v %v", a, err)
	}
	if a, err := nuevasComisionesDietasDesarrollo(config.Config{DietasBorradoresEnabled: "true"}, nil, nil); err == nil || a != nil {
		t.Fatalf("selector sin gobierno abrió ruta: %v %v", a, err)
	}
}

func TestFronteraComisionesDietasNoDelegaNiSirveSinSesion(t *testing.T) {
	for _, caso := range []struct {
		ruta, metodo string
		admitido     bool
	}{
		{dietashttp.RutaBorradores, http.MethodGet, true},
		{dietashttp.RutaBorradores, http.MethodPost, true},
		{dietashttp.RutaBorradores, http.MethodPut, false},
		{dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa", http.MethodGet, true},
		{dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa", http.MethodPost, false},
	} {
		if obtenido := metodoComisionesDietasValido(caso.ruta, caso.metodo); obtenido != caso.admitido {
			t.Fatalf("método %s %s admitido=%t", caso.metodo, caso.ruta, obtenido)
		}
	}
	delegada := &autoridadExactaDelegadaPrueba{}
	a := autoridadExactasConDietas{delegada: delegada, dietas: &autoridadComisionesDietasDesarrollo{}}
	for _, ruta := range []string{dietashttp.RutaBorradores, dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa"} {
		if err := a.AutorizarRutaExacta(context.Background(), ruta); err != vechttp.ErrAutenticacionRutaExactaRequerida {
			t.Fatalf("ruta %q: se esperaba autenticación, recibida %v", ruta, err)
		}
	}
	if delegada.llamadas != 0 {
		t.Fatal("Dietas pasó por la autoridad ajena")
	}
	if err := a.AutorizarRutaExacta(context.Background(), "/api/vec/contratacion/alta"); err != nil || delegada.llamadas != 1 {
		t.Fatalf("ruta ajena no delegada: %v %d", err, delegada.llamadas)
	}
	llamadas := 0
	protegida := (&autoridadComisionesDietasDesarrollo{}).proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamadas++; w.WriteHeader(http.StatusNoContent) }))
	respuesta := httptest.NewRecorder()
	protegida.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores, nil))
	if respuesta.Code != http.StatusUnauthorized || llamadas != 0 || respuesta.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("sin certificado: estado=%d llamadas=%d", respuesta.Code, llamadas)
	}
}
