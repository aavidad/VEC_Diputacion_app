package domain

import (
	"errors"
	"testing"
)

func TestNuevoBorradorLlamamientoFijaEstadoVersionYContenidoMinimo(t *testing.T) {
	b, err := NuevoBorradorLlamamiento("borrador-llamamiento:alta:abc", "per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", ContenidoBorradorLlamamiento{Resumen: "Cobertura temporal de necesidad interna"})
	if err != nil || b.Estado() != EstadoBorradorLlamamientoInterno || b.Version() != 1 || b.Validar() != nil {
		t.Fatalf("borrador invalido: %#v, %v", b, err)
	}
	_, err = NuevoBorradorLlamamiento("borrador-llamamiento:alta:abc", "per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", ContenidoBorradorLlamamiento{Resumen: "Contactar con dni 12345678Z"})
	if !errors.Is(err, ErrBorradorLlamamientoInvalido) {
		t.Fatalf("acepto contenido personal: %v", err)
	}
	if _, err := NuevoBorradorLlamamiento("borrador-llamamiento:alta:def", "per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", ContenidoBorradorLlamamiento{Resumen: "Necesidad de ingeniero"}); err != nil {
		t.Fatalf("rechazo una palabra ordinaria: %v", err)
	}
	if _, err := NuevoBorradorLlamamiento("borrador-llamamiento:alta:ghi", "per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", ContenidoBorradorLlamamiento{Resumen: "NIE: X"}); !errors.Is(err, ErrBorradorLlamamientoInvalido) {
		t.Fatalf("acepto etiqueta personal: %v", err)
	}
}

func TestHuellaComandoBorradorExcluyeReferenciaGeneradaYEsEstable(t *testing.T) {
	c := ContenidoBorradorLlamamiento{Resumen: "Cobertura temporal de necesidad interna"}
	a, err := HuellaComandoCrearBorradorLlamamiento("per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", "clave-prueba-001", c)
	b, err2 := HuellaComandoCrearBorradorLlamamiento("per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", "clave-prueba-001", c)
	if err != nil || err2 != nil || a != b || a != "cb1d853380e856638e589804b5be90745aefba5741140d94790f4fa11e913ced" {
		t.Fatalf("huella no canonica: %q %q %v %v", a, b, err, err2)
	}
}

func TestHuellaComandoBorradorNoAplicaEscapeHTML(t *testing.T) {
	huella, err := HuellaComandoCrearBorradorLlamamiento(
		"per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", "clave-prueba-002",
		ContenidoBorradorLlamamiento{Resumen: "Cobertura <A&B> ágil"},
	)
	if err != nil || huella != "8014b8d00f5637189e8e11e8031d2dfaeaf23a6f81c5846885308d284e1cdbb5" {
		t.Fatalf("huella incompatible con JSON canónico PostgreSQL: %q, %v", huella, err)
	}
	if _, err := HuellaComandoCrearBorradorLlamamiento(
		"per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", "clave-prueba-003",
		ContenidoBorradorLlamamiento{Resumen: "Cobertura\u2028interna"},
	); !errors.Is(err, ErrBorradorLlamamientoInvalido) {
		t.Fatalf("acepto separador Unicode no canónico: %v", err)
	}
	if _, err := HuellaComandoCrearBorradorLlamamiento(
		"per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", "clave-prueba-004",
		ContenidoBorradorLlamamiento{Resumen: "Cobertura\x00interna"},
	); !errors.Is(err, ErrBorradorLlamamientoInvalido) {
		t.Fatalf("acepto NUL incompatible con jsonb: %v", err)
	}
}
