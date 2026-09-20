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
	catalogo := domain.CatalogoRPTPublica{Esquema: "vec.catalogo.rpt.v1", Fuente: domain.FuenteRPTPublica{Documento: "RPT publicada", Importacion: "rpt-publica-v1", GeneradoEn: "2026-09-17", Aviso: "Datos públicos sin ocupantes.", HuellaSHA256: strings.Repeat("a", 64)}, Resumen: domain.ResumenRPTPublica{Puestos: 1, Dotacion: 14, Categorias: 1, Centros: 1}, Categorias: []domain.CategoriaRPTPublica{{Clave: "aux-tec-sup-informatica", Denominacion: "AUX. TEC. SUP. INFORMATICA", Grupos: []string{"B"}, Escalas: []string{}, Puestos: 1, Dotacion: 14}}, Puestos: []domain.PuestoRPTPublico{{Codigo: "430-101-001", Denominacion: "AUXILIAR TECNICO", CentroCodigo: "101", Centro: "CENTRO DE PRUEBA", Delegacion: "PRESIDENCIA", Grupos: []string{}, Escala: "", CategoriaClave: "aux-tec-sup-informatica", NivelDestino: 17, ComplementoEspecificoAnualCentimos: 1400000, Dotacion: 14, Tipo: "E", Provision: "I"}}}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileRRHHPresentation, RRHHPresentationEnabled: true, RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement, RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement}
	handler, err := NewHandlerRPTPublicaPresentacion(cfg, consultaRPTPublicaPrueba{catalogo})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		want         int
	}{{http.MethodGet, rutaRPTPublicaPresentacion + "?q=aux&limit=1&offset=0", 200}, {http.MethodGet, rutaRPTPublicaPresentacion + "?vista=puestos&q=centro&limit=1&offset=0", 200}, {http.MethodGet, rutaRPTPublicaPresentacion + "?vista=jefaturas&q=&limit=1&offset=0", 400}, {http.MethodGet, "/api/vec/personal/rpt-publica/administrativo", 404}, {http.MethodPost, rutaRPTPublicaPresentacion, 405}, {http.MethodGet, rutaRPTPublicaPresentacion + "?limit=1&offset=0&nivel=1", 400}} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if rec.Code != caso.want {
			t.Errorf("%s %s = %d: %s", caso.metodo, caso.ruta, rec.Code, rec.Body.String())
		}
		if caso.want == http.StatusOK && !strings.Contains(rec.Body.String(), `"resumen":{"puestos":1,"dotacion":14,"categorias":1,"centros":1}`) {
			t.Errorf("respuesta no entrega resumen completo: %s", rec.Body.String())
		}
		if caso.want == http.StatusOK && strings.Contains(rec.Body.String(), `"vista":"puestos"`) && (!strings.Contains(rec.Body.String(), `"nivel_destino":17`) || !strings.Contains(rec.Body.String(), `"complemento_especifico_anual_centimos":1400000`)) {
			t.Errorf("respuesta de puestos pierde campos publicos: %s", rec.Body.String())
		}
		if caso.want == http.StatusOK && strings.Contains(rec.Body.String(), `"vista":"categorias"`) && (strings.Contains(rec.Body.String(), "nivel_destino") || strings.Contains(rec.Body.String(), "complemento_especifico")) {
			t.Errorf("respuesta de categorias expone campos de puesto: %s", rec.Body.String())
		}
		if caso.want == http.StatusOK && strings.Contains(rec.Body.String(), `"vista":"categorias"`) && (strings.Contains(rec.Body.String(), `"escalas":null`) || !strings.Contains(rec.Body.String(), `"escalas":[]`)) {
			t.Errorf("respuesta no normaliza escalas vacías: %s", rec.Body.String())
		}
	}
}
