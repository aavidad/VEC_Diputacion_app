package cobertura_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestSesionTCBFragmentosSonOpacosRedactadosYDefensivos(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(escenario.reciboConcedido),
	}
	transaccion, _ :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if _, err := cobertura.ConfirmarOperacionDecisionCobertura(
		context.Background(),
		transaccion,
		escenario.ordenConcedida,
	); err != nil {
		t.Fatal(err)
	}
	sesion := ejecutor.ultimaSesion()
	ejecutorDenegado := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(escenario.reciboDenegado),
	}
	transaccionDenegada, _ :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
			ejecutorDenegado,
		)
	if _, err := cobertura.ConfirmarOperacionDecisionCobertura(
		context.Background(),
		transaccionDenegada,
		escenario.ordenDenegada,
	); err != nil {
		t.Fatal(err)
	}
	sesionDenegada := ejecutorDenegado.ultimaSesion()
	sesion.mu.Lock()
	gobierno := sesion.gobiernoOpaco
	efecto := sesion.concesionOpaca
	cabecera := sesion.cabeceraOpaca
	datosCabecera := sesion.cabecera
	datosGobierno := sesion.gobierno
	consumo := sesion.consumosOpacos[0]
	datosConsumo := sesion.consumos[0]
	datosEfecto := sesion.concesion
	sesion.mu.Unlock()
	sesionDenegada.mu.Lock()
	terminal := sesionDenegada.denegacionOpaca
	datosTerminal := sesionDenegada.denegacion
	sesionDenegada.mu.Unlock()

	primeroGobierno, err := gobierno.Datos()
	if err != nil {
		t.Fatal(err)
	}
	primeroGobierno.Catalogo.Vias[0].Clave = "via_mutada"
	segundoGobierno, err := gobierno.Datos()
	if err != nil ||
		segundoGobierno.Catalogo.Vias[0].Clave == "via_mutada" {
		t.Fatal("el gobierno compartió colecciones mutables")
	}
	primeroEfecto, err := efecto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	primeroEfecto.AgregadoSiguiente.Referencia = "expediente_mutado"
	if len(primeroEfecto.Propuesta.Resultados) != 0 {
		primeroEfecto.Propuesta.Resultados[0].Clave = "comprobacion_mutada"
	}
	segundoEfecto, err := efecto.Datos()
	if err != nil ||
		segundoEfecto.AgregadoSiguiente.Referencia == "expediente_mutado" ||
		(len(segundoEfecto.Propuesta.Resultados) != 0 &&
			segundoEfecto.Propuesta.Resultados[0].Clave ==
				"comprobacion_mutada") {
		t.Fatal("el efecto compartió agregado o propuesta mutables")
	}

	crudo := datosReciboSesionTCBPrueba(escenario.reciboConcedido)
	valores := []any{
		cabecera,
		datosCabecera,
		gobierno,
		datosGobierno,
		consumo,
		datosConsumo,
		efecto,
		datosEfecto,
		terminal,
		datosTerminal,
		crudo,
	}
	sensibles := []string{
		datosCabecera.TokenPropietarioSHA256,
		datosCabecera.AmbitoIdempotenciaHMAC,
		datosCabecera.HuellaSemanticaHMAC,
		datosCabecera.DecisionVECRef,
		datosCabecera.ReciboRef,
	}
	for _, valor := range valores {
		if _, err := json.Marshal(valor); !errors.Is(
			err,
			cobertura.ErrSerializacionOperacionDecisionCoberturaProhibida,
		) {
			t.Fatalf("%T serializable: %v", valor, err)
		}
		formato := fmt.Sprintf("%v %#v", valor, valor)
		var salida bytes.Buffer
		slog.New(slog.NewJSONHandler(&salida, nil)).Info(
			"prueba",
			"fragmento",
			valor,
		)
		for _, sensible := range sensibles {
			if sensible != "" &&
				(strings.Contains(formato, sensible) ||
					strings.Contains(salida.String(), sensible)) {
				t.Fatalf(
					"%T filtró material sensible %q: %q / %s",
					valor,
					sensible,
					formato,
					salida.String(),
				)
			}
		}
	}

	for _, valor := range []any{
		cobertura.CabeceraSesionTCBOperacionDecisionCobertura{},
		cobertura.GobiernoSesionTCBOperacionDecisionCobertura{},
		cobertura.DecisionVECSesionTCBOperacionDecisionCobertura{},
		cobertura.ConsumoC1SesionTCBOperacionDecisionCobertura{},
		cobertura.EfectoConcedidoSesionTCBOperacionDecisionCobertura{},
		cobertura.TerminalDenegadoSesionTCBOperacionDecisionCobertura{},
	} {
		tipo := reflect.TypeOf(valor)
		for indice := 0; indice < tipo.NumField(); indice++ {
			if tipo.Field(indice).IsExported() {
				t.Fatalf(
					"fragmento %s construible por campos: %s",
					tipo.Name(),
					tipo.Field(indice).Name,
				)
			}
		}
	}
}
