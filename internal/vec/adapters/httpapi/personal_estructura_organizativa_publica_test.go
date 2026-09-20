package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

type consultaEstructuraPublicaPrueba struct {
	estructura domain.EstructuraOrganizativaPublica
}

func (c consultaEstructuraPublicaPrueba) Obtener(context.Context) (domain.EstructuraOrganizativaPublica, error) {
	return c.estructura, nil
}
func TestHandlerEstructuraOrganizativaPublicaLimitaRutaMetodoYCampos(t *testing.T) {
	unidades := make([]domain.UnidadEstructuraPublica, 0, 66)
	for i := 0; i < 14; i++ {
		unidades = append(unidades, domain.UnidadEstructuraPublica{Clave: "delegacion-" + string(rune('a'+i)), Etiqueta: "Delegacion", Tipo: "delegacion"})
	}
	for i := 0; i < 41; i++ {
		unidades = append(unidades, domain.UnidadEstructuraPublica{Clave: "centro-" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Etiqueta: "Centro", Tipo: "centro", AdscripcionClave: "delegacion-a"})
	}
	for i := 0; i < 11; i++ {
		unidades = append(unidades, domain.UnidadEstructuraPublica{Clave: "puesto-" + string(rune('a'+i)), Etiqueta: "Jefatura", Tipo: "puesto_responsabilidad", AdscripcionClave: "centro-a"})
	}
	estructura := domain.EstructuraOrganizativaPublica{Esquema: "vec.personal.estructura-organizativa-publica.v1", CatalogoID: "estructura-organizativa-dipgra", CatalogoVersion: 1, CatalogoRevision: 1, FuenteRef: "https://example.test/rpt", Fuente: domain.FuenteEstructuraPublica{Revision: "demo-v1", ActualizadaEn: "2026-09-06T00:00:00Z", Demostracion: true, Aviso: "Demo sin vigencia", HuellaSHA256: strings.Repeat("a", 64)}, Unidades: unidades}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileRRHHPresentation, RRHHPresentationEnabled: true, RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement, RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement}
	h, err := NewHandlerEstructuraOrganizativaPublicaPresentacion(cfg, consultaEstructuraPublicaPrueba{estructura})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		want         int
	}{{http.MethodGet, rutaEstructuraOrganizativaPublicaPresentacion, 200}, {http.MethodHead, rutaEstructuraOrganizativaPublicaPresentacion, 200}, {http.MethodPost, rutaEstructuraOrganizativaPublicaPresentacion, 405}, {http.MethodGet, rutaEstructuraOrganizativaPublicaPresentacion + "?x=1", 404}, {http.MethodGet, rutaEstructuraOrganizativaPublicaPresentacion + "/x", 404}} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if rec.Code != caso.want {
			t.Errorf("%s %s = %d: %s", caso.metodo, caso.ruta, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "codigo_fuente") || strings.Contains(rec.Body.String(), "pagina_fuente") || strings.Contains(rec.Body.String(), "modificada_localmente") {
			t.Errorf("proyeccion sensible: %s", rec.Body.String())
		}
	}
}
