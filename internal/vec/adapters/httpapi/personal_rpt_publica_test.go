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

type consultaRPTPublicaPrueba struct{ catalogo domain.CatalogoRPTPublica }

func (c consultaRPTPublicaPrueba) Listar(context.Context) (domain.CatalogoRPTPublica, error) {
	return c.catalogo, nil
}
func TestHandlerRPTPublicaPresentacionLimitaRutaMetodoYProyeccion(t *testing.T) {
	catalogo := domain.CatalogoRPTPublica{Esquema: "vec.catalogo.rpt.v1", Fuente: domain.FuenteRPTPublica{Documento: "RPT publicada", Importacion: "rpt-publica-v1", GeneradoEn: "2026-09-17", Aviso: "Datos públicos sin ocupantes.", HuellaSHA256: strings.Repeat("a", 64)}, Categorias: []domain.CategoriaRPTPublica{{Clave: "aux-tec-sup-informatica", Denominacion: "AUX. TEC. SUP. INFORMATICA", Grupos: []string{"B"}, Puestos: 5, Dotacion: 14}}}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileRRHHPresentation, RRHHPresentationEnabled: true, RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement, RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement}
	handler, err := NewHandlerRPTPublicaPresentacion(cfg, consultaRPTPublicaPrueba{catalogo})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		want         int
	}{{http.MethodGet, rutaRPTPublicaPresentacion + "?q=aux&limit=1&offset=0", 200}, {http.MethodGet, "/api/vec/personal/rpt-publica/administrativo", 404}, {http.MethodPost, rutaRPTPublicaPresentacion, 405}, {http.MethodGet, rutaRPTPublicaPresentacion + "?limit=1&offset=0&nivel=1", 400}} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if rec.Code != caso.want {
			t.Errorf("%s %s = %d: %s", caso.metodo, caso.ruta, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "nivel_destino") || strings.Contains(rec.Body.String(), "complemento_especifico") {
			t.Errorf("respuesta expone campo prohibido: %s", rec.Body.String())
		}
		if caso.want == http.StatusOK && (strings.Contains(rec.Body.String(), `"escalas":null`) || !strings.Contains(rec.Body.String(), `"escalas":[]`)) {
			t.Errorf("respuesta no normaliza escalas vacías: %s", rec.Body.String())
		}
	}
}
