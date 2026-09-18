package ports

import (
	"testing"
	"time"
)

func fechaPrueba(valor string) time.Time {
	fecha, err := FechaEstadisticasRRHH(valor)
	if err != nil {
		panic(err)
	}
	return fecha
}

func TestConsultaEstadisticasRRHHValidar(t *testing.T) {
	casos := []struct {
		nombre   string
		consulta ConsultaEstadisticasRRHH
		valida   bool
	}{
		{"mensual un año", ConsultaEstadisticasRRHH{Periodo: "mensual", Desde: fechaPrueba("2025-10-01"), Hasta: fechaPrueba("2026-09-18")}, true},
		{"semanal", ConsultaEstadisticasRRHH{Periodo: "semanal", Desde: fechaPrueba("2026-08-31"), Hasta: fechaPrueba("2026-09-20")}, true},
		{"anual cinco años", ConsultaEstadisticasRRHH{Periodo: "anual", Desde: fechaPrueba("2022-01-01"), Hasta: fechaPrueba("2026-12-31")}, true},
		{"periodo desconocido", ConsultaEstadisticasRRHH{Periodo: "diario", Desde: fechaPrueba("2026-09-01"), Hasta: fechaPrueba("2026-09-18")}, false},
		{"desde posterior a hasta", ConsultaEstadisticasRRHH{Periodo: "mensual", Desde: fechaPrueba("2026-09-19"), Hasta: fechaPrueba("2026-09-18")}, false},
		{"sin fechas", ConsultaEstadisticasRRHH{Periodo: "mensual"}, false},
		{"anterior a 2000", ConsultaEstadisticasRRHH{Periodo: "anual", Desde: fechaPrueba("1999-12-31"), Hasta: fechaPrueba("2026-01-01")}, false},
		{"semanal demasiado amplio", ConsultaEstadisticasRRHH{Periodo: "semanal", Desde: fechaPrueba("2016-01-01"), Hasta: fechaPrueba("2026-01-01")}, false},
	}
	for _, caso := range casos {
		if err := caso.consulta.Validar(); (err == nil) != caso.valida {
			t.Fatalf("%s: valida=%v err=%v", caso.nombre, caso.valida, err)
		}
	}
}

func TestAlcanceEstadisticasRRHHValidar(t *testing.T) {
	if err := (AlcanceEstadisticasRRHH{OrganizacionRef: "organizacion:desarrollo:dipgra", ClaseAmbito: AmbitoOrganizacionRRHH, AmbitoRef: "organizacion:desarrollo:dipgra"}).Validar(); err != nil {
		t.Fatalf("alcance de organización válido: %v", err)
	}
	if err := (AlcanceEstadisticasRRHH{OrganizacionRef: "organizacion:desarrollo:dipgra", ClaseAmbito: AmbitoOrganizacionRRHH, AmbitoRef: "centro:x"}).Validar(); err == nil {
		t.Fatal("el ámbito de organización debe coincidir con la organización")
	}
	if err := (AlcanceEstadisticasRRHH{OrganizacionRef: "organizacion:desarrollo:dipgra", ClaseAmbito: "otro", AmbitoRef: "organizacion:desarrollo:dipgra"}).Validar(); err == nil {
		t.Fatal("clase de ámbito desconocida")
	}
	if err := (AlcanceEstadisticasRRHH{OrganizacionRef: "", ClaseAmbito: AmbitoCentroRRHH, AmbitoRef: "centro:x"}).Validar(); err == nil {
		t.Fatal("organización vacía")
	}
}

func TestFechaEstadisticasRRHH(t *testing.T) {
	if _, err := FechaEstadisticasRRHH("2026-9-1"); err == nil {
		t.Fatal("formato no canónico aceptado")
	}
	if _, err := FechaEstadisticasRRHH("2026-02-30"); err == nil {
		t.Fatal("fecha inexistente aceptada")
	}
	if fecha, err := FechaEstadisticasRRHH("2026-09-18"); err != nil || fecha.Location() != time.UTC || fecha.Hour() != 0 {
		t.Fatalf("fecha canónica: %v %v", fecha, err)
	}
}

func TestEstadisticasRRHHTotales(t *testing.T) {
	e := EstadisticasRRHH{Series: []SerieEstadisticasRRHH{
		{Altas: 52, Formalizaciones: 2, Incidencias: 1},
		{Formalizaciones: 1},
		{Altas: 19, Llamamientos: 3, Cierres: 2, Incidencias: 1},
	}}
	total := e.Totales()
	if total.Altas != 71 || total.Llamamientos != 3 || total.Formalizaciones != 3 || total.Cierres != 2 || total.Incidencias != 2 {
		t.Fatalf("totales inesperados: %+v", total)
	}
}
