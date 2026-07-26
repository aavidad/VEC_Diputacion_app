package cobertura

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func solicitudRecuperacionOperacionDecisionCoberturaPrueba(
	t *testing.T,
	expedienteRef string,
) SolicitudRecuperacionResultadoOperacionDecisionCobertura {
	t.Helper()
	contexto, solicitudContexto :=
		contextoAutorizacionOperacionDecisionCoberturaPrueba(t)
	contextoRecuperacion, err :=
		ports.NuevoContextoRecuperacionResultadoCobertura(
			solicitudContexto,
			contexto,
			"organizacion_diputacion_granada",
			instanteOperacionDecisionCoberturaPrueba,
		)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err :=
		NuevaPreimagenAmbitoRecuperacionOperacionDecisionCobertura(
			"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
			expedienteRef,
			contextoRecuperacion,
			instanteOperacionDecisionCoberturaPrueba,
		)
	if err != nil {
		t.Fatal(err)
	}
	ambitos := sellosOperacionDecisionCoberturaPrueba(t).
		AmbitosIdempotenciaHMAC
	solicitud, err :=
		NuevaSolicitudRecuperacionResultadoOperacionDecisionCobertura(
			preimagen,
			ambitos,
		)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func TestPreimagenRecuperacionReutilizaCanonAmbitoC3Exacto(t *testing.T) {
	contexto, solicitudContexto :=
		contextoAutorizacionOperacionDecisionCoberturaPrueba(t)
	contextoRecuperacion, err :=
		ports.NuevoContextoRecuperacionResultadoCobertura(
			solicitudContexto,
			contexto,
			"organizacion_diputacion_granada",
			instanteOperacionDecisionCoberturaPrueba,
		)
	if err != nil {
		t.Fatal(err)
	}
	base := identidadOperacionDecisionCoberturaPrueba()
	identidad, err := NuevaIdentidadOperacionDecisionCobertura(
		base.claveIdempotencia,
		base.tipo,
		base.organizacionRef,
		base.expedienteRef,
		base.versionExpediente,
		contexto,
		solicitudContexto,
		instanteOperacionDecisionCoberturaPrueba,
		base.accion,
		base.viaElegida,
		base.identidadSemantica,
		base.motivo,
		base.predecesoraRef,
		base.predecesoraHuella,
	)
	if err != nil {
		t.Fatal(err)
	}
	completas, err := NuevasPreimagenesOperacionDecisionCobertura(identidad)
	if err != nil {
		t.Fatal(err)
	}
	esperada, _ := completas.BytesAmbito()
	recuperacion, err :=
		NuevaPreimagenAmbitoRecuperacionOperacionDecisionCobertura(
			base.claveIdempotencia,
			base.expedienteRef,
			contextoRecuperacion,
			instanteOperacionDecisionCoberturaPrueba,
		)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, _ := recuperacion.Bytes()
	if string(obtenida) != string(esperada) {
		t.Fatalf("canon de ámbito divergente:\n%x\n%x", obtenida, esperada)
	}
}

func TestResultadoHistoricoCotejaAmbitoCoordenadasYEvidencia(t *testing.T) {
	solicitud := solicitudRecuperacionOperacionDecisionCoberturaPrueba(
		t,
		"expediente_temporal_2026_5487",
	)
	_, reserva := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidadOperacionDecisionCoberturaPrueba(),
	)
	datosReserva := datosReservaTerminalOperacionDecisionCoberturaPrueba(
		t,
		reserva,
	)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, reserva)
	base := DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura{
		Reserva:     datosReserva,
		Recibo:      recibo,
		ObservadaEn: recibo.ConfirmadaEn,
	}
	resultado, err :=
		nuevoResultadoHistoricoConfirmadoOperacionDecisionCobertura(
			solicitud,
			base,
		)
	if err != nil {
		t.Fatal(err)
	}
	if recuperado, confirmado := resultado.ReciboConfirmadoPara(solicitud); !confirmado || recuperado.ReciboRef != recibo.ReciboRef {
		t.Fatal("resultado histórico válido no recuperado")
	}

	casos := map[string]func(*DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura){
		"organizacion": func(e *DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura) {
			e.Reserva.OrganizacionRef = "organizacion_cruzada_historica_01"
		},
		"expediente": func(e *DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura) {
			e.Reserva.ExpedienteRef = "expediente_cruzado_historico_01"
		},
		"ambito": func(e *DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura) {
			e.Reserva.AmbitoIdempotenciaHMAC = selloOperacionDecisionCoberturaPrueba(
				dominioAmbitoOperacionDecisionCobertura,
				2,
				"f",
			)
			e.Recibo.AmbitoIdempotenciaHMAC =
				e.Reserva.AmbitoIdempotenciaHMAC
		},
		"generacion_semantica": func(e *DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura) {
			e.Reserva.HuellaSemanticaHMAC = selloOperacionDecisionCoberturaPrueba(
				dominioSemanticaOperacionDecisionCobertura,
				1,
				"c",
			)
			e.Recibo.HuellaSemanticaHMAC =
				e.Reserva.HuellaSemanticaHMAC
		},
		"referencia": func(e *DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura) {
			e.Recibo.ReservaRef = "reserva_historica_cruzada_01"
		},
		"observacion": func(e *DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura) {
			e.ObservadaEn = e.Recibo.ConfirmadaEn.Add(-time.Microsecond)
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			evidencia := base
			alterar(&evidencia)
			if _, err :=
				nuevoResultadoHistoricoConfirmadoOperacionDecisionCobertura(
					solicitud,
					evidencia,
				); err == nil {
				t.Fatal("evidencia contradictoria aceptada")
			}
		})
	}
}

func TestResultadoHistoricoDenegadoConservaReferenciasPreasignadas(t *testing.T) {
	solicitud := solicitudRecuperacionOperacionDecisionCoberturaPrueba(
		t,
		"expediente_temporal_2026_5487",
	)
	_, reserva := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidadOperacionDecisionCoberturaPrueba(),
	)
	datosReserva := datosReservaTerminalOperacionDecisionCoberturaPrueba(
		t,
		reserva,
	)
	recibo := reciboDenegadoVECOperacionDecisionCoberturaPrueba(t, reserva)
	resultado, err :=
		nuevoResultadoHistoricoConfirmadoOperacionDecisionCobertura(
			solicitud,
			DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura{
				Reserva:     datosReserva,
				Recibo:      recibo,
				ObservadaEn: recibo.ConfirmadaEn,
			},
		)
	if err != nil {
		t.Fatal(err)
	}
	if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); !confirmado {
		t.Fatal("denegación histórica válida rechazada")
	}
}

func TestNoObservableQuedaLigadoASolicitudExacta(t *testing.T) {
	original := solicitudRecuperacionOperacionDecisionCoberturaPrueba(
		t,
		"expediente_temporal_2026_5487",
	)
	cruzada := solicitudRecuperacionOperacionDecisionCoberturaPrueba(
		t,
		"expediente_temporal_2026_otra",
	)
	resultado, err :=
		nuevoResultadoHistoricoNoObservableOperacionDecisionCobertura(
			original,
			instanteOperacionDecisionCoberturaPrueba,
		)
	if err != nil {
		t.Fatal(err)
	}
	if !resultado.NoObservablePara(original) ||
		resultado.NoObservablePara(cruzada) {
		t.Fatal("no observable no quedó ligado a coordenadas exactas")
	}
}
