package ginpixapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestOperacionGINPIXRecuperableReservaVarianteFallaAntesDeEfecto(t *testing.T) {
	solicitud, base := solicitudOperacionRecuperablePrueba(t)
	variantes := map[string]func(*ports.ReservaOperacionGINPIX){
		"reserva":        func(r *ports.ReservaOperacionGINPIX) { r.ReservaRef = "" },
		"clave":          func(r *ports.ReservaOperacionGINPIX) { r.ClaveOperacionRef += "-otra" },
		"intento":        func(r *ports.ReservaOperacionGINPIX) { r.Intento = 0 },
		"situacion":      func(r *ports.ReservaOperacionGINPIX) { r.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion },
		"situacion cero": func(r *ports.ReservaOperacionGINPIX) { r.Situacion = 0 },
	}
	for nombre, mutar := range variantes {
		t.Run(nombre, func(t *testing.T) {
			reserva := base
			mutar(&reserva)
			transporte := transporteQueNoDebeInvocarse(t)
			adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
			recibo, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitud, reserva)
			if recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
				!errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) ||
				transporte.total() != 0 {
				t.Fatalf("reserva variante alcanzo efecto: %#v / %v", recibo, err)
			}
		})
	}
	t.Run("consulta exige pendiente", func(t *testing.T) {
		transporte := transporteQueNoDebeInvocarse(t)
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		recibo, err := adaptador.ConsultarOperacionGINPIX(context.Background(), solicitud, base)
		if recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
			!errors.Is(err, ports.ErrConsultaOperacionGINPIXNoDisponible) || transporte.total() != 0 {
			t.Fatalf("consulta con reserva de emision alcanzo efecto: %#v / %v", recibo, err)
		}
	})
}

func TestOperacionGINPIXRecuperableCancelacionConservaFronteraYCausa(t *testing.T) {
	solicitud, reserva := solicitudOperacionRecuperablePrueba(t)

	t.Run("antes", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		transporte := transporteQueNoDebeInvocarse(t)
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.EmitirOperacionGINPIX(ctx, solicitud, reserva)
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) ||
			errors.Is(err, ports.ErrEmisionOperacionGINPIXIndeterminada) || transporte.total() != 0 {
			t.Fatalf("cancelacion previa perdio causalidad: %v", err)
		}
	})

	t.Run("tras autenticar antes de roundtrip", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		transporte := transporteQueNoDebeInvocarse(t)
		autenticador := &autenticadorCancelaSinError{cancelar: cancelar}
		adaptador := nuevoAdaptadorPrueba(t, transporte, autenticador, politicaPrueba())
		_, err := adaptador.EmitirOperacionGINPIX(ctx, solicitud, reserva)
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) ||
			errors.Is(err, ports.ErrEmisionOperacionGINPIXIndeterminada) ||
			transporte.total() != 0 || autenticador.total.Load() != 1 {
			t.Fatalf("cancelacion pre-roundtrip perdio causalidad: %v", err)
		}
	})

	t.Run("durante", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			cancelar()
			return nil, context.Canceled
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.EmitirOperacionGINPIX(ctx, solicitud, reserva)
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ports.ErrEmisionOperacionGINPIXIndeterminada) || transporte.total() != 1 {
			t.Fatalf("cancelacion posterior perdio causalidad: %v", err)
		}
	})

	t.Run("consulta antes", func(t *testing.T) {
		reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		transporte := transporteQueNoDebeInvocarse(t)
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.ConsultarOperacionGINPIX(ctx, solicitud, reserva)
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ports.ErrConsultaOperacionGINPIXNoDisponible) || transporte.total() != 0 {
			t.Fatalf("cancelacion de consulta perdio causalidad: %v", err)
		}
	})

	t.Run("consulta durante", func(t *testing.T) {
		reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
		ctx, cancelar := context.WithCancel(context.Background())
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			cancelar()
			return nil, context.Canceled
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.ConsultarOperacionGINPIX(ctx, solicitud, reserva)
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ports.ErrConsultaOperacionGINPIXNoDisponible) || transporte.total() != 1 {
			t.Fatalf("cancelacion durante consulta perdio causalidad: %v", err)
		}
	})
}

func TestOperacionGINPIXRecuperableNilYNilTipadoFallanCerrados(t *testing.T) {
	solicitud, reserva := solicitudOperacionRecuperablePrueba(t)
	var adaptador *Adaptador
	if recibo, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitud, reserva); recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
		!errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) {
		t.Fatalf("adaptador nil tipado aceptado: %#v / %v", recibo, err)
	}
	reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
	if recibo, err := adaptador.ConsultarOperacionGINPIX(context.Background(), solicitud, reserva); recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
		!errors.Is(err, ports.ErrConsultaOperacionGINPIXNoDisponible) {
		t.Fatalf("consultor nil tipado aceptado: %#v / %v", recibo, err)
	}
	adaptador = nuevoAdaptadorPrueba(t, transporteQueNoDebeInvocarse(t), &autenticadorFalso{}, politicaPrueba())
	reserva.Situacion = ports.ReservaOperacionGINPIXEmisionAutorizada
	if _, err := adaptador.EmitirOperacionGINPIX(nil, solicitud, reserva); !errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) {
		t.Fatalf("contexto nil aceptado: %v", err)
	}
	var solicitudCero ports.SolicitudOperacionGINPIX
	if _, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitudCero, reserva); !errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) {
		t.Fatalf("solicitud cero aceptada: %v", err)
	}
}

func TestOperacionGINPIXRecuperableNoFiltraDatosSensibles(t *testing.T) {
	solicitud, reserva := solicitudOperacionRecuperablePrueba(t)
	marcador := "SECRETO-PRIVADO-NO-EXPONER"
	autenticador := &autenticadorFalso{err: errors.New(marcador)}
	adaptador := nuevoAdaptadorPrueba(t, transporteQueNoDebeInvocarse(t), autenticador, politicaPrueba())
	_, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitud, reserva)
	if !errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) ||
		strings.Contains(err.Error(), marcador) ||
		strings.Contains(err.Error(), datoPersonalSinteticoPrueba) ||
		strings.Contains(err.Error(), secretoSinteticoPrueba) {
		t.Fatalf("error no saneado: %v", err)
	}
}
