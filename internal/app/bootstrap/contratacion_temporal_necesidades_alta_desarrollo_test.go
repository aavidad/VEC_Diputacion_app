package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestCatalogosAltaV1ConservadoYV2DistingueNecesidad(t *testing.T) {
	catalogos, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		t.Fatal(err)
	}
	manejador := &manejadorCatalogosAltaContratacionTemporalDesarrollo{origen: nuevoOrigenConsultasConCatalogoDesarrollo(catalogos)}
	consultar := func(query string) (int, map[string]any) {
		peticion := httptest.NewRequest(http.MethodGet, rutaCatalogosAltaContratacionTemporalDesarrollo+query, nil)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		var cuerpo map[string]any
		if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
			t.Fatal(err)
		}
		return respuesta.Code, cuerpo
	}
	if estado, cuerpo := consultar(""); estado != 200 || cuerpo["data"].(map[string]any)["esquema"] != esquemaCatalogosAltaContratacionTemporal ||
		len(cuerpo["data"].(map[string]any)["motivos"].([]any)) != 1 {
		t.Fatal("se alteró el contrato v1")
	}
	estado, cuerpo := consultar("?version=2")
	if estado != 200 {
		t.Fatalf("v2 respondió %d", estado)
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
	if estado, _ := consultar("?version=3"); estado != 400 {
		t.Fatalf("versión desconocida respondió %d", estado)
	}
	if estado, _ := consultar("?version=%32"); estado != 400 {
		t.Fatalf("versión no canónica respondió %d", estado)
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
