package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type contextoCotejoCanceladoTrasPrimeraConsulta struct {
	context.Context
	consultas atomic.Uint32
	terminado chan struct{}
	cancelar  sync.Once
}

func nuevoContextoCotejoCanceladoTrasPrimeraConsulta() *contextoCotejoCanceladoTrasPrimeraConsulta {
	return &contextoCotejoCanceladoTrasPrimeraConsulta{
		Context: context.Background(), terminado: make(chan struct{}),
	}
}

func (c *contextoCotejoCanceladoTrasPrimeraConsulta) Done() <-chan struct{} { return c.terminado }

func (c *contextoCotejoCanceladoTrasPrimeraConsulta) Err() error {
	if c.consultas.Add(1) == 1 {
		return nil
	}
	c.cancelar.Do(func() { close(c.terminado) })
	return context.Canceled
}

func TestCotejoMemoriaConfirmarYAbandonarSeLinealizan(t *testing.T) {
	const intentos = 16
	for intento := 0; intento < intentos; intento++ {
		store := NewStore()
		fecha := time.Date(2026, time.July, 16, 10, intento, 0, 0, time.UTC)
		documento := domain.ReferenciaDocumento{ID: "documento-cotejo-concurrente-001", Version: 1}
		solicitud := cotejoMemoriaPruebaSolicitudReserva(
			"idempotencia-cotejo-concurrente-001", documento, cotejoMemoriaPruebaSolicitudA, fecha,
		)
		reserva, err := store.ReservarEmisionCodigoCotejo(context.Background(), solicitud)
		if err != nil {
			t.Fatalf("intento %d: reservar codigo: %v", intento, err)
		}
		codigo := cotejoMemoriaPruebaCodigoReservado(
			"codigo-cotejo-concurrente-001", documento, cotejoMemoriaPruebaIndiceA,
			cotejoMemoriaPruebaProteccionA, fecha,
		)
		cotejoMemoriaPruebaSembrarDependenciasReserva(store, codigo)
		confirmadaEn := fecha.Add(time.Minute)
		traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(
			codigo, domain.AccionCodigoCotejoReservado, "", confirmadaEn,
		)

		inicio := make(chan struct{})
		resultados := make(chan error, 2)
		go func() {
			<-inicio
			resultados <- store.ConfirmarReservaCodigoCotejo(
				context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC,
				confirmadaEn, codigo, traza, evento,
			)
		}()
		go func() {
			<-inicio
			resultados <- store.AbandonarReservaCodigoCotejo(context.Background(), reserva.Token)
		}()
		close(inicio)

		exitos, rechazadas := 0, 0
		for operacion := 0; operacion < 2; operacion++ {
			err := <-resultados
			switch {
			case err == nil:
				exitos++
			case errors.Is(err, ports.ErrReservaCodigoCotejoNoValida):
				rechazadas++
			default:
				t.Fatalf("intento %d: resultado concurrente inesperado: %v", intento, err)
			}
		}
		if exitos != 1 || rechazadas != 1 {
			t.Fatalf("intento %d: exitos=%d rechazadas=%d", intento, exitos, rechazadas)
		}

		claveAmbito := claveAmbitoCotejo(solicitud.PrincipalID, solicitud.ClaveIdempotencia)
		guardada := store.reservasCotejo[claveAmbito]
		huellaToken := huellaTokenReservaCotejoPrueba(t, reserva.Token)
		if _, existe := store.reservasCotejoPorHuellaToken[huellaToken]; existe {
			t.Fatalf("intento %d: la capacidad consumida conserva su selector", intento)
		}
		switch guardada.Estado {
		case estadoReservaCotejoConfirmada:
			if len(store.codigosCotejo) != 1 || len(store.cotejoPorDocumento) != 1 ||
				len(store.cotejoPorIndice) != 1 || len(store.audit) != 1 || len(store.events) != 1 ||
				guardada.Codigo.Validar() != nil {
				t.Fatalf("intento %d: confirmacion no atomica: reserva=%+v codigos=%d documentos=%d indices=%d auditoria=%d eventos=%d",
					intento, guardada, len(store.codigosCotejo), len(store.cotejoPorDocumento),
					len(store.cotejoPorIndice), len(store.audit), len(store.events))
			}
		case estadoReservaCotejoAbandonada:
			if len(store.codigosCotejo) != 0 || len(store.cotejoPorDocumento) != 0 ||
				len(store.cotejoPorIndice) != 0 || len(store.audit) != 0 || len(store.events) != 0 {
				t.Fatalf("intento %d: el abandono dejo efectos parciales", intento)
			}
		default:
			t.Fatalf("intento %d: estado final no linealizable: %q", intento, guardada.Estado)
		}

		if err := store.ConfirmarReservaCodigoCotejo(
			context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC,
			confirmadaEn, codigo, traza, evento,
		); !errors.Is(err, ports.ErrReservaCodigoCotejoNoValida) {
			t.Fatalf("intento %d: replay de confirmacion: %v", intento, err)
		}
		if err := store.AbandonarReservaCodigoCotejo(context.Background(), reserva.Token); !errors.Is(err, ports.ErrReservaCodigoCotejoNoValida) {
			t.Fatalf("intento %d: replay de abandono: %v", intento, err)
		}
	}
}

