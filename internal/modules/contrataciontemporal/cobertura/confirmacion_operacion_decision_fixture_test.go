package cobertura_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	dominioct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type escenarioConfirmacionOrdenC3 struct {
	base               time.Time
	limite             time.Time
	ordenConcedida     cobertura.OrdenOperacionDecisionCobertura
	ordenDenegada      cobertura.OrdenOperacionDecisionCobertura
	reciboConcedido    cobertura.ReciboOperacionDecisionCobertura
	reciboDenegado     cobertura.ReciboOperacionDecisionCobertura
	resultadoConcedido cobertura.ResultadoConfirmacionOperacionDecisionCobertura
	resultadoDenegado  cobertura.ResultadoConfirmacionOperacionDecisionCobertura
}

func nuevoEscenarioConfirmacionOrdenC3(
	t *testing.T,
) escenarioConfirmacionOrdenC3 {
	t.Helper()
	escenarioOrden := nuevoEscenarioOrdenC3(t)
	reloj := relojGobiernoOrdenC3{
		ahora: escenarioOrden.base.Add(4 * time.Millisecond),
	}
	ordenConcedida, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
		context.Background(),
		reloj,
		escenarioOrden.preparacion,
		escenarioOrden.candidataConcedida,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenDenegada, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
		context.Background(),
		reloj,
		escenarioOrden.preparacion,
		escenarioOrden.candidataDenegada,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision := decisionEsperadaConfirmacionOrdenC3(
		t,
		escenarioOrden.base,
	)
	reciboConcedido := reciboConfirmacionOrdenC3(
		t,
		escenarioOrden.candidataConcedida,
		escenarioOrden.base.Add(4500*time.Microsecond),
		&cobertura.ResultadoAplicadoOperacionDecisionCobertura{
			DecisionCoberturaRef:    decision.Referencia,
			DecisionCoberturaHuella: decision.HuellaSHA256,
			VersionResultante:       decision.VersionExpediente,
			EventoRef:               "evento_decision_cobertura_orden_c3_01",
			ActuacionRef:            "actuacion_decision_cobertura_orden_c3_01",
		},
	)
	reciboDenegado := reciboConfirmacionOrdenC3(
		t,
		escenarioOrden.candidataDenegada,
		escenarioOrden.base.Add(4500*time.Microsecond),
		nil,
	)
	resultadoConcedido, err :=
		cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
			ordenConcedida,
			reciboConcedido,
		)
	if err != nil {
		t.Fatalf("resultado concedido: %v", err)
	}
	resultadoDenegado, err :=
		cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
			ordenDenegada,
			reciboDenegado,
		)
	if err != nil {
		t.Fatalf("resultado denegado: %v", err)
	}
	return escenarioConfirmacionOrdenC3{
		base: escenarioOrden.base, limite: escenarioOrden.limite,
		ordenConcedida: ordenConcedida, ordenDenegada: ordenDenegada,
		reciboConcedido: reciboConcedido, reciboDenegado: reciboDenegado,
		resultadoConcedido: resultadoConcedido,
		resultadoDenegado:  resultadoDenegado,
	}
}

