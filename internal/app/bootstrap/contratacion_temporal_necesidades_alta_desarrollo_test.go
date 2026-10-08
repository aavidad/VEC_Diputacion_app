package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestAltaV1ExigeModalidadMientrasV3ExigeNecesidadSellada(t *testing.T) {
	soporte := &soporteAltaContratacionTemporalDesarrollo{}
	if admitido, _ := soporte.motivoAltaAdmitido(context.Background(), "organizacion:dipgra", "programa_temporal", "", nil); admitido {
		t.Fatal("v1 interpretó necesidad de programa como modalidad jurídica")
	}
	if admitido, err := soporte.motivoAltaAdmitido(context.Background(), "organizacion:dipgra", "sustitucion", "", nil); err != nil || !admitido {
		t.Fatal("v1 dejó de admitir su modalidad histórica", err)
	}
	if admitido, _ := soporte.motivoAltaAdmitido(context.Background(), "organizacion:dipgra", "programa_temporal", "vec.ct.alta_necesidad.v1", nil); admitido {
		t.Fatal("v3 admitió causa sin material sellado")
	}
}

func TestCatalogosAltaSinFuenteConservaV1YOcultaCapacidadV2(t *testing.T) {
	catalogos, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		t.Fatal(err)
	}
	manejador := &manejadorCatalogosAltaContratacionTemporalDesarrollo{origen: nuevoOrigenConsultasConCatalogoDesarrollo(catalogos)}
	consultar := func(metodo, query string) (int, map[string]any, http.Header, int) {
		peticion := httptest.NewRequest(metodo, rutaCatalogosAltaContratacionTemporalDesarrollo+query, nil)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		var cuerpo map[string]any
		if metodo == http.MethodGet {
			if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
				t.Fatal(err)
			}
		}
		return respuesta.Code, cuerpo, respuesta.Header(), respuesta.Body.Len()
	}
	if estado, cuerpo, _, _ := consultar(http.MethodGet, ""); estado != 200 || cuerpo["data"].(map[string]any)["esquema"] != esquemaCatalogosAltaContratacionTemporal ||
		len(cuerpo["data"].(map[string]any)["motivos"].([]any)) != 1 {
		t.Fatal("se alteró el contrato v1")
	}
	for _, metodo := range []string{http.MethodGet, http.MethodHead} {
		estado, cuerpo, cabeceras, longitud := consultar(metodo, "?version=2")
		if estado != http.StatusServiceUnavailable || cabeceras.Get("Cache-Control") != "no-store, no-transform" ||
			cabeceras.Get("Content-Length") == "" || cabeceras.Get("Set-Cookie") != "" {
			t.Fatalf("v2 sin fuente respondió %d con cabeceras %v", estado, cabeceras)
		}
		if metodo == http.MethodGet {
			errorRespuesta := cuerpo["error"].(map[string]any)
			if errorRespuesta["codigo"] != "capacidad_no_configurada" ||
				errorRespuesta["clave_i18n"] != "api.contratacion_temporal.catalogos_alta.error.capacidad_no_configurada" ||
				cuerpo["data"] != nil {
				t.Fatalf("v2 anunció una capacidad ausente: %+v", cuerpo)
			}
		} else if longitud != 0 {
			t.Fatal("HEAD devolvió cuerpo")
		}
	}
	if estado, _, _, _ := consultar(http.MethodGet, "?version=3"); estado != 400 {
		t.Fatalf("versión desconocida respondió %d", estado)
	}
	if estado, _, _, _ := consultar(http.MethodGet, "?version=%32"); estado != 400 {
		t.Fatalf("versión no canónica respondió %d", estado)
	}
}

func TestCatalogosAltaConFuenteExponeV2DistinguiendoNecesidad(t *testing.T) {
	ruta := filepath.Join("..", "..", "modules", "contrataciontemporal", "adapters", "catalogoalta", "necesidades_v1.ejemplo.json")
	catalogos, err := nuevoCatalogoDesarrollo("", "", ruta)
	if err != nil {
		t.Fatal(err)
	}
	manejador := &manejadorCatalogosAltaContratacionTemporalDesarrollo{origen: nuevoOrigenConsultasConCatalogoDesarrollo(catalogos)}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaCatalogosAltaContratacionTemporalDesarrollo+"?version=2", nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("v2 con fuente respondió %d: %s", respuesta.Code, respuesta.Body.String())
	}
	var cuerpo map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	datos := cuerpo["data"].(map[string]any)
	necesidades := datos["necesidades"].(map[string]any)
	if datos["esquema"] != esquemaCatalogosAltaContratacionTemporalV2 ||
		datos["motivos"] != nil || len(necesidades["causas"].([]any)) != 4 ||
		necesidades["jornada_referencia_minutos"] != float64(2250) {
		t.Fatalf("v2 mezcla modalidad y necesidad: %+v", datos)
	}
	if _, ok := necesidades["causas"].([]any)[0].(map[string]any)["etiqueta_clave"]; !ok {
		t.Fatal("v2 sin clave i18n")
	}
}

