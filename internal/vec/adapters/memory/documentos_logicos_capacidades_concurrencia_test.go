package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestRepositorioDocumentosLogicosConfirmarYAbandonarSeLinealizan(t *testing.T) {
	const intentos = 16
	for intento := 0; intento < intentos; intento++ {
		store := NewStore()
		fecha := time.Date(2026, time.July, 16, 9, intento, 0, 0, time.UTC)
		solicitud := solicitudReservaLogicaPrueba("tecnico-rrhh-concurrencia", fecha)
		reserva, err := store.ReservarGeneracion(context.Background(), solicitud)
		if err != nil {
			t.Fatalf("intento %d: reservar generacion: %v", intento, err)
		}
		resultado := resultadoDocumentoLogicoMemoriaPrueba(t, store, solicitud.PrincipalID, fecha.Add(time.Minute))
		traza, evento := evidenciaDocumentoLogicoMemoriaPrueba(resultado, fecha.Add(time.Minute))

		inicio := make(chan struct{})
		resultados := make(chan error, 2)
		go func() {
			<-inicio
			resultados <- store.ConfirmarGeneracionLogica(
				context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC,
				fecha.Add(time.Minute), resultado, traza, evento,
			)
		}()
		go func() {
			<-inicio
			resultados <- store.AbandonarGeneracion(context.Background(), reserva.Token)
		}()
		close(inicio)

		exitos, rechazadas := 0, 0
		for operacion := 0; operacion < 2; operacion++ {
			err := <-resultados
			switch {
			case err == nil:
				exitos++
			case errors.Is(err, ports.ErrReservaDocumentoNoValida):
				rechazadas++
			default:
				t.Fatalf("intento %d: resultado concurrente inesperado: %v", intento, err)
			}
		}
		if exitos != 1 || rechazadas != 1 {
			t.Fatalf("intento %d: exitos=%d rechazadas=%d", intento, exitos, rechazadas)
		}

		claveAmbito := claveAmbitoIdempotenciaDocumento(solicitud.PrincipalID, solicitud.ClaveIdempotencia)
		guardada := store.reservasDocumentales[claveAmbito]
		huellaToken := huellaTokenReservaDocumentoPrueba(t, reserva.Token)
		if _, existe := store.reservasPorHuellaToken[huellaToken]; existe {
			t.Fatalf("intento %d: la capacidad consumida conserva su selector", intento)
		}
		switch guardada.Estado {
		case estadoReservaDocumentalConfirmada:
			if len(store.documentosLogicos) != 1 || len(store.representaciones) != 2 ||
				len(store.audit) != 1 || len(store.events) != 1 || guardada.Resultado.Validar() != nil {
				t.Fatalf("intento %d: confirmacion no atomica: reserva=%+v documentos=%d representaciones=%d auditoria=%d eventos=%d",
					intento, guardada, len(store.documentosLogicos), len(store.representaciones), len(store.audit), len(store.events))
			}
		case estadoReservaDocumentalAbandonada:
			if len(store.documentosLogicos) != 0 || len(store.representaciones) != 0 ||
				len(store.audit) != 0 || len(store.events) != 0 {
				t.Fatalf("intento %d: el abandono dejo efectos parciales", intento)
			}
		default:
			t.Fatalf("intento %d: estado final no linealizable: %q", intento, guardada.Estado)
		}

		if err := store.ConfirmarGeneracionLogica(
			context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC,
			fecha.Add(time.Minute), resultado, traza, evento,
		); !errors.Is(err, ports.ErrReservaDocumentoNoValida) {
			t.Fatalf("intento %d: replay de confirmacion: %v", intento, err)
		}
		if err := store.AbandonarGeneracion(context.Background(), reserva.Token); !errors.Is(err, ports.ErrReservaDocumentoNoValida) {
			t.Fatalf("intento %d: replay de abandono: %v", intento, err)
		}
	}
}
