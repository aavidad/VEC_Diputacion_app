package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestDatosOfertaValidar(t *testing.T) {
	valida := DatosOferta{Categoria: "Auxiliar", Centro: "Residencia", FechaInicio: "2026-10-01", Descripcion: "Sustitución"}
	if err := valida.Validar(); err != nil {
		t.Fatalf("oferta sin fecha de fin rechazada: %v", err)
	}
	casos := map[string]func(*DatosOferta){
		"categoria vacía":   func(d *DatosOferta) { d.Categoria = "" },
		"centro con blanco": func(d *DatosOferta) { d.Centro = " Residencia" },
		"fecha inválida":    func(d *DatosOferta) { d.FechaInicio = "2026-13-01" },
		"fin anterior":      func(d *DatosOferta) { d.FechaFin = "2026-09-30" },
		"fin mal formada":   func(d *DatosOferta) { d.FechaFin = "31/12/2026" },
		"descripción larga": func(d *DatosOferta) { d.Descripcion = strings.Repeat("x", 2001) },
	}
	for nombre, mutar := range casos {
		d := valida
		mutar(&d)
		if err := d.Validar(); !errors.Is(err, ErrDatosOfertaInvalidos) {
			t.Errorf("%s: err=%v", nombre, err)
		}
	}
	d := valida
	d.FechaFin = "2026-10-01"
	if err := d.Validar(); err != nil {
		t.Fatalf("fin igual al inicio rechazado: %v", err)
	}
}
