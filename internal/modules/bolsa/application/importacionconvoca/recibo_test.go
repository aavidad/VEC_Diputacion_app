package importacionconvoca_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	memoria "vec-diputacion-granada/internal/modules/bolsa/adapters/memory"
	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
)

func TestReciboImportacionDistingueAltaYReplaySinDatosDeFilas(t *testing.T) {
	instante := time.Date(2026, 9, 28, 10, 15, 0, 0, time.UTC)
	servicio, err := aplicacion.NuevoServicio(decodificadorResumenValido{}, memoria.NuevoRepositorioImportacionesConvoca(), func() time.Time {
		ahora := instante
		instante = instante.Add(time.Hour)
		return ahora
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudValida()
	var primero aplicacion.ReciboImportacion
	for intento := 0; intento < 2; intento++ {
		resultado, err := servicio.Importar(context.Background(), solicitud)
		if err != nil {
			t.Fatalf("importacion %d: %v", intento, err)
		}
		recibo, err := resultado.Recibo()
		if err != nil {
			t.Fatalf("recibo %d: %v", intento, err)
		}
		if intento == 0 {
			primero = recibo
			if recibo.Estado != aplicacion.EstadoReciboNueva {
				t.Fatalf("primera importacion: %q", recibo.Estado)
			}
		} else {
			if recibo.Estado != aplicacion.EstadoReciboReutilizada || recibo.ActaRef != primero.ActaRef ||
				recibo.ImportacionRef != primero.ImportacionRef || !recibo.RegistradaEn.Equal(primero.RegistradaEn) {
				t.Fatalf("replay sin recibo original: primero=%+v segundo=%+v", primero, recibo)
			}
		}
		if recibo.Esquema != aplicacion.EsquemaReciboImportacion ||
			recibo.Autoridad != "no_autoritativa" || recibo.HabilitaActosConEfectos || !recibo.RequiereConfirmacionRegistro ||
			recibo.FilasLeidas != 1 || recibo.FilasAceptadas != 1 || recibo.FilasRechazadas != 0 {
			t.Fatalf("recibo incompleto o autoridad incorrecta: %+v", recibo)
		}
		serializado, err := json.Marshal(recibo)
		if err != nil {
			t.Fatal(err)
		}
		for _, sensible := range []string{solicitud.NombreFichero, solicitud.FicheroCustodiadoRef, solicitud.ActorRef, "Persona"} {
			if strings.Contains(string(serializado), sensible) {
				t.Fatalf("recibo revela %q", sensible)
			}
		}
	}
}

func TestReciboImportacionRechazaActaInvalida(t *testing.T) {
	if _, err := (aplicacion.ResultadoImportacion{}).Recibo(); !errors.Is(err, aplicacion.ErrResultadoInseguro) {
		t.Fatalf("acta vacia aceptada: %v", err)
	}
}
