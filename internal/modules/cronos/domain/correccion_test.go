package domain

import (
	"errors"
	"testing"
	"time"
)

func solicitudCorreccionValida() SolicitudCorreccion {
	return SolicitudCorreccion{
		EmpleadoRef: "emp_0123456789abcdefghijkl", ActorRef: "per_0123456789abcdefghijkl",
		PerfilRef: "prf_0123456789abcdefghijkl", ClaveOperacion: "olvido_0001",
		HuecoDeclarado: true, Movimiento: PunchEntry,
		FechaCivil: "2026-09-24", HoraPretendida: "08:15", MotivoCodigo: MotivoOlvidoMarcaje,
		SolicitadaEnUTC: time.Date(2026, 9, 24, 9, 0, 0, 123456000, time.UTC),
	}
}

func TestSolicitudCorreccionExigeHuecoUOriginalYHoraCivilValida(t *testing.T) {
	s := solicitudCorreccionValida()
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	s.MarcajeOriginalRef = "marcaje:cronos:original_0001"
	if !errors.Is(s.Validar(), ErrCorreccionInvalida) {
		t.Fatal("acepto original y hueco simultaneos")
	}
	s.HuecoDeclarado = false
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	s.FechaCivil = "2026-02-30"
	if !errors.Is(s.Validar(), ErrCorreccionInvalida) {
		t.Fatal("acepto fecha civil inexistente")
	}
	s.FechaCivil = "2026-09-24"
	s.HoraPretendida = "24:00"
	if !errors.Is(s.Validar(), ErrCorreccionInvalida) {
		t.Fatal("acepto hora inexistente")
	}
}

func TestCorreccionSoloSeAplicaTrasDosDecisionesFavorables(t *testing.T) {
	estado, err := SiguienteEstadoCorreccion(CorreccionPendienteResponsable, PasoDecisionResponsable, ResultadoFavorable)
	if err != nil || estado != CorreccionPendienteRRHH {
		t.Fatal(estado, err)
	}
	if _, err := SiguienteEstadoCorreccion(estado, PasoAplicacion, ""); !errors.Is(err, ErrCorreccionInvalida) {
		t.Fatal("aplicacion antes de RRHH")
	}
	estado, err = SiguienteEstadoCorreccion(estado, PasoResolucionRRHH, ResultadoFavorable)
	if err != nil || estado != CorreccionPendienteAplicacion {
		t.Fatal(estado, err)
	}
	estado, err = SiguienteEstadoCorreccion(estado, PasoAplicacion, "")
	if err != nil || estado != CorreccionAplicada {
		t.Fatal(estado, err)
	}
	if _, err := SiguienteEstadoCorreccion(estado, PasoAplicacion, ""); !errors.Is(err, ErrCorreccionInvalida) {
		t.Fatal("segunda aplicacion")
	}
	denegada, err := SiguienteEstadoCorreccion(CorreccionPendienteResponsable, PasoDecisionResponsable, ResultadoDesfavorable)
	if err != nil || denegada != CorreccionDenegadaResponsable {
		t.Fatal(denegada, err)
	}
	if _, err := SiguienteEstadoCorreccion(denegada, PasoResolucionRRHH, ResultadoFavorable); !errors.Is(err, ErrCorreccionInvalida) {
		t.Fatal("resolucion tras denegacion terminal")
	}
}

func TestHuellaCorreccionPermiteReplayPeroDetectaOtroContenido(t *testing.T) {
	s := solicitudCorreccionValida()
	primera, err := s.HuellaSemantica()
	if err != nil {
		t.Fatal(err)
	}
	s.SolicitadaEnUTC = s.SolicitadaEnUTC.Add(time.Minute)
	reintento, err := s.HuellaSemantica()
	if err != nil || primera != reintento {
		t.Fatal("la hora de recepción alteró el replay", err)
	}
	s.HoraPretendida = "08:16"
	alterada, err := s.HuellaSemantica()
	if err != nil || primera == alterada {
		t.Fatal("el cambio semántico conservó la huella", err)
	}
}
