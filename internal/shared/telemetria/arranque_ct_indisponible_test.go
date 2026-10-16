package telemetria

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBorradorNoDisponibleTieneCausaCatalogadaSinRuta(t *testing.T) {
	privado := "postgres://usuario:clave@localhost/base"
	err := fmt.Errorf("%w (bolsa_borrador_llamamiento_desarrollo.go:880): %s",
		errors.New("bootstrap: borrador de llamamiento no disponible"), privado)
	if got := MensajeErrorArranque(err); got != "borrador_llamamiento_no_disponible" {
		t.Fatalf("diagnóstico seguro: %q", got)
	}
	if got := ClaseErrorArranque(err); got != "borrador_llamamiento_no_disponible" {
		t.Fatalf("clase segura: %q", got)
	}
	if strings.Contains(MensajeErrorArranque(err), privado) {
		t.Fatal("el diagnóstico contiene el DSN")
	}
	compuesto := fmt.Errorf("%w: %w", err, errors.New(privado))
	if got := ClaseErrorArranque(compuesto); got != "borrador_llamamiento_no_disponible" {
		t.Fatalf("clase de errores unidos: %q", got)
	}
	if strings.Contains(MensajeErrorArranque(compuesto), privado) {
		t.Fatal("el diagnóstico unido contiene el DSN")
	}
}