func TestCotejoMemoriaRevalidaCancelacionDentroDelMutex(t *testing.T) {
	fecha := time.Date(2026, time.July, 16, 11, 0, 0, 0, time.UTC)
	documento := domain.ReferenciaDocumento{ID: "documento-cotejo-cancelacion-001", Version: 1}
	solicitud := cotejoMemoriaPruebaSolicitudReserva(
		"idempotencia-cotejo-cancelacion-001", documento, cotejoMemoriaPruebaSolicitudA, fecha,
	)

	t.Run("reservar", func(t *testing.T) {
		store := NewStore()
		ctx := nuevoContextoCotejoCanceladoTrasPrimeraConsulta()
		if _, err := store.ReservarEmisionCodigoCotejo(ctx, solicitud); !errors.Is(err, context.Canceled) {
			t.Fatalf("reserva con cancelacion adquirida durante el bloqueo: %v", err)
		}
		if len(store.reservasCotejo) != 0 || len(store.reservasCotejoPorHuellaToken) != 0 {
			t.Fatal("la reserva cancelada dejo estado")
		}
	})

	t.Run("confirmar y abandonar", func(t *testing.T) {
		store := NewStore()
		reserva, err := store.ReservarEmisionCodigoCotejo(context.Background(), solicitud)
		if err != nil {
			t.Fatalf("preparar reserva: %v", err)
		}
		codigo := cotejoMemoriaPruebaCodigoReservado(
			"codigo-cotejo-cancelacion-001", documento, cotejoMemoriaPruebaIndiceA,
			cotejoMemoriaPruebaProteccionA, fecha,
		)
		cotejoMemoriaPruebaSembrarDependenciasReserva(store, codigo)
		confirmadaEn := fecha.Add(time.Minute)
		traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(
			codigo, domain.AccionCodigoCotejoReservado, "", confirmadaEn,
		)
		huellaToken := huellaTokenReservaCotejoPrueba(t, reserva.Token)

		ctxConfirmar := nuevoContextoCotejoCanceladoTrasPrimeraConsulta()
		if err := store.ConfirmarReservaCodigoCotejo(
			ctxConfirmar, reserva.Token, solicitud.HuellaSolicitudHMAC,
			confirmadaEn, codigo, traza, evento,
		); !errors.Is(err, context.Canceled) {
			t.Fatalf("confirmacion con cancelacion adquirida durante el bloqueo: %v", err)
		}
		claveAmbito := claveAmbitoCotejo(solicitud.PrincipalID, solicitud.ClaveIdempotencia)
		if store.reservasCotejo[claveAmbito].Estado != estadoReservaCotejoActiva ||
			len(store.codigosCotejo) != 0 || len(store.audit) != 0 || len(store.events) != 0 {
			t.Fatal("la confirmacion cancelada dejo efectos")
		}
		if _, existe := store.reservasCotejoPorHuellaToken[huellaToken]; !existe {
			t.Fatal("la confirmacion cancelada consumio la capacidad")
		}

		ctxAbandonar := nuevoContextoCotejoCanceladoTrasPrimeraConsulta()
		if err := store.AbandonarReservaCodigoCotejo(ctxAbandonar, reserva.Token); !errors.Is(err, context.Canceled) {
			t.Fatalf("abandono con cancelacion adquirida durante el bloqueo: %v", err)
		}
		if store.reservasCotejo[claveAmbito].Estado != estadoReservaCotejoActiva {
			t.Fatal("el abandono cancelado altero la reserva")
		}
		if _, existe := store.reservasCotejoPorHuellaToken[huellaToken]; !existe {
			t.Fatal("el abandono cancelado consumio la capacidad")
		}
	})
}
