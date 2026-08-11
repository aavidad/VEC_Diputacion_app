package main

import (
	"go/ast"
	"testing"
)

func TestDetectarCiclo(t *testing.T) {
	funciones := map[string]*ast.FuncDecl{"a": {}, "b": {}, "c": {}}
	if got := detectarCiclo(funciones, []arista{{Desde: "func:a", Hacia: "func:b"}, {Desde: "func:b", Hacia: "func:c"}}); got != "" {
		t.Fatalf("DAG rechazado: %s", got)
	}
	if got := detectarCiclo(funciones, []arista{{Desde: "func:a", Hacia: "func:b"}, {Desde: "func:b", Hacia: "func:a"}}); got == "" {
		t.Fatal("ciclo aceptado")
	}
	if got := detectarCiclo(funciones, []arista{{Desde: "tipo:x", Hacia: "tipo:x", Tipo: "ownership"}}); got == "" {
		t.Fatal("autociclo ownership aceptado")
	}
}

func TestCardinalidadYOrdenFallanCerrado(t *testing.T) {
	if err := exigir([]byte("uno uno"), "uno", 1, "cardinalidad"); err == nil {
		t.Fatal("cardinalidad divergente aceptada")
	}
	if err := enOrden([]byte("segundo primero"), []string{"primero", "segundo"}, "orden"); err == nil {
		t.Fatal("orden divergente aceptado")
	}
	if err := enOrden([]byte("primero segundo"), []string{"primero", "segundo"}, "orden"); err != nil {
		t.Fatalf("orden exacto rechazado: %v", err)
	}
}

func TestResumenFuentesDeterminista(t *testing.T) {
	a := resumenFuentes(map[string]string{"b": "2", "a": "1"})
	b := resumenFuentes(map[string]string{"a": "1", "b": "2"})
	if a != b || a == "" {
		t.Fatalf("resumen no determinista: %q %q", a, b)
	}
}

func TestModoMutanteOmiteSoloSHA(t *testing.T) {
	if err := validarSHA("mutada", "sellada", false); err == nil {
		t.Fatal("modo normal aceptó una huella mutada")
	}
	if err := validarSHA("mutada", "sellada", true); err != nil {
		t.Fatalf("modo mutante murió solo por SHA: %v", err)
	}
	if err := enOrden([]byte("CONT marca"), []string{"marca", "CONT"}, "causal"); err == nil {
		t.Fatal("modo mutante dejó de aplicar el oráculo causal")
	}
}
