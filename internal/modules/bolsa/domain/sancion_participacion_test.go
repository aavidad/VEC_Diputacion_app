package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func datosSancionPrueba() DatosSancion {
	return DatosSancion{
		Consecuencia: "b24.sancion.suspension", Causa: "No se presentó a la incorporación",
		FechaNotificacion: "2026-09-20",
		Resolucion:        DocumentoSancion{Referencia: "registro:2026/000123", SHA256: strings.Repeat("a", 64)},
		ResueltaPor:       "jefatura-rrhh",
	}
}

func TestDatosSancionValidos(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	if err := datosSancionPrueba().Validar(ahora); err != nil {
		t.Fatalf("datos válidos rechazados: %v", err)
	}
}

func TestDatosSancionRechazaCamposInvalidos(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	mutaciones := map[string]func(*DatosSancion){
		"consecuencia vacía":     func(d *DatosSancion) { d.Consecuencia = "" },
		"consecuencia mayúscula": func(d *DatosSancion) { d.Consecuencia = "B24" },
		"causa vacía":            func(d *DatosSancion) { d.Causa = "" },
		"causa con espacios":     func(d *DatosSancion) { d.Causa = " x" },
		"causa larga":            func(d *DatosSancion) { d.Causa = strings.Repeat("x", 1001) },
		"fecha futura":           func(d *DatosSancion) { d.FechaNotificacion = "2026-10-20" },
		"fecha mal formada":      func(d *DatosSancion) { d.FechaNotificacion = "2026-9-20" },
		"huella corta":           func(d *DatosSancion) { d.Resolucion.SHA256 = "abc" },
		"referencia con DNI":     func(d *DatosSancion) { d.Resolucion.Referencia = "dni-12345678Z" },
		"sin quien resuelve":     func(d *DatosSancion) { d.ResueltaPor = "" },
	}
	for nombre, mutar := range mutaciones {
		datos := datosSancionPrueba()
		mutar(&datos)
		if err := datos.Validar(ahora); !errors.Is(err, ErrSancionParticipacionInvalida) {
			t.Errorf("%s: se esperaba rechazo, obtenido %v", nombre, err)
		}
	}
}

func TestEfectosSancion(t *testing.T) {
	for _, efecto := range []string{EfectoSancionNinguno, OperacionPausar, OperacionExcluir} {
		if !EfectoSancionValido(efecto) {
			t.Errorf("%s debería ser válido", efecto)
		}
	}
	for _, efecto := range []string{OperacionReactivar, "", "suspender"} {
		if EfectoSancionValido(efecto) {
			t.Errorf("%s no debería ser válido", efecto)
		}
	}
}

func TestEventoRecurso(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	evento := EventoRecursoSancion{Estado: "interpuesto", Fecha: "2026-09-24"}
	if err := evento.ValidarDatos(ahora); err != nil {
		t.Fatalf("evento válido rechazado: %v", err)
	}
	evento.Documento = &DocumentoSancion{Referencia: "registro:2026/9", SHA256: "x"}
	if evento.ValidarDatos(ahora) == nil {
		t.Fatal("documento con huella inválida aceptado")
	}
	evento = EventoRecursoSancion{Estado: "Interpuesto", Fecha: "2026-09-24"}
	if evento.ValidarDatos(ahora) == nil {
		t.Fatal("estado mal formado aceptado")
	}
	s := SancionParticipacion{Recursos: []EventoRecursoSancion{{Estado: "interpuesto"}, {Estado: "desestimado"}}}
	if s.EstadoRecurso() != "desestimado" || (SancionParticipacion{}).EstadoRecurso() != "" {
		t.Fatal("estado vigente del recurso incorrecto")
	}
}
