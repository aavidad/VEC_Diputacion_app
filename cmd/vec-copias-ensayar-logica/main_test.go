package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRechazaEntradaAntesDeEjecutarYConservaAlcance(t *testing.T) {
	raiz := t.TempDir()
	config, solicitud := filepath.Join(raiz, "config.json"), filepath.Join(raiz, "solicitud.json")
	// Configuración incompleta: la frontera del adaptador rechaza antes de Docker.
	if err := os.WriteFile(config, []byte(`{"tiempo_limite_segundos":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(solicitud, []byte(`{"sintetica":true,"dump":{"ruta":"sintetica.dump","sha256":"invalida"},"globals":{"ruta":"sintetica.sql","sha256":"invalida"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			var out, diag bytes.Buffer
			args := []string{"-configuracion", config, "-solicitud", solicitud, "-catalogo", "../../web/static/textos/" + idioma + "/copias-ensayo-logico.json"}
			if run(args, &out, &diag) != 1 || diag.Len() != 0 {
				t.Fatalf("resultado no esperado: %s", &diag)
			}
			var r salida
			if json.Unmarshal(out.Bytes(), &r) != nil || r.Mensaje == "" || r.Alcance == "" || r.Resultado.HabilitaRestauracion || !r.Resultado.LimpiezaCompletada || r.Resultado.Etapa != "entrada" {
				t.Fatal("alcance o catálogo incorrecto")
			}
			if strings.Contains(out.String(), raiz) || strings.Contains(out.String(), "sintetica.dump") {
				t.Fatal("la respuesta filtra rutas locales")
			}
		})
	}
}

func TestLecturaJSONRechazaCamposDesconocidosYContenidoAdicional(t *testing.T) {
	for _, texto := range []string{`{"desconocido":true}`, `{} {}`, strings.Repeat(" ", (1<<20)+1)} {
		ruta := filepath.Join(t.TempDir(), "entrada.json")
		if err := os.WriteFile(ruta, []byte(texto), 0600); err != nil {
			t.Fatal(err)
		}
		var c configuracionCLI
		if leer(ruta, &c) == nil {
			t.Fatal("entrada no admitida aceptada")
		}
	}
}