func decisionEsperadaConfirmacionOrdenC3(
	t *testing.T,
	base time.Time,
) dominioct.PublicacionDecisionCoberturaGobernada {
	t.Helper()
	expediente := expedienteConAnalisisOrdenC3(t, base)
	solicitudO3, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		lectorAnalisisOrdenC3{expediente: expediente},
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	agregado, analisisRef, analisisHuella, err := instantanea.DesplegarPara(
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, politica := catalogoYPoliticaOrdenC3(t, base)
	preparacionC1 := prepararC1OrdenC3(
		t,
		agregado,
		analisisRef,
		analisisHuella,
		catalogo,
		politica,
		base.Add(time.Millisecond),
	)
	datosPropuesta, err := preparacionC1.DatosCrearPropuestaEn(
		base.Add(1500 * time.Microsecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := dominioct.CrearPropuestaDecisionCobertura(datosPropuesta)
	if err != nil {
		t.Fatal(err)
	}
	contexto, _ := contextoActorOrdenC3(t, base)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	siguiente, err := agregado.RegistrarDecisionCoberturaGobernada(
		agregado.Version,
		dominioct.DatosAdoptarDecisionCobertura{
			PerfilRef:  vinculo.PerfilActivoRef,
			ViaElegida: "bolsa_vigente",
		},
		propuesta,
		dominioct.DatosActuacion{
			AccionClave:   dominioct.AccionDecidirCoberturaGobernada,
			ActorRef:      vinculo.PrincipalID,
			UnidadRef:     "unidad_rrhh_cobertura_orden_c3",
			ReciboRef:     "recibo_decision_cobertura_orden_c3_01",
			RealizadaEn:   base.Add(4 * time.Millisecond),
			FaseDestino:   "asignacion_unidad",
			EstadoDestino: dominioct.EstadoEnCurso,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return siguiente.DecisionesCobertura[len(siguiente.DecisionesCobertura)-1]
}

func reciboConfirmacionOrdenC3(
	t *testing.T,
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	confirmadaEn time.Time,
	aplicada *cobertura.ResultadoAplicadoOperacionDecisionCobertura,
) cobertura.ReciboOperacionDecisionCobertura {
	t.Helper()
	resumen, err := candidata.Resumen()
	if err != nil {
		t.Fatal(err)
	}
	datos, err := resumen.Datos()
	if err != nil {
		t.Fatal(err)
	}
	recibo := cobertura.ReciboOperacionDecisionCobertura{
		ReciboRef:               "recibo_decision_cobertura_orden_c3_01",
		ReservaRef:              "reserva_decision_cobertura_orden_c3_01",
		AuditoriaRef:            "auditoria_decision_cobertura_orden_c3_01",
		CorrelacionVECRef:       "correlacion_11111111111111111111111111111111",
		DecisionVECRef:          datos.DecisionRef,
		DecisionVECHuellaSHA256: datos.DecisionHuellaSHA256,
		CodigoProbatorioVEC:     datos.CodigoProbatorio,
		ConcedidaVEC:            datos.Concedida,
		RevisionCercado:         1,
		AmbitoIdempotenciaHMAC: "hmac-sha256:vec.contratacion-temporal." +
			"cobertura-decision.ambito/v1:" + strings.Repeat("a", 64),
		HuellaSemanticaHMAC: "hmac-sha256:vec.contratacion-temporal." +
			"cobertura-decision.semantica/v1:" + strings.Repeat("b", 64),
		ConfirmadaEn: confirmadaEn,
	}
	if aplicada != nil {
		copia := *aplicada
		recibo.Aplicada = &copia
	} else {
		recibo.DenegadaVEC =
			&cobertura.ResultadoDenegadoVECOperacionDecisionCobertura{}
	}
	return recibo
}

func clonarReciboConfirmacionOrdenC3(
	origen cobertura.ReciboOperacionDecisionCobertura,
) cobertura.ReciboOperacionDecisionCobertura {
	clon := origen
	if origen.Aplicada != nil {
		aplicada := *origen.Aplicada
		clon.Aplicada = &aplicada
	}
	if origen.DenegadaVEC != nil {
		denegada := *origen.DenegadaVEC
		clon.DenegadaVEC = &denegada
	}
	return clon
}

func nuevaTransaccionTCBConfirmacionOrdenC3(
	t *testing.T,
	recibo cobertura.ReciboOperacionDecisionCobertura,
	configurar func(*ejecutorSesionTCBOperacionDecisionPrueba),
) (
	cobertura.TransaccionOperacionDecisionCobertura,
	*ejecutorSesionTCBOperacionDecisionPrueba,
) {
	t.Helper()
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(recibo),
	}
	if configurar != nil {
		configurar(ejecutor)
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	return transaccion, ejecutor
}

type reconciliadorConfirmacionOrdenC3 struct {
	resultado cobertura.ResultadoReconciliacionOperacionDecisionCobertura
	err       error
	despues   func()
	llamadas  atomic.Int32
}

func (r *reconciliadorConfirmacionOrdenC3) ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
	context.Context,
	cobertura.SolicitudReconciliacionOperacionDecisionCobertura,
) (cobertura.ResultadoReconciliacionOperacionDecisionCobertura, error) {
	r.llamadas.Add(1)
	if r.despues != nil {
		r.despues()
	}
	return r.resultado, r.err
}

var errTransporteConfirmacionOrdenC3 = errors.New(
	"error privado de transporte de prueba",
)
