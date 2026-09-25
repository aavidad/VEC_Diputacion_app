package main

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func analizarFuente(t *testing.T, fuente string) []string {
	t.Helper()
	fset := token.NewFileSet()
	fichero, err := parser.ParseFile(fset, "x.go", "package x\n"+fuente, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var r []string
	for _, h := range AnalizarFichero(fset, fichero, "x.go") {
		r = append(r, h.Funcion+":"+h.Regla)
	}
	sort.Strings(r)
	return r
}

func comprobar(t *testing.T, fuente string, esperado ...string) {
	t.Helper()
	got := analizarFuente(t, fuente)
	if strings.Join(got, ",") != strings.Join(esperado, ",") {
		t.Fatalf("hallazgos = %v, esperado %v\n%s", got, esperado, fuente)
	}
}

func TestVS001Positivos(t *testing.T) {
	comprobar(t, `func a() error { if err := f(); err != nil { return nil }; return nil }`, "a:VS001")
	comprobar(t, `func b() (int, bool) { v, err := g(); if err != nil { return 0, false }; return v, true }`, "b:VS001")
	comprobar(t, `func c() { for { if err := f(); err != nil { continue } } }`, "c:VS001")
	comprobar(t, `func d() { if err := f(); err != nil { } }`, "d:VS001")
	comprobar(t, `func e() *T { if errCarga := f(); errCarga != nil { return &T{} }; return nil }`, "e:VS001")
	comprobar(t, `func (s *S) m() (T, bool) { if err := f(); nil != err { return vacio, false }; return T{}, true }`, "S.m:VS001")
	comprobar(t, `func h() { go func() { if err := f(); err != nil { return } }() }`, "h:VS001")
}

func TestVS001Negativos(t *testing.T) {
	comprobar(t, `func a() error { if err := f(); err != nil { return err }; return nil }`)
	comprobar(t, `func b() error { if err := f(); err != nil { return ErrCarga }; return nil }`)
	comprobar(t, `func c() error { if err := f(); err != nil { return fmt.Errorf("x: %w", err) }; return nil }`)
	comprobar(t, `func d() bool { if err := f(); err != nil { log.Print("fijo"); return false }; return true }`)
	comprobar(t, `func e() { if err := f(); err != nil { emisor.Emitir(s); return } }`)
	comprobar(t, `func g(w http.ResponseWriter) { if err := f(); err != nil { http.Error(w, "x", 400); return } }`)
	comprobar(t, `func h() (err error) { if err2 := f(); err2 != nil { return } ; return nil }`)
	comprobar(t, `func i() error { if err := f(); err != nil { return pkg.ErrNoDisponible }; return nil }`)
	comprobar(t, `func j() { if err := f(); err != nil { panic("inalcanzable") } }`)
	comprobar(t, `func k() bool { if x != nil { return false }; return true }`)
	comprobar(t, `func l() bool {
	//vec:silencio-justificado PREDICADO_VALIDACION el error significa no valido
	if err := f(); err != nil { return false }
	return true }`)
	comprobar(t, `func m() bool { if err := f(); err != nil { return false } //vec:silencio-justificado HASH_INFALIBLE texto
	return true }`)
}

func TestVS000MotivoFueraDeLaLista(t *testing.T) {
	comprobar(t, `func a() bool {
	//vec:silencio-justificado PORQUE_SI texto
	if err := f(); err != nil { return false }
	return true }`, "(directiva):VS000", "a:VS001")
	comprobar(t, `func b() {} //vec:silencio-justificado HASH_INFALIBLE`, "(directiva):VS000")
}

func TestVS002(t *testing.T) {
	comprobar(t, `func a() { defer func() { _ = recover() }() }`, "a:VS002")
	comprobar(t, `func b() (err error) { defer func() { if r := recover(); r != nil { err = errX } }(); return nil }`, "b:VS002")
	comprobar(t, `func c() { defer func() { if r := recover(); r != nil { emitirIncidenciaPanico(); } }() }`)
	comprobar(t, `func d() { defer func() { if r := recover(); r != nil { panic(r) } }() }`)
	comprobar(t, `func e() { defer func() { if r := recover(); r != nil { slog.Error("panico") } }() }`)
}

func TestVS003(t *testing.T) {
	comprobar(t, `func a(w http.ResponseWriter) { w.WriteHeader(http.StatusServiceUnavailable) }`, "a:VS003")
	comprobar(t, `func b(w http.ResponseWriter) { writeError(w, 502, "x") }`, "b:VS003")
	comprobar(t, `func c(w http.ResponseWriter, r *http.Request) { ports.EmitirIncidenciaTecnicaDesdeContexto(r.Context(), s); w.WriteHeader(http.StatusInternalServerError) }`)
	comprobar(t, `func d(w http.ResponseWriter) { w.WriteHeader(http.StatusNotFound) }`)
}

func TestLineaBaseSoloDecrece(t *testing.T) {
	base := map[string]int{"a.go:f:VS001": 2, "a.go:g:VS002": 1}
	vigentes := map[string]int{"a.go:f:VS001": 1, "a.go:h:VS001": 1, "a.go:i:VS003": 5, "a.go:(directiva):VS000": 1}
	c := Comparar(base, vigentes)
	if strings.Join(c.Nuevas, ",") != "a.go:(directiva):VS000,a.go:h:VS001" {
		t.Fatalf("nuevas = %v", c.Nuevas)
	}
	if len(c.Reducibles) != 2 {
		t.Fatalf("reducibles = %v", c.Reducibles)
	}
	reducida := Reducir(base, vigentes)
	if len(reducida) != 1 || reducida["a.go:f:VS001"] != 1 {
		t.Fatalf("reducida = %v", reducida)
	}
	if c := Comparar(base, map[string]int{"a.go:f:VS001": 3}); len(c.Nuevas) != 1 {
		t.Fatalf("un recuento creciente debe fallar: %v", c.Nuevas)
	}
}

func TestEjecutarFallaConHuellaNuevaYPasaConBase(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "internal", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	fuente := "package p\nfunc f() bool { if err := g(); err != nil { return false }; return true }\nfunc g() error { return nil }\n"
	if err := os.WriteFile(filepath.Join(raiz, "internal", "p", "p.go"), []byte(fuente), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "internal", "p", "p_test.go"), []byte("package p\nfunc h() { if err := g(); err != nil { return } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(raiz, "base.txt")
	var salida, errores bytes.Buffer
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "internal/p/p.go:2: VS001 en f") {
		t.Fatalf("código %d, errores %q", codigo, errores.String())
	}
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-generar-base"}, &salida, &errores); codigo != 0 {
		t.Fatalf("generar: %d", codigo)
	}
	errores.Reset()
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base}, &salida, &errores); codigo != 0 {
		t.Fatalf("con base: %d %q", codigo, errores.String())
	}
	contenido, err := os.ReadFile(base)
	if err != nil || !strings.Contains(string(contenido), "internal/p/p.go:f:VS001 1\n") {
		t.Fatalf("base = %q", contenido)
	}
	// Corregido el fallo, -actualizar-base retira la huella y nunca añade.
	if err := os.WriteFile(filepath.Join(raiz, "internal", "p", "p.go"), []byte("package p\nfunc f() error { return g() }\nfunc g() error { return nil }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-actualizar-base"}, &salida, &errores); codigo != 0 {
		t.Fatalf("actualizar: %d", codigo)
	}
	contenido, _ = os.ReadFile(base)
	if strings.Contains(string(contenido), "VS001") {
		t.Fatalf("la base no decreció: %q", contenido)
	}
}

func TestLeerBaseRechazaLineasMalFormadas(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "base.txt")
	for _, contenido := range []string{"a.go:f:VS001\n", "a.go:f:VS001 0\n", "af 1\n"} {
		if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LeerBase(ruta); err == nil {
			t.Fatalf("aceptó %q", contenido)
		}
	}
}
