package main

import (
	"go/ast"
	"testing"
)

func TestDetectarCiclo(t *testing.T) {
	funciones := map[string]*ast.FuncDecl{"a": {}, "b": {}, "c": {}}
	if got := detectarCiclo(funciones, []arista{{Desde: "a", Hacia: "b"}, {Desde: "b", Hacia: "c"}}); got != "" {
		t.Fatalf("DAG rechazado: %s", got)
	}
	if got := detectarCiclo(funciones, []arista{{Desde: "a", Hacia: "b"}, {Desde: "b", Hacia: "a"}}); got == "" {
		t.Fatal("ciclo aceptado")
	}
}

func TestExigirFallaCerrado(t *testing.T) {
	if err := exigir([]byte("uno uno"), "uno", 1, "cardinalidad"); err == nil {
		t.Fatal("cardinalidad divergente aceptada")
	}
	if err := exigir([]byte("uno"), "uno", 1, "cardinalidad"); err != nil {
		t.Fatalf("cardinalidad exacta rechazada: %v", err)
	}
}

func TestResumenFuentesDeterminista(t *testing.T) {
	a := resumenFuentes(map[string]string{"b": "2", "a": "1"})
	b := resumenFuentes(map[string]string{"a": "1", "b": "2"})
	if a != b || a == "" {
		t.Fatalf("resumen no determinista: %q %q", a, b)
	}
}
