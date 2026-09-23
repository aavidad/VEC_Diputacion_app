package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestCatalogoTemaCerradoYRevisionExacta(t *testing.T) {
	for _, id := range []TemaID{TemaInstitucional, TemaGranate} {
		if err := (Tema{ID: id, Revision: RevisionTemaActual}).Validar(); err != nil {
			t.Fatalf("tema incluido %q: %v", id, err)
		}
	}
	for _, tema := range []Tema{
		{}, {ID: "externo", Revision: 1}, {ID: TemaGranate, Revision: 0},
		{ID: TemaGranate, Revision: 2}, {ID: "granate;url(https://example.invalid)", Revision: 1},
	} {
		if !errors.Is(tema.Validar(), ErrTemaInvalido) {
			t.Fatalf("tema ajeno admitido: %+v", tema)
		}
	}
}

func TestEstadoGlobalDistingueAusenciaYConfirmacionDurable(t *testing.T) {
	if err := (EstadoApariencia{}).Validar(); err != nil {
		t.Fatalf("ausencia legítima: %v", err)
	}
	vigente := EstadoApariencia{
		Configurado: true, Tema: Tema{ID: TemaInstitucional, Revision: 1},
		RevisionGlobal: 7, ReciboRef: "recibo:tema:7", HistoriaRef: "historia:tema:7",
		PublicadaEn: time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC),
	}
	if err := vigente.Validar(); err != nil {
		t.Fatalf("estado confirmado: %v", err)
	}
	mutaciones := []func(*EstadoApariencia){
		func(e *EstadoApariencia) { e.RevisionGlobal = 0 },
		func(e *EstadoApariencia) { e.ReciboRef = "" },
		func(e *EstadoApariencia) { e.HistoriaRef = "" },
		func(e *EstadoApariencia) { e.PublicadaEn = e.PublicadaEn.In(time.FixedZone("CET", 3600)) },
		func(e *EstadoApariencia) { e.Tema.Revision = 2 },
		func(e *EstadoApariencia) { e.Configurado = false },
	}
	for i, mutar := range mutaciones {
		candidato := vigente
		mutar(&candidato)
		if !errors.Is(candidato.Validar(), ErrEstadoAparienciaInvalido) {
			t.Fatalf("estado incompleto %d admitido", i)
		}
	}
}

func TestOrdenPublicacionExigeCASYClaveOpaca(t *testing.T) {
	orden := OrdenPublicacion{
		Tema:                   Tema{ID: TemaGranate, Revision: 1},
		RevisionGlobalEsperada: 0,
		ClaveIdempotencia:      "tema-publicacion-0001",
	}
	if siguiente, err := orden.RevisionSiguiente(); err != nil || siguiente != 1 {
		t.Fatalf("primera revisión: siguiente=%d err=%v", siguiente, err)
	}
	orden.RevisionGlobalEsperada = 41
	if siguiente, err := orden.RevisionSiguiente(); err != nil || siguiente != 42 {
		t.Fatalf("CAS: siguiente=%d err=%v", siguiente, err)
	}
	orden.RevisionGlobalEsperada = math.MaxUint64
	if !errors.Is(orden.Validar(), ErrOrdenPublicacionInvalida) {
		t.Fatal("desbordamiento admitido")
	}
	orden.RevisionGlobalEsperada = 1
	for _, clave := range []string{"", "corta", " clave-con-espacio ", "tema-publicacion-\n001"} {
		orden.ClaveIdempotencia = clave
		if !errors.Is(orden.Validar(), ErrOrdenPublicacionInvalida) {
			t.Fatalf("clave inválida admitida: %q", clave)
		}
	}
}
