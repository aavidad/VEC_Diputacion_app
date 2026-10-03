package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	d "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func escribir(t *testing.T, nombre string, valor any) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), nombre)
	b, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta
}
func snapshotCLI() d.Snapshot {
	s := d.Snapshot{Version: 1, PostgreSQL: "18.4", Completo: true}
	for _, c := range []string{"esquema", "roles", "acl", "extensiones", "privilegios_defecto"} {
		h := sha256.Sum256([]byte(c))
		s.Objetos = append(s.Objetos, d.Objeto{Clase: c, Clave: "inventario", Cantidad: 0, SHA256: hex.EncodeToString(h[:])})
	}
	for _, c := range []string{"tablas", "secuencias", "objetos_grandes"} {
		s.Objetos = append(s.Objetos, d.Resumir(c, s.Objetos))
	}
	return s
}

func ejecutar(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	var out, diag bytes.Buffer
	code := run(context.Background(), args, &out, &diag)
	return code, out.String(), diag.String()
}
func catalogoCLI(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../web/static/textos/es/copias_contraste.json")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCLIComparaEstados(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*d.Snapshot)
		code   int
		estado string
	}{
		{"igual", func(*d.Snapshot) {}, 0, "igual"},
		{"cambio", func(s *d.Snapshot) { s.Objetos[0].Cantidad++ }, 1, "diferente"},
		{"incompleto", func(s *d.Snapshot) { s.Completo = false }, 1, "no_comprobable"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a, b := snapshotCLI(), snapshotCLI()
			caso.mutar(&b)
			code, out, diag := ejecutar(t, []string{"--modo", "comparar", "--catalogo", catalogoCLI(t), "--esperado", escribir(t, "a.json", a), "--observado", escribir(t, "b.json", b)})
			if code != caso.code || !strings.Contains(out, `"estado":"`+caso.estado+`"`) || diag != "" {
				t.Fatalf("code %d out %s diag %s", code, out, diag)
			}
		})
	}
}

func TestCLISinErroresPrivados(t *testing.T) {
	privada := filepath.Join(t.TempDir(), "dsn-privada-no-exponer")
	code, out, diag := ejecutar(t, []string{"--modo", "comparar", "--catalogo", catalogoCLI(t), "--esperado", privada, "--observado", privada})
	if code != 2 || out != "" || strings.Contains(diag, privada) || !strings.Contains(diag, "copias_contraste_error_entrada") {
		t.Fatalf("%d %s %s", code, out, diag)
	}
}

func TestLecturaEstricta(t *testing.T) {
	for _, texto := range []string{`{"version":1,"version":1}`, `{"completo":false,"Completo":true}`, `{"Completo":true}`, `{"objetos":[{"clase":"tablas","Clase":"roles"}]}`, `{"objetos":[{"ſha256":"oculto"}]}`, "{\"objetos\":[{\"clave\":\"\xff\"}]}", `{"extra":1}`, `{} {}`, `[]`, strings.Repeat("[", 34) + strings.Repeat("]", 34)} {
		p := filepath.Join(t.TempDir(), "entrada")
		if err := os.WriteFile(p, []byte(texto), 0600); err != nil {
			t.Fatal(err)
		}
		var s d.Snapshot
		if leer(p, &s, 1024) == nil {
			t.Errorf("entrada ambigua aceptada: %q", texto)
		}
	}
}

func TestCatalogosAyudaYCodigos(t *testing.T) {
	var es, en map[string]string
	if leer(catalogoCLI(t), &es, 1<<20) != nil || leer("../../web/static/textos/en/copias_contraste.json", &en, 1<<20) != nil {
		t.Fatal("catálogos")
	}
	if len(es) != len(en) {
		t.Fatal("claves")
	}
	for k := range es {
		if en[k] == "" {
			t.Fatal(k)
		}
	}
	code, out, diag := ejecutar(t, []string{"--ayuda", "--catalogo", catalogoCLI(t)})
	if code != 0 || !strings.Contains(out, "--modo capturar") || diag != "" {
		t.Fatalf("%d %s %s", code, out, diag)
	}
}

type diagnosticoFallido struct{ intentos int }

func (w *diagnosticoFallido) Write([]byte) (int, error) {
	w.intentos++
	return 0, io.ErrClosedPipe
}

func TestFalloDiagnosticoImpideExito(t *testing.T) {
	for _, codigo := range []int{0, 1, 2} {
		w := &diagnosticoFallido{}
		if obtenido := emitirDiagnostico(w, diagnostico{Clave: "captura_completada"}, codigo); obtenido != 2 || w.intentos != 1 {
			t.Fatalf("diagnóstico fallido: código %d, intentos %d", obtenido, w.intentos)
		}
		var salida bytes.Buffer
		if obtenido := emitirDiagnostico(&salida, diagnostico{Clave: "captura_completada"}, codigo); obtenido != codigo || salida.Len() == 0 {
			t.Fatal("diagnóstico válido no conserva su código")
		}
	}
	w := &diagnosticoFallido{}
	if codigo := run(context.Background(), nil, io.Discard, w); codigo != 2 || w.intentos != 1 {
		t.Fatalf("CLI con diagnóstico fallido: código %d, intentos %d", codigo, w.intentos)
	}
}

func TestCLIMultibaseSinExclusionNoConecta(t *testing.T) {
	c := configuracion{DSN: "host=127.0.0.1 port=1 dbname=postgres password=secreto_sintetico", VersionPostgreSQL: "18.4", TiempoMaximoSegundos: 1, MaxFilas: 100000, MaxBytes: 8 << 20, MaxObjetos: 1000, BasesInventariadas: []string{"postgres", "template0", "template1"}}
	cfg := escribir(t, "config.json", c)
	code, out, diag := ejecutar(t, []string{"--modo", "capturar", "--configuracion", cfg, "--catalogo", catalogoCLI(t)})
	if code != 2 || out != "" || !strings.Contains(diag, "copias_contraste_error_captura") || strings.Contains(diag, "secreto_sintetico") {
		t.Fatalf("multibase sin exclusión: código %d, salida %q, diagnóstico %q", code, out, diag)
	}
	w := &diagnosticoFallido{}
	if code := run(context.Background(), []string{"--modo", "capturar", "--configuracion", cfg, "--catalogo", catalogoCLI(t)}, io.Discard, w); code != 2 || w.intentos != 1 {
		t.Fatal("error multibase pierde fallo de diagnóstico")
	}
}
