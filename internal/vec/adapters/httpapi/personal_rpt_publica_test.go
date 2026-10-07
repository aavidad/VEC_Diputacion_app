package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

type consultaRPTPublicaPrueba struct{ catalogo domain.CatalogoRPTPublica }

func (c consultaRPTPublicaPrueba) Listar(context.Context) (domain.CatalogoRPTPublica, error) {
	return c.catalogo, nil
}
func TestHandlerRPTPublicaLimitaRutaMetodoYProyeccion(t *testing.T) {
	catalogo := domain.CatalogoRPTPublica{Esquema: "vec.catalogo.rpt.v1", Fuente: domain.FuenteRPTPublica{Documento: "RPT publicada", Importacion: "rpt-publica-v1", GeneradoEn: "2026-09-17", Aviso: "Datos públicos sin ocupantes.", HuellaSHA256: strings.Repeat("a", 64)}, Resumen: domain.ResumenRPTPublica{Puestos: 1, Dotacion: 14, Categorias: 1, Centros: 1}, Categorias: []domain.CategoriaRPTPublica{{Clave: "aux-tec-sup-informatica", Denominacion: "AUX. TEC. SUP. INFORMATICA", Grupos: []string{"B"}, Escalas: []string{}, Puestos: 1, Dotacion: 14}}, Puestos: []domain.PuestoRPTPublico{{Codigo: "430-101-001", Denominacion: "AUXILIAR TECNICO", CentroCodigo: "101", Centro: "CENTRO DE PRUEBA", Delegacion: "PRESIDENCIA", Grupos: []string{}, Escala: "", CategoriaClave: "aux-tec-sup-informatica", NivelDestino: 17, ComplementoEspecificoAnualCentimos: 1400000, Dotacion: 14, Tipo: "E", Provision: "I"}}}
	handler, err := NewHandlerRPTPublica(consultaRPTPublicaPrueba{catalogo})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		want         int
	}{{http.MethodGet, RutaRPTPublicaPersonal + "?q=aux&limit=1&offset=0", 200}, {http.MethodGet, RutaRPTPublicaPersonal + "?vista=puestos&q=centro&limit=1&offset=0", 200}, {http.MethodGet, RutaRPTPublicaPersonal + "?vista=jefaturas&q=&limit=1&offset=0", 400}, {http.MethodGet, "/api/vec/personal/rpt-publica/administrativo", 404}, {http.MethodPost, RutaRPTPublicaPersonal, 405}, {http.MethodGet, RutaRPTPublicaPersonal + "?limit=1&offset=0&nivel=1", 400}} {
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

func TestRPTPublicaEnlacesOptativosConFiltrosExactos(t *testing.T) {
	puesto := func(codigo, centro, categoria string, dotacion int) domain.PuestoRPTPublico {
		return domain.PuestoRPTPublico{Codigo: codigo, Denominacion: "ADMINISTRATIVO", CentroCodigo: centro, Centro: "CENTRO " + centro,
			Delegacion: "PERSONAL", Grupos: []string{"C1"}, CategoriaClave: categoria, NivelDestino: 17,
			ComplementoEspecificoAnualCentimos: 1000000, Dotacion: dotacion, Tipo: "F", Provision: "C"}
	}
	catalogo := domain.CatalogoRPTPublica{Esquema: "vec.catalogo.rpt.v1", Fuente: domain.FuenteRPTPublica{Documento: "RPT publicada", Importacion: "rpt-publica-v1", GeneradoEn: "2026-09-17", Aviso: "Sin ocupantes", HuellaSHA256: strings.Repeat("a", 64)},
		Resumen:    domain.ResumenRPTPublica{Puestos: 3, Dotacion: 9, Categorias: 1, Centros: 2},
		Categorias: []domain.CategoriaRPTPublica{{Clave: "administrativo", Denominacion: "ADMINISTRATIVO", Grupos: []string{"C1"}, Escalas: []string{}, Puestos: 1, Dotacion: 2}},
		Puestos:    []domain.PuestoRPTPublico{puesto("A-1", "101", "administrativo", 2), puesto("A-2", "101", "administrativo", 3), puesto("B-1", "102", "", 4)}}
	h, err := NewHandlerRPTPublica(consultaRPTPublicaPrueba{catalogo})
	if err != nil {
		t.Fatal(err)
	}
	pedir := func(query string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaRPTPublicaPersonal+"?"+query, nil))
		return w
	}
	legacy := pedir("vista=categorias&q=&limit=25&offset=0")
	if legacy.Code != 200 || strings.Contains(legacy.Body.String(), "puestos_vinculados") || strings.Contains(legacy.Body.String(), `"enlaces"`) {
		t.Fatalf("contrato legacy cambió: %s", legacy.Body.String())
	}
	categorias := pedir("vista=categorias&q=&limit=25&offset=0&enlaces=1")
	if categorias.Code != 200 || !strings.Contains(categorias.Body.String(), `"puestos_vinculados":2`) || !strings.Contains(categorias.Body.String(), `"dotacion_vinculada":5`) || !strings.Contains(categorias.Body.String(), `"recuento_coincide":false`) {
		t.Fatalf("recuento enlazado: %s", categorias.Body.String())
	}
	centros := pedir("vista=centros&q=&limit=1&offset=0&enlaces=1")
	if centros.Code != 200 || !strings.Contains(centros.Body.String(), `"total":2`) || !strings.Contains(centros.Body.String(), `"puestos":2`) {
		t.Fatalf("centros paginados: %s", centros.Body.String())
	}
	filtrados := pedir("vista=puestos&q=&limit=25&offset=0&enlaces=1&categoria_clave=administrativo&centro_codigo=101")
	if filtrados.Code != 200 || !strings.Contains(filtrados.Body.String(), `"total":2`) || strings.Contains(filtrados.Body.String(), `"B-1"`) {
		t.Fatalf("filtros exactos: %s", filtrados.Body.String())
	}
	for _, query := range []string{
		"vista=centros&q=&limit=25&offset=0", "vista=categorias&q=&limit=25&offset=0&categoria_clave=administrativo",
		"vista=puestos&q=&limit=25&offset=0&centro_codigo=101", "vista=puestos&q=&limit=25&offset=0&enlaces=2",
		"vista=puestos&q=&limit=25&offset=0&enlaces=1&categoria_clave=Administrativo",
	} {
		if w := pedir(query); w.Code != 400 {
			t.Errorf("filtro aceptado %s: %d", query, w.Code)
		}
	}
}
