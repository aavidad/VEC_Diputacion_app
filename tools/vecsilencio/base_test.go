package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// P1-1: la base solo puede decrecer también frente a la rama base y
// -generar-base no sirve para rehacerla.
func TestLineaBaseNoCreceFrenteALaRamaBase(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "internal", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	fuente := "package p\nfunc f() bool { if err := g(); err != nil { return false }; return true }\nfunc g() error { return nil }\n"
	if err := os.WriteFile(filepath.Join(raiz, "internal", "p", "p.go"), []byte(fuente), 0o644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(raiz, "base.txt")
	anterior := filepath.Join(raiz, "anterior.txt")
	var salida, errores bytes.Buffer
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-generar-base"}, &salida, &errores); codigo != 0 {
		t.Fatalf("alta inicial: %d %q", codigo, errores.String())
	}
	errores.Reset()
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-generar-base"}, &salida, &errores); codigo != 2 || !strings.Contains(errores.String(), "ya existe") {
		t.Fatalf("-generar-base sobre una base existente: %d %q", codigo, errores.String())
	}
	// Rama base sin la huella: una base editada para admitirla falla.
	if err := os.WriteFile(anterior, []byte("# rama base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errores.Reset()
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-base-anterior", anterior}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "internal/p/p.go:f:VS001 (1 > 0)") {
		t.Fatalf("huella nueva en la base: %d %q", codigo, errores.String())
	}
	// Rama base con la huella y un recuento mayor: la base decrece y pasa.
	if err := os.WriteFile(anterior, []byte("internal/p/p.go:f:VS001 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errores.Reset()
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-base-anterior", anterior}, &salida, &errores); codigo != 0 {
		t.Fatalf("base que decrece: %d %q", codigo, errores.String())
	}
	// Un recuento que sube respecto a la rama base falla.
	if err := os.WriteFile(base, []byte("internal/p/p.go:f:VS001 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errores.Reset()
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-base-anterior", anterior}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "(3 > 2)") {
		t.Fatalf("recuento creciente: %d %q", codigo, errores.String())
	}
	// Una base de referencia ilegible no se ignora.
	if err := os.WriteFile(anterior, []byte("basura\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-base-anterior", anterior}, &salida, &errores); codigo != 2 {
		t.Fatalf("base anterior mal formada: %d", codigo)
	}
}

// P2-8: las justificaciones se cuentan por motivo en el resumen y una nueva
// falla como cualquier huella nueva.
func TestJustificacionesSoloDecrecen(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "internal", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	escribir := func(n int) {
		t.Helper()
		var b strings.Builder
		b.WriteString("package p\n")
		for i := 0; i < n; i++ {
			b.WriteString("//vec:silencio-justificado HASH_INFALIBLE escritura en hash\n")
			b.WriteString("func f" + string(rune('a'+i)) + "() bool { if err := g(); err != nil { return false }; return true }\n")
		}
		b.WriteString("func g() error { return nil }\n")
		if err := os.WriteFile(filepath.Join(raiz, "internal", "p", "p.go"), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir(1)
	base := filepath.Join(raiz, "base.txt")
	var salida, errores bytes.Buffer
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base, "-generar-base"}, &salida, &errores); codigo != 0 {
		t.Fatalf("alta: %d", codigo)
	}
	if !strings.Contains(salida.String(), "vecsilencio: justificadas HASH_INFALIBLE 1") {
		t.Fatalf("resumen sin recuento por motivo: %q", salida.String())
	}
	contenido, err := os.ReadFile(base)
	if err != nil || !strings.Contains(string(contenido), "(justificadas):HASH_INFALIBLE:VSJ 1\n") {
		t.Fatalf("base = %q", contenido)
	}
	escribir(2)
	errores.Reset()
	if codigo := ejecutar([]string{"-raiz", raiz, "-base", base}, &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "VSJ en HASH_INFALIBLE") {
		t.Fatalf("una justificación nueva debe fallar: %d %q", codigo, errores.String())
	}
}

func TestPaqueteDeApoyoAPruebasFueraDelArbol(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "internal", "servidorprueba")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fuente := "package servidorprueba\nfunc f() []byte { if err := g(); err != nil { return nil }; return nil }\nfunc g() error { return nil }\n"
	if err := os.WriteFile(filepath.Join(dir, "s.go"), []byte(fuente), 0o644); err != nil {
		t.Fatal(err)
	}
	hallazgos, err := AnalizarArbol(raiz, []string{"internal"})
	if err != nil || len(hallazgos) != 0 {
		t.Fatalf("hallazgos = %v, err = %v", hallazgos, err)
	}
}
