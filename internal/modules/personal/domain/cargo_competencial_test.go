package domain

import (
	"testing"
	"time"
)

func TestEnlaceCargoCompetencialSeparaClasesYVigencia(t *testing.T) {
	desde := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	e := EnlaceCargoCompetencial{
		Referencia: "enc_1234567890123456789012", Version: 1,
		HuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CargoRef:     "car_1234567890123456789012", PersonaRef: "per_1234567890123456789012",
		Clase: EjercicioDelegacionFirma, Desde: desde, Hasta: desde.Add(24 * time.Hour), ActoRef: "acto:delegacion:1",
	}
	if err := e.Validar(); err != nil {
		t.Fatal(err)
	}
	e.RequiereEnlaceLaboral = true
	if err := e.Validar(); err == nil {
		t.Fatal("acto con enlace laboral requerido aceptado sin ocupacion")
	}
	e.EmpleadoRef = "emp_1234567890123456789012"
	e.OcupacionRef = "ocu_1234567890123456789012"
	e.OcupacionRevision = 1
	if err := e.Validar(); err != nil {
		t.Fatal(err)
	}
	e.Clase = "perfil"
	if err := e.Validar(); err == nil {
		t.Fatal("perfil de acceso aceptado como cargo")
	}
	e.Clase = EjercicioTitular
	e.Hasta = e.Desde
	if err := e.Validar(); err == nil {
		t.Fatal("intervalo vacio aceptado")
	}
}
