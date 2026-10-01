package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIVistaMovimientosESENConservaSaldo(t *testing.T) {
	for _, c := range []struct{ carpeta, idioma string }{{"es", "es-ES"}, {"en", "en-GB"}} {
		t.Run(c.carpeta, func(t *testing.T) {
			var salida bytes.Buffer
			args := []string{"--vista=movimientos", "../../web/static/textos/" + c.carpeta + "/cronos-informe-movimientos.json", "testdata/movimientos.json"}
			if err := ejecutar(context.Background(), args, &salida); err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(salida.Bytes(), []byte("%PDF-")) || !bytes.Contains(salida.Bytes(), []byte("/Lang ("+c.idioma+")")) {
				t.Fatal("idioma o documento discordante")
			}
			primera := append([]byte(nil), salida.Bytes()...)
			salida.Reset()
			if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.Equal(primera, salida.Bytes()) {
				t.Fatal("PDF no determinista", err)
			}
			salida.Reset()
			args = []string{"../../web/static/textos/" + c.carpeta + "/cronos-informe-saldo.json", "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"}
			if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.Contains(salida.Bytes(), []byte("/Lang ("+c.idioma+")")) {
				t.Fatal("modo saldo alterado", err)
			}
		})
	}
}
func TestCLIVistaMovimientosRechazaEntradaAjenaSinBytes(t *testing.T) {
	raw, err := os.ReadFile("testdata/movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	var base map[string]any
	if err := json.Unmarshal(raw, &base); err != nil {
		t.Fatal(err)
	}
	casos := map[string][]byte{
		"sin_demo":            bytes.Replace(raw, []byte(`"demo": true`), []byte(`"demo": false`), 1),
		"campo_ajeno":         bytes.Replace(raw, []byte(`"nombre":`), []byte(`"motivo": "dato ajeno", "nombre":`), 1),
		"campo_ajeno_anidado": bytes.Replace(raw, []byte(`"movimiento":`), []byte(`"marcaje_ref": "dato ajeno", "movimiento":`), 1),
		"duplicado":           bytes.Replace(raw, []byte(`"demo": true`), []byte(`"demo": false, "demo": true`), 1),
		"alias_duplicado":     bytes.Replace(raw, []byte(`"demo": true`), []byte(`"demo": false, "DEMO": true`), 1),
		"movimiento_ajeno":    bytes.Replace(raw, []byte(`"salida"`), []byte(`"ajeno"`), 1),
		"origen_ajeno":        bytes.Replace(raw, []byte(`"remoto"`), []byte(`"portal"`), 1),
		"utc_falso":           bytes.Replace(raw, []byte(`2026-10-25T01:30:00Z`), []byte(`2026-10-25T02:30:00+01:00`), 1),
		"precision_ajena":     bytes.Replace(raw, []byte(`2026-10-25T01:30:00Z`), []byte(`2026-10-25T01:30:00.000000001Z`), 1),
		"fuera_periodo":       bytes.Replace(raw, []byte(`2026-10-25T01:30:00Z`), []byte(`2026-10-26T01:30:00Z`), 1),
		"demasiados_bytes":    append(append([]byte(nil), raw...), bytes.Repeat([]byte(" "), 65537)...),
		"segundo_json":        append(append([]byte(nil), raw...), []byte(`{}`)...),
	}
	for _, campo := range []string{"completo", "marcajes"} {
		copia := make(map[string]any, len(base))
		for k, v := range base {
			copia[k] = v
		}
		delete(copia, campo)
		b, err := json.Marshal(copia)
		if err != nil {
			t.Fatal(err)
		}
		casos["ausente_"+campo] = b
	}
	for nombre, datos := range casos {
		t.Run(nombre, func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "ejemplo.json")
			if err := os.WriteFile(ruta, datos, 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if err := ejecutar(context.Background(), []string{"--vista=movimientos", "../../web/static/textos/es/cronos-informe-movimientos.json", ruta}, &salida); err == nil || salida.Len() != 0 {
				t.Fatal("entrada inválida produjo bytes", err)
			}
		})
	}
	for _, args := range [][]string{{"--vista=ajena", "../../web/static/textos/es/cronos-informe-movimientos.json", "testdata/movimientos.json"}, {"--vista=movimientos", "../../web/static/textos/es/cronos-informe-saldo.json", "testdata/movimientos.json"}, {"../../web/static/textos/es/cronos-informe-saldo.json", "testdata/movimientos.json"}, {"--vista=movimientos", "../../web/static/textos/es/cronos-informe-movimientos.json", "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"}} {
		var salida bytes.Buffer
		if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
			t.Fatal("esquemas de vistas mezclados", args, err)
		}
	}
}
func TestCLIVistaMovimientosCatalogoInvalidoSinBytes(t *testing.T) {
	base, err := os.ReadFile("../../web/static/textos/en/cronos-informe-movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{bytes.Replace(base, []byte(`"en-GB"`), []byte(`"en_GB"`), 1), bytes.Replace(base, []byte(`"idioma": "en-GB"`), []byte(`"idioma": "es-ES", "idioma": "en-GB"`), 1), append(append([]byte(nil), base...), bytes.Repeat([]byte(" "), 65537)...)} {
		ruta := filepath.Join(t.TempDir(), "catalogo.json")
		if err := os.WriteFile(ruta, raw, 0600); err != nil {
			t.Fatal(err)
		}
		var salida bytes.Buffer
		err := ejecutar(context.Background(), []string{"--vista=movimientos", ruta, "testdata/movimientos.json"}, &salida)
		if err == nil || salida.Len() != 0 {
			t.Fatal("catálogo inválido produjo bytes", err)
		}
		var aviso bytes.Buffer
		if err := informarError(&aviso, err); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(aviso.String(), "en_GB") || !strings.HasPrefix(aviso.String(), "cronos_exportacion_saldo_") {
			t.Fatal("el error filtra datos", aviso.String())
		}
	}
}

func TestCLIVistaMovimientosRechazaAliasUnicodeSinPerderFilas(t *testing.T) {
	raw, err := os.ReadFile("testdata/movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	objeto := bytes.TrimSpace(raw)
	for nombre, clave := range map[string]string{"ascii": "MARCAJES", "unicode": "marcajeſ", "unicode_escapado": `marcaje\u017f`} {
		t.Run(nombre, func(t *testing.T) {
			datos := append(append([]byte(nil), objeto[:len(objeto)-1]...), []byte(`, "`+clave+`": []}`)...)
			ruta := filepath.Join(t.TempDir(), "ejemplo.json")
			if err := os.WriteFile(ruta, datos, 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if err := ejecutar(context.Background(), []string{"--vista=movimientos", "../../web/static/textos/es/cronos-informe-movimientos.json", ruta}, &salida); err == nil || salida.Len() != 0 {
				t.Fatal("un alias borró filas y produjo bytes", err)
			}
		})
	}
}
