package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type consultaEnsayoFallida struct {
	llamadas int
	err      error
}

func (c *consultaEnsayoFallida) Centros(context.Context, int, time.Time) ([]calports.CentroConCalendario, error) {
	return nil, c.err
}
func (c *consultaEnsayoFallida) CalcularPlazo(context.Context, calports.SolicitudCalculoPlazo) (calports.ResultadoCalculoPlazo, error) {
	return calports.ResultadoCalculoPlazo{}, c.err
}
func (c *consultaEnsayoFallida) CalendarioCentro(context.Context, calports.SolicitudCalendarioCentro) (calports.CalendarioCentro, error) {
	c.llamadas++
	return calports.CalendarioCentro{}, c.err
}

func solicitudEnsayoMinima(t *testing.T) ports.SolicitudEnsayoCalendario {
	t.Helper()
	desde, _ := cal.ParsearFechaCivil("2026-06-01")
	hasta, _ := cal.ParsearFechaCivil("2026-06-03")
	conocido := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	fuente := ports.FuenteEnsayoCalendario{Referencia: "fuente:demo", SHA256: strings.Repeat("a", 64)}
	return ports.SolicitudEnsayoCalendario{Desde: desde, Hasta: hasta, ConocidoEn: conocido, Snapshot: ports.SnapshotEnsayoCalendario{Sintetico: true, Ref: "snapshot:demo", PersonaRef: "persona:demo", ConocidoEn: conocido, Fuente: fuente, Adscripciones: []ports.AdscripcionEnsayoCalendario{}}}
}

func TestEnsayoNoConsultaHuecoNiSolape(t *testing.T) {
	c := &consultaEnsayoFallida{err: ErrEnsayoCalendarioNoDisponible}
	s, _ := NuevoEnsayoCalendarioHistorico(c)
	for _, cantidad := range []int{0, 2} {
		sol := solicitudEnsayoMinima(t)
		for i := 0; i < cantidad; i++ {
			ref := "adscripcion:uno"
			if i > 0 {
				ref = "adscripcion:dos"
			}
			sol.Snapshot.Adscripciones = append(sol.Snapshot.Adscripciones, ports.AdscripcionEnsayoCalendario{Ref: ref, Version: 1, CentroRef: "centro:demo", Desde: sol.Desde, Hasta: sol.Hasta, Fuente: sol.Snapshot.Fuente})
		}
		r, err := s.Consultar(context.Background(), sol)
		if err != nil || r.Estado != "indeterminado" || c.llamadas != 0 || len(r.Tramos) != 1 || r.Tramos[0].Calendario != nil {
			t.Fatalf("r=%+v err=%v llamadas=%d", r, err, c.llamadas)
		}
	}
}

func TestEnsayoFallosYCancelacionNoDeterminanCalendario(t *testing.T) {
	c := &consultaEnsayoFallida{err: errors.New("fallo:demo")}
	s, _ := NuevoEnsayoCalendarioHistorico(c)
	sol := solicitudEnsayoMinima(t)
	sol.Snapshot.Adscripciones = []ports.AdscripcionEnsayoCalendario{{Ref: "adscripcion:uno", Version: 1, CentroRef: "centro:demo", Desde: sol.Desde, Hasta: sol.Hasta, Fuente: sol.Snapshot.Fuente}}
	r, err := s.Consultar(context.Background(), sol)
	if err != nil || r.Estado != "indeterminado" || r.Tramos[0].Carencias[0] != "calendario_no_disponible" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = s.Consultar(ctx, sol); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	c.err = context.DeadlineExceeded
	if _, err = s.Consultar(context.Background(), sol); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestEnsayoDependenciasYEntrada(t *testing.T) {
	var dependencia *consultaEnsayoFallida
	if _, err := NuevoEnsayoCalendarioHistorico(dependencia); err == nil {
		t.Fatal("nil")
	}
	s, _ := NuevoEnsayoCalendarioHistorico(&consultaEnsayoFallida{})
	sol := solicitudEnsayoMinima(t)
	sol.ConocidoEn = time.Time{}
	if _, err := s.Consultar(context.Background(), sol); !errors.Is(err, ErrEnsayoCalendarioInvalido) {
		t.Fatal(err)
	}
	sol = solicitudEnsayoMinima(t)
	sol.Snapshot.ConocidoEn = sol.ConocidoEn.Add(time.Second)
	if _, err := s.Consultar(context.Background(), sol); !errors.Is(err, ErrEnsayoCalendarioInvalido) {
		t.Fatal(err)
	}
	sol = solicitudEnsayoMinima(t)
	sol.Hasta, _ = cal.ParsearFechaCivil("2100-01-01")
	if _, err := s.Consultar(context.Background(), sol); !errors.Is(err, ErrEnsayoCalendarioInvalido) {
		t.Fatal(err)
	}
}
