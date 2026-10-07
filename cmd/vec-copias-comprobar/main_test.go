package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func datosEjemplo(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCLITresEstadosSinAfirmarAutenticidadNiRestauracion(t *testing.T) {
	for _, tt := range []struct {
		nombre string
		estado copias.Estado
		codigo int
	}{{"compatible", copias.Compatible, 0}, {"incompatible", copias.Incompatible, 1}, {"no-comprobable", copias.NoComprobable, 2}} {
		t.Run(tt.nombre, func(t *testing.T) {
			var out, diag bytes.Buffer
			if code := run(nil, bytes.NewReader(datosEjemplo(t, tt.nombre)), &out, &diag); code != tt.codigo || diag.Len() != 0 {
				t.Fatalf("%d %s", code, diag.String())
			}
			var s salida
			if json.Unmarshal(out.Bytes(), &s) != nil {
				t.Fatal("salida inválida")
			}
			if s.Resultado.Estado != tt.estado || s.HabilitaRestauracion || s.Autenticidad != "no_comprobada" || s.VerificacionRestauracion != "no_comprobada" {
				t.Fatalf("%+v", s)
			}
		})
	}
}

func TestCLICatalogosRealesYClavesCoincidentes(t *testing.T) {
	var anterior map[string]string
	for _, idioma := range []string{"es", "en"} {
		p := "../../web/static/textos/" + idioma + "/copias-seguridad.json"
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var textos map[string]string
		if leerEstricto(bytes.NewReader(b), &textos) != nil {
			t.Fatal("catálogo inválido")
		}
		if anterior != nil {
			if len(anterior) != len(textos) {
				t.Fatal("claves distintas")
			}
			for key := range anterior {
				if textos[key] == "" {
					t.Fatalf("falta %s", key)
				}
			}
		}
		anterior = textos
		for _, nombre := range []string{"compatible", "incompatible", "no-comprobable"} {
			var out, diag bytes.Buffer
			if run([]string{"-catalogo", p}, bytes.NewReader(datosEjemplo(t, nombre)), &out, &diag) == 3 || diag.Len() != 0 {
				t.Fatal(diag.String())
			}
			var s salida
			_ = json.Unmarshal(out.Bytes(), &s)
			if s.Mensaje == "" || len(s.Mensajes) != len(s.Resultado.Razones) {
				t.Fatal("no usa el catálogo real")
			}
		}
	}
}

func TestEntradaEstrictoSinFiltrarContenido(t *testing.T) {
	b := string(datosEjemplo(t, "compatible"))
	for _, tt := range []struct{ nombre, dato string }{
		{"desconocido", strings.Replace(b, `"modo": "conjunto_completo"`, `"modo": "conjunto_completo", "secreto": "no_imprimir"`, 1)},
		{"repetido", strings.Replace(b, `"modo": "conjunto_completo"`, `"modo": "conjunto_completo", "modo": "solo_base"`, 1)},
		{"repetido_escapado", strings.Replace(b, `"modo": "conjunto_completo"`, `"modo": "conjunto_completo", "\u006dodo": "solo_base"`, 1)},
		{"repetido_anidado", strings.Replace(b, `"tamano_bytes": 9`, `"tamano_bytes": 9, "tamano_bytes": 10`, 1)},
		{"mayusculas", strings.Replace(b, `"modo":`, `"Modo":`, 1)},
		{"mayusculas_anidado", strings.Replace(b, `"postgresql":`, `"PostgreSQL":`, 1)},
		{"nulo", strings.Replace(b, `"releases_revocadas": []`, `"releases_revocadas": null`, 1)},
		{"array_omitido", strings.Replace(b, ",\n    \"releases_revocadas\": []", "", 1)},
		{"otro_documento", b + `{}`},
		{"malformado", `{"secreto":"no_imprimir"`},
		{"profundo", strings.Repeat("[", maxProfundidad+2) + "1" + strings.Repeat("]", maxProfundidad+2)},
		{"demasiado_grande", strings.Repeat(" ", maxEntradaBytes+1)},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			if tt.dato == b {
				t.Fatal("la mutación no se aplicó")
			}
			var out, diag bytes.Buffer
			if run(nil, strings.NewReader(tt.dato), &out, &diag) != 3 || out.Len() != 0 || strings.Contains(diag.String(), "no_imprimir") {
				t.Fatalf("%s / %s", out.String(), diag.String())
			}
			if diag.String() != "{\"error_clave\":\"copias_seguridad_error_entrada\"}\n" {
				t.Fatal("diagnóstico inesperado")
			}
		})
	}
}

func TestFormatoDesconocidoYModoSinForzarBloquean(t *testing.T) {
	for _, tt := range []struct {
		antes, despues string
		code           int
	}{{`"formato_version": 1`, `"formato_version": 2`, 2}, {`"modo": "conjunto_completo"`, `"modo": "forzar"`, 1}} {
		b := strings.Replace(string(datosEjemplo(t, "compatible")), tt.antes, tt.despues, 1)
		var out, diag bytes.Buffer
		if got := run(nil, strings.NewReader(b), &out, &diag); got != tt.code {
			t.Fatalf("%d / %s", got, diag.String())
		}
	}
}

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) { return 0, errors.New("no_imprimir") }

func TestErroresLocalesNoRevelanRutaNiContenido(t *testing.T) {
	for _, args := range [][]string{{"-catalogo", "ruta_privada_no_imprimir"}, {"-forzar"}} {
		var out, diag bytes.Buffer
		if run(args, bytes.NewReader(datosEjemplo(t, "compatible")), &out, &diag) != 3 || strings.Contains(diag.String(), "no_imprimir") {
			t.Fatal("filtra ruta")
		}
	}
	var diag bytes.Buffer
	if run(nil, bytes.NewReader(datosEjemplo(t, "compatible")), escritorFallido{}, &diag) != 3 || strings.Contains(diag.String(), "no_imprimir") {
		t.Fatal("filtra error de salida")
	}
}

func TestFalloAlEscribirDiagnosticoDevuelveCuatro(t *testing.T) {
	for _, tt := range []struct {
		nombre  string
		args    []string
		entrada io.Reader
		salida  io.Writer
	}{
		{"argumentos", []string{"-forzar"}, strings.NewReader(""), io.Discard},
		{"catalogo", []string{"-catalogo", "ruta_privada_no_imprimir"}, strings.NewReader(""), io.Discard},
		{"entrada", nil, strings.NewReader(`{"secreto":"no_imprimir"}`), io.Discard},
		{"salida", nil, bytes.NewReader(datosEjemplo(t, "compatible")), escritorFallido{}},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			if code := run(tt.args, tt.entrada, tt.salida, escritorFallido{}); code != 4 {
				t.Fatalf("código %d; esperado 4", code)
			}
		})
	}
	if diagnosticar(escritorFallido{}, "copias_seguridad_error_entrada") == nil {
		t.Fatal("descarta el error de escritura")
	}
}

func TestParserRechazaLectorFallido(t *testing.T) {
	if leerEstricto(io.LimitReader(lectorFallido{}, 10), new(entrada)) == nil {
		t.Fatal("acepta error")
	}
}

type lectorFallido struct{}

func (lectorFallido) Read([]byte) (int, error) { return 0, errors.New("no_imprimir") }