func TestCatalogosAltaV2ConFuenteQueFallaNoFingeAusenciaDeConfiguracion(t *testing.T) {
	origen := filepath.Join("..", "..", "modules", "contrataciontemporal", "adapters", "catalogoalta", "necesidades_v1.ejemplo.json")
	contenido, err := os.ReadFile(origen)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "necesidades.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	catalogos, err := nuevoCatalogoDesarrollo("", "", ruta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	manejador := &manejadorCatalogosAltaContratacionTemporalDesarrollo{origen: nuevoOrigenConsultasConCatalogoDesarrollo(catalogos)}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaCatalogosAltaContratacionTemporalDesarrollo+"?version=2", nil))
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("fuente caída respondió %d", respuesta.Code)
	}
	var cuerpo map[string]map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if cuerpo["error"]["codigo"] != "servicio_no_disponible" || cuerpo["data"] != nil {
		t.Fatalf("fuente caída confundida con ausencia de capacidad: %+v", cuerpo)
	}
}

func TestProyeccionPreparadaDeCuatroCausasDeNecesidadVersionadas(t *testing.T) {
	catalogo, err := catalogoNecesidadesAltaDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogo.Causas) != 4 || catalogo.JornadaReferenciaMinutos != 2250 || !catalogo.EsEjemplo {
		t.Fatalf("catálogo incompleto: causas=%d jornada=%d", len(catalogo.Causas), catalogo.JornadaReferenciaMinutos)
	}
	claves := []string{"vacante", "sustitucion", "acumulacion_tareas", "programa_temporal"}
	for i, causa := range catalogo.Causas {
		if causa.Clave != claves[i] || causa.EtiquetaClave == "" || causa.ReglaRef == "" || len(catalogo.HuellaSHA256) != 64 {
			t.Fatalf("causa %d sin identidad de regla: %+v", i, causa)
		}
	}
	if catalogo.Causas[1].FechaFin != "opcional" || catalogo.Causas[1].CausaFin != "reincorporacion_titular" {
		t.Fatal("se perdió la regla de fin abierto de sustitución")
	}
	puestoObligatorio := false
	for _, campo := range catalogo.Causas[1].CamposObligatorios {
		if campo == "puesto_codigo" {
			puestoObligatorio = true
		}
	}
	if !puestoObligatorio {
		t.Fatal("sustitución sin puesto RPT obligatorio")
	}
}

func TestJornadaSinReglaUsaDatoYReglaDeclaradaIncompletaFalla(t *testing.T) {
	if minutos, err := (fuenteJornadaCompletaDesarrollo{}).minutos(t.Context()); err != nil || minutos != 2250 {
		t.Fatalf("referencia distribuida: %d %v", minutos, err)
	}
	ruta := catalogoReglasCTModificadoPrueba(t, func(entradas []map[string]any) []map[string]any {
		filtradas := make([]map[string]any, 0, len(entradas)-1)
		for _, entrada := range entradas {
			if entrada["clave"] != reglas.CTJornadaCompleta {
				filtradas = append(filtradas, entrada)
			}
		}
		return filtradas
	})
	f := fuenteJornadaCompletaDesarrollo{resolutor: resolutorReglasCTPrueba(t, ruta)}
	if _, err := f.minutos(t.Context()); !errors.Is(err, errJornadaCompletaNoDisponible) {
		t.Fatalf("c07 ausente en reglas declaradas: %v", err)
	}
	ausente := fuenteJornadaCompletaDesarrollo{rutaCatalogo: filepath.Join(t.TempDir(), "sin-catalogo.json")}
	if _, err := ausente.minutos(t.Context()); !errors.Is(err, errJornadaCompletaNoDisponible) {
		t.Fatalf("catálogo ausente se sustituyó: %v", err)
	}
	rutaInvalida := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(rutaInvalida, []byte(`{"esquema":"incorrecto"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	invalida := fuenteJornadaCompletaDesarrollo{rutaCatalogo: rutaInvalida}
	if _, err := invalida.minutos(t.Context()); !errors.Is(err, errJornadaCompletaNoDisponible) {
		t.Fatalf("catálogo inválido se sustituyó: %v", err)
	}
}

func TestCatalogoAltaDeclaradoAusenteImpideComposicion(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "necesidades-ausentes.json")
	if _, err := nuevoCatalogoDesarrollo("", "", ruta); err == nil {
		t.Fatal("la composición sustituyó silenciosamente la ruta declarada")
	}
}
