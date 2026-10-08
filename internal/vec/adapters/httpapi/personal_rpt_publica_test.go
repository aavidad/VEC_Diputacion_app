package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	rptpublica "vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

type consultaRPTPublicaPrueba struct{ catalogo domain.CatalogoRPTPublica }

func (c consultaRPTPublicaPrueba) Listar(context.Context) (domain.CatalogoRPTPublica, error) {
	return c.catalogo, nil
}

type consultaRPTConteoPrueba struct {
	catalogo domain.CatalogoRPTPublica
	llamadas int
}

func (c *consultaRPTConteoPrueba) Listar(context.Context) (domain.CatalogoRPTPublica, error) {
	c.llamadas++
	return c.catalogo, nil
}

func TestHandlerRPTPublicaCodigoPuestoLiteralAntesDeTotal(t *testing.T) {
	fuente, err := rptpublica.NuevaFuente(filepath.Join("..", "..", "..", "..", "data", "catalogos", "rpt", "v1.rpt-2026.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := fuente.ObtenerRPTPublica(t.Context())
	if err != nil || len(catalogo.Puestos) != 842 {
		t.Fatalf("fuente RPT pública no disponible: %v", err)
	}
	consulta := &consultaRPTConteoPrueba{catalogo: catalogo}
	handler, err := NewHandlerRPTPublica(consulta)
	if err != nil {
		t.Fatal(err)
	}
	consultar := func(query string) (int, map[string]any) {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaRPTPublicaPersonal+query, nil))
		var respuesta map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &respuesta); err != nil {
			t.Fatal(err)
		}
		return rec.Code, respuesta
	}
	estado, respuesta := consultar("?vista=puestos&q=&limit=1&offset=0&codigo_puesto=217")
	rpt := respuesta["data"].(map[string]any)["rpt"].(map[string]any)
	items := rpt["items"].([]any)
	if estado != http.StatusOK || rpt["total"] != float64(1) || len(items) != 1 ||
		items[0].(map[string]any)["codigo"] != "217" ||
		rpt["fuente"].(map[string]any)["importacion"] != catalogo.Fuente.Importacion ||
		rpt["fuente"].(map[string]any)["huella_sha256"] != catalogo.Fuente.HuellaSHA256 ||
		consulta.llamadas != 1 {
		t.Fatal("código literal, procedencia o única lectura alterados")
	}
	for _, caso := range []struct {
		codigo string
		quiere string
	}{
		{"21", "21"}, {"217-A", ""}, {"999999", ""},
	} {
		estado, respuesta = consultar("?vista=puestos&q=&limit=1&offset=0&codigo_puesto=" + caso.codigo)
		rpt = respuesta["data"].(map[string]any)["rpt"].(map[string]any)
		filas := rpt["items"].([]any)
		if estado != http.StatusOK || rpt["total"] != float64(len(filas)) ||
			(caso.quiere == "" && len(filas) != 0) ||
			(caso.quiere != "" && (len(filas) != 1 || filas[0].(map[string]any)["codigo"] != caso.quiere)) {
			t.Fatalf("se aceptó coincidencia aproximada para %s", caso.codigo)
		}
	}
	for _, query := range []string{
		"?vista=categorias&q=&limit=1&offset=0&codigo_puesto=217",
		"?vista=puestos&q=&limit=1&offset=0&codigo_puesto=",
		"?vista=puestos&q=&limit=1&offset=0&codigo_puesto=217&codigo_puesto=218",
		"?vista=puestos&q=&limit=1&offset=0&codigo_puesto=217%20",
		"?vista=puestos&q=&limit=1&offset=0&codigo_puesto=217a",
	} {
		if estado, _ := consultar(query); estado != http.StatusBadRequest {
			t.Fatalf("código anómalo admitido: %s", query)
		}
	}
	if consulta.llamadas != 4 {
		t.Fatalf("petición inválida consultó catálogo: %d", consulta.llamadas)
	}
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
