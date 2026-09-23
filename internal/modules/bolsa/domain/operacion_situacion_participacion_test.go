package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func cambioOperacionValido(destino string) CambioSituacionParticipacion {
	registrada := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	return CambioSituacionParticipacion{
		ParticipacionRef: "participacion:bolsa:demo:0001",
		Origen:           SituacionDisponible,
		Destino:          destino,
		Desde:            registrada,
		Motivo:           "solicitud de la persona candidata",
		RegistradaEn:     registrada,
	}
}

func operacionValida(operacion, destino string) OperacionSituacionParticipacion {
	return OperacionSituacionParticipacion{
		Cambio:    cambioOperacionValido(destino),
		Operacion: operacion,
		Justificante: JustificanteOperacionSituacion{
			Tipo:       JustificanteSolicitudCandidato,
			Referencia: "custodia:bolsa:justificante:0001",
			SHA256:     strings.Repeat("ab", 32),
		},
		Actor:      "per_tecnica_rrhh",
		Validador:  "per_jefatura_rrhh",
		ValidadaEn: time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC),
	}
}

func TestOperacionSituacionParticipacionValidaLasTresOperaciones(t *testing.T) {
	casos := map[string]string{
		OperacionPausar:    SituacionNoDisponible,
		OperacionReactivar: SituacionDisponible,
		OperacionExcluir:   SituacionExcluido,
	}
	for operacion, destino := range casos {
		o := operacionValida(operacion, destino)
		if operacion == OperacionReactivar {
			o.Cambio.Origen = SituacionNoDisponible
		}
		if err := o.Validar(); err != nil {
			t.Fatalf("%s deberia ser valida: %v", operacion, err)
		}
		if obtenido, ok := DestinoOperacionSituacion(operacion); !ok || obtenido != destino {
			t.Fatalf("%s deberia producir %q, produjo %q (%t)", operacion, destino, obtenido, ok)
		}
	}
}

func TestOperacionSituacionParticipacionRechazaDestinoIncoherente(t *testing.T) {
	o := operacionValida(OperacionPausar, SituacionExcluido)
	if err := o.Validar(); !errors.Is(err, ErrOperacionSituacionParticipacionInvalida) {
		t.Fatalf("una pausa que excluye debe rechazarse, dio %v", err)
	}
}

func TestOperacionSituacionParticipacionRechazaOperacionDesconocida(t *testing.T) {
	o := operacionValida("archivar", SituacionNoDisponible)
	if err := o.Validar(); !errors.Is(err, ErrOperacionSituacionParticipacionInvalida) {
		t.Fatalf("una operacion fuera del catalogo debe rechazarse, dio %v", err)
	}
}

func TestOperacionSituacionParticipacionExigeJustificanteCompleto(t *testing.T) {
	for nombre, romper := range map[string]func(*OperacionSituacionParticipacion){
		"tipo fuera de catalogo":  func(o *OperacionSituacionParticipacion) { o.Justificante.Tipo = "invento" },
		"referencia vacia":        func(o *OperacionSituacionParticipacion) { o.Justificante.Referencia = "" },
		"referencia sin recortar": func(o *OperacionSituacionParticipacion) { o.Justificante.Referencia = " custodia:x " },
		"huella corta":            func(o *OperacionSituacionParticipacion) { o.Justificante.SHA256 = "ab" },
		"huella en mayusculas":    func(o *OperacionSituacionParticipacion) { o.Justificante.SHA256 = strings.Repeat("AB", 32) },
	} {
		o := operacionValida(OperacionPausar, SituacionNoDisponible)
		romper(&o)
		if err := o.Validar(); !errors.Is(err, ErrOperacionSituacionParticipacionInvalida) {
			t.Fatalf("%s deberia invalidar la operacion, dio %v", nombre, err)
		}
	}
}

func TestExclusionExigeValidadorDistintoYLasDemasNo(t *testing.T) {
	excluir := operacionValida(OperacionExcluir, SituacionExcluido)
	excluir.Validador = excluir.Actor
	if err := excluir.Validar(); !errors.Is(err, ErrOperacionSituacionParticipacionInvalida) {
		t.Fatalf("excluir con un solo interviniente debe rechazarse, dio %v", err)
	}
	pausar := operacionValida(OperacionPausar, SituacionNoDisponible)
	pausar.Validador = pausar.Actor
	if err := pausar.Validar(); err != nil {
		t.Fatalf("la pausa no exige segunda persona mientras RRHH no lo decida: %v", err)
	}
	if !ExigeValidadorDistinto(OperacionExcluir) || ExigeValidadorDistinto(OperacionPausar) {
		t.Fatal("la regla provisional solo alcanza a la exclusion")
	}
}

func TestOperacionSituacionParticipacionRechazaValidacionPosterior(t *testing.T) {
	o := operacionValida(OperacionPausar, SituacionNoDisponible)
	o.ValidadaEn = o.Cambio.RegistradaEn.Add(time.Second)
	if err := o.Validar(); !errors.Is(err, ErrOperacionSituacionParticipacionInvalida) {
		t.Fatalf("no se puede validar despues de registrar, dio %v", err)
	}
}

func TestOperacionesDisponiblesDesdeRespetanLasTransicionesDeB2(t *testing.T) {
	casos := map[string][]string{
		SituacionDisponible:   {OperacionPausar, OperacionExcluir},
		SituacionNoDisponible: {OperacionReactivar, OperacionExcluir},
		SituacionTrabajando:   {OperacionReactivar, OperacionExcluir},
		SituacionExcluido:     {},
	}
	for origen, esperadas := range casos {
		obtenidas := OperacionesDisponiblesDesde(origen)
		if len(obtenidas) != len(esperadas) {
			t.Fatalf("desde %q se esperaban %v, se obtuvo %v", origen, esperadas, obtenidas)
		}
		for i, esperada := range esperadas {
			if obtenidas[i] != esperada {
				t.Fatalf("desde %q se esperaba %v, se obtuvo %v", origen, esperadas, obtenidas)
			}
		}
	}
}
