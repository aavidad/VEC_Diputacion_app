package application

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestServicioAsignacionRechazaConsultorTypedNil(t *testing.T) {
	var consultorNulo *consultorAsignacionDoble
	servicio, err := NuevoServicioAsignacion(
		&resolutorContextoDoble{},
		&selladorAmbitoAsignacionDoble{},
		&derivadorHuellaAsignacionDoble{},
		consultorNulo,
		&preparadorAsignacionDoble{},
		&resolutorDestinoAsignacionDoble{},
		&resolutorPoliticaAsignacionDoble{},
		&generadorReferenciasDoble{},
		&autorizadorV3Doble{t: t},
		&relojMutable{},
		&transaccionAsignacionDoble{},
	)
	if servicio != nil || !errors.Is(err, ErrServicioAsignacionInvalido) {
		t.Fatalf("consultor nil tipado aceptado: %#v / %v", servicio, err)
	}
}

func TestReplayAsignacionRenuevaAutorizacionConAliasActivoSinMutar(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	estado := configurarReplayAsignacionConfirmado(t, dependencias)

	recibo, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("replay autorizado: %v", err)
	}
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		dependencias.preparar,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosV3, err := dependencias.autorizador.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if recibo != dependencias.transaccion.recibo || estado.EsCero() ||
		dependencias.consultas.llamadas != 1 ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.destinos.llamadas != 1 ||
		dependencias.politicas.llamadas != 1 ||
		dependencias.autorizador.llamadas != 1 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("el replay repitió consulta o mutación: %#v", dependencias)
	}
	if datosV3.Accion != ports.AccionRegistrarAsignacion ||
		datosV3.Recurso.Referencia != escenario.solicitud.ExpedienteRef ||
		datosV3.Recurso.Ambitos["expediente_ref"] !=
			escenario.solicitud.ExpedienteRef ||
		datosV3.Recurso.Atributos[ports.AtributoVersionAsignacion] !=
			strconv.FormatUint(escenario.solicitud.VersionEsperada, 10) ||
		datosV3.Recurso.Atributos[ports.AtributoUnidadDestino] !=
			escenario.solicitud.UnidadRef ||
		datosV3.Recurso.Atributos[ports.AtributoResponsableDestino] !=
			escenario.solicitud.ResponsableRef ||
		datosV3.Recurso.Atributos[ports.AtributoAmbitoIdempotenciaActivo] !=
			consulta.AmbitoIdempotenciaHMACActivo ||
		datosV3.Recurso.Atributos[ports.AtributoHuellaPeticionAsignacion] !=
			consulta.HuellaPeticionHMACActiva ||
		datosV3.Finalidad != "gestionar_contratacion_temporal" {
		t.Fatalf("autorización de replay incompleta: %#v", datosV3)
	}
}

func TestReplayAsignacionVistaRepetidaNoExtraeYReconciliacionEsUnica(
	t *testing.T,
) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	estado := configurarReplayAsignacionConfirmado(t, dependencias)
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		dependencias.preparar,
	)
	if err != nil {
		t.Fatal(err)
	}

	primera, err := estado.VistaPara(consulta)
	if err != nil {
		t.Fatalf("primera vista mínima: %v", err)
	}
	primera.ExpedienteRef = "expediente:alterado-solo-en-copia"
	segunda, err := estado.VistaPara(consulta)
	if err != nil || segunda.ExpedienteRef != consulta.ExpedienteRef {
		t.Fatalf("vista repetida mutable o inválida: %#v / %v", segunda, err)
	}
	codificada, err := json.Marshal(segunda)
	if err != nil {
		t.Fatalf("serializar vista mínima: %v", err)
	}
	for _, datoTerminal := range []string{
		dependencias.transaccion.recibo.ReciboRef,
		dependencias.transaccion.recibo.NotificacionRef,
		dependencias.transaccion.recibo.BandejaRef,
		dependencias.transaccion.recibo.AuditoriaRef,
		dependencias.transaccion.recibo.EventoRef,
		dependencias.transaccion.recibo.ConcesionV3DecisionRef,
	} {
		if strings.Contains(string(codificada), datoTerminal) {
			t.Fatalf("vista mínima serializó dato terminal %q", datoTerminal)
		}
	}

	recibo, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if err != nil || recibo != dependencias.transaccion.recibo ||
		dependencias.destinos.llamadas != 1 ||
		dependencias.politicas.llamadas != 1 ||
		dependencias.autorizador.llamadas != 1 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("reconciliación sin autoridades frescas: %#v / %v", recibo, err)
	}
	posterior, err := estado.VistaPara(consulta)
	if err != nil || posterior.ExpedienteRef != consulta.ExpedienteRef {
		t.Fatalf("vista posterior expuso o perdió el agregado previo: %#v / %v", posterior, err)
	}
	segundo, err := estado.Reconciliar(ports.EvidenciaReconciliacionAsignacion{})
	if !errors.Is(err, ports.ErrResultadoAsignacionNoConfiable) ||
		segundo != (ports.ReciboAsignacion{}) {
		t.Fatalf("segunda extracción del recibo: %#v / %v", segundo, err)
	}
}

func TestReplayReasignacionRenuevaLaOperacionExactaSinMutar(t *testing.T) {
	escenario := nuevoEscenarioReasignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	configurarReplayAsignacionConfirmado(t, dependencias)

	recibo, err := servicio.Reasignar(
		context.Background(),
		escenario.reasignacion,
	)
	if err != nil {
		t.Fatalf("replay de reasignación: %v", err)
	}
	datosV3, err := dependencias.autorizador.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if recibo.Operacion != ports.OperacionRegistrarReasignacion ||
		datosV3.Accion != ports.AccionRegistrarReasignacion ||
		dependencias.consultas.llamadas != 1 ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("replay de reasignación repitió el efecto: %#v %#v", recibo, dependencias)
	}
}

func TestReplayAsignacionDeniegaAutorizacionNuevaCaducada(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	configurarReplayAsignacionConfirmado(t, dependencias)
	servicio.reloj.(*relojMutable).fijar(escenario.instante.Add(10 * time.Minute))

	_, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrAsignacionDenegada) ||
		dependencias.consultas.llamadas != 1 ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("autorización caducada no denegó sin mutar: %v %#v", err, dependencias)
	}
}

func TestReplayAsignacionRechazaDestinoOPoliticaAlterados(t *testing.T) {
	for _, dimension := range []string{"destino", "politica"} {
		t.Run(dimension, func(t *testing.T) {
			escenario := nuevoEscenarioAsignacion(t)
			servicio, dependencias := construirServicioAsignacion(t, escenario)
			configurarReplayAsignacionConfirmado(t, dependencias)
			if dimension == "destino" {
				dependencias.destinos.evidenciaAlterna = true
			} else {
				dependencias.politicas.definicionAlterna = true
			}

			_, err := servicio.Asignar(context.Background(), escenario.solicitud)
			if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
				dependencias.consultas.llamadas != 1 ||
				dependencias.preparaciones.llamadas != 0 ||
				dependencias.transaccion.llamadas != 0 {
				t.Fatalf("%s alterado permitió replay: %v %#v", dimension, err, dependencias)
			}
		})
	}
}

func TestReplayAsignacionEvidenciaInvalidaConsumeSinRevelarRecibo(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	configurarReplayAsignacionConfirmado(t, dependencias)
	dependencias.destinos.evidenciaAlterna = true

	primero, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
		primero != (ports.ReciboAsignacion{}) {
		t.Fatalf("evidencia inválida reveló recibo: %#v / %v", primero, err)
	}
	dependencias.destinos.evidenciaAlterna = false
	segundo, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
		segundo != (ports.ReciboAsignacion{}) ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("estado consumido entregó recibo después: %#v / %v", segundo, err)
	}
}

func TestReplayAsignacionRotacionPosteriorNoRevelaRecibo(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	configurarReplayAsignacionConfirmado(t, dependencias)
	ambitoV2, huellaV2, err := ports.ParActivoColeccionesHMAC(
		dependencias.preparar.AmbitosHMAC,
		ports.DominioAmbitoIdempotenciaAsignacion,
		dependencias.preparar.HuellasPeticionHMAC,
		ports.DominioHuellaPeticionAsignacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	ambitoV3 := selloHMACRegistroPrueba(
		ports.DominioAmbitoIdempotenciaAsignacion+"/v3",
		"3",
	)
	huellaV3 := selloHMACRegistroPrueba(
		ports.DominioHuellaPeticionAsignacion+"/v3",
		"4",
	)
	ambitos, err := ports.NuevaColeccionSellosHMAC(ambitoV3, []string{ambitoV2})
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(huellaV3, []string{huellaV2})
	if err != nil {
		t.Fatal(err)
	}
	servicio.ambitos.(*selladorAmbitoAsignacionDoble).coleccion = ambitos
	servicio.huellas.(*derivadorHuellaAsignacionDoble).coleccion = huellas

	recibo, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
		recibo != (ports.ReciboAsignacion{}) ||
		dependencias.destinos.llamadas != 0 ||
		dependencias.autorizador.llamadas != 0 ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("rotación posterior reveló recibo: %#v / %v", recibo, err)
	}
}

func TestReplayAsignacionRechazaCandidatoSinAliasHMACActivo(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	prepararAnterior := dependencias.preparar
	prepararAnterior.AmbitosHMAC = coleccionAsignacionPrueba(
		t,
		ports.DominioAmbitoIdempotenciaAsignacion+"/v1",
		"3",
	)
	prepararAnterior.HuellasPeticionHMAC = coleccionAsignacionPrueba(
		t,
		ports.DominioHuellaPeticionAsignacion+"/v1",
		"4",
	)
	consultaAnterior, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		prepararAnterior,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmada := dependencias.preparaciones.preparacion
	recibo := dependencias.transaccion.recibo
	confirmada.Estado = ports.PreparacionAsignacionConfirmada
	confirmada.ReciboConfirmado = &recibo
	estado, err := ports.NuevoEstadoCandidatoAsignacionIdempotente(
		ports.DatosEstadoCandidatoAsignacionIdempotente{
			Consulta:                     consultaAnterior,
			Preparacion:                  confirmada,
			DestinoEvidenciaRef:          dependencias.destinos.destino.EvidenciaRef,
			DestinoEvidenciaHuellaSHA256: dependencias.destinos.destino.EvidenciaHuellaSHA256,
			PoliticaRef:                  "politica:asignacion-sintetica-001",
			PoliticaVersion:              2,
			PoliticaHuellaSHA256:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Finalidad:                    "gestionar_contratacion_temporal",
		},
	)
	if !errors.Is(err, ports.ErrResultadoAsignacionNoConfiable) || !estado.EsCero() {
		t.Fatalf("candidato sin par activo construible: %#v / %v", estado, err)
	}
	dependencias.consultas.estado = estado
	dependencias.consultas.encontrado = true

	_, err = servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
		dependencias.consultas.llamadas != 1 ||
		dependencias.destinos.llamadas != 0 ||
		dependencias.autorizador.llamadas != 0 ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("alias no activo permitió replay: %v %#v", err, dependencias)
	}
}

func TestReplayAsignacionRechazaPreparacionRetenidaBajoConsultaActiva(
	t *testing.T,
) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	ambitoV2 := selloHMACRegistroPrueba(
		ports.DominioAmbitoIdempotenciaAsignacion+"/v2",
		"a",
	)
	huellaV2 := selloHMACRegistroPrueba(
		ports.DominioHuellaPeticionAsignacion+"/v2",
		"b",
	)
	ambitoV1 := selloHMACRegistroPrueba(
		ports.DominioAmbitoIdempotenciaAsignacion+"/v1",
		"c",
	)
	huellaV1 := selloHMACRegistroPrueba(
		ports.DominioHuellaPeticionAsignacion+"/v1",
		"d",
	)
	ambitos, err := ports.NuevaColeccionSellosHMAC(ambitoV2, []string{ambitoV1})
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(huellaV2, []string{huellaV1})
	if err != nil {
		t.Fatal(err)
	}
	servicio.ambitos.(*selladorAmbitoAsignacionDoble).coleccion = ambitos
	servicio.huellas.(*derivadorHuellaAsignacionDoble).coleccion = huellas
	dependencias.preparar.AmbitosHMAC = ambitos
	dependencias.preparar.HuellasPeticionHMAC = huellas

	preparacion := dependencias.preparaciones.preparacion
	recibo := dependencias.transaccion.recibo
	preparacion.AmbitoIdempotenciaHMAC = ambitoV1
	preparacion.HuellaPeticionHMAC = huellaV1
	recibo.AmbitoIdempotenciaHMAC = ambitoV1
	recibo.HuellaPeticionHMAC = huellaV1
	preparacion.Estado = ports.PreparacionAsignacionConfirmada
	preparacion.ReciboConfirmado = &recibo
	if err := preparacion.ValidarPara(dependencias.preparar); err != nil {
		t.Fatalf("la validación generacional dejó de admitir el par retenido: %v", err)
	}
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		dependencias.preparar,
	)
	if err != nil || consulta.AmbitoIdempotenciaHMACActivo != ambitoV2 ||
		consulta.HuellaPeticionHMACActiva != huellaV2 {
		t.Fatalf("consulta activa inválida: %#v / %v", consulta, err)
	}
	estado, err := ports.NuevoEstadoCandidatoAsignacionIdempotente(
		ports.DatosEstadoCandidatoAsignacionIdempotente{
			Consulta:                     consulta,
			Preparacion:                  preparacion,
			DestinoEvidenciaRef:          dependencias.destinos.destino.EvidenciaRef,
			DestinoEvidenciaHuellaSHA256: dependencias.destinos.destino.EvidenciaHuellaSHA256,
			PoliticaRef:                  "politica:asignacion-sintetica-001",
			PoliticaVersion:              2,
			PoliticaHuellaSHA256:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Finalidad:                    "gestionar_contratacion_temporal",
		},
	)
	if !errors.Is(err, ports.ErrResultadoAsignacionNoConfiable) || !estado.EsCero() {
		t.Fatalf("el terminal retenido quedó ligado a la consulta activa: %#v / %v", estado, err)
	}
	dependencias.consultas.estado = estado
	dependencias.consultas.encontrado = true

	_, err = servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
		dependencias.consultas.llamadas != 1 ||
		dependencias.destinos.llamadas != 0 ||
		dependencias.autorizador.llamadas != 0 ||
		dependencias.preparaciones.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("el terminal retenido alcanzó mutadores: %v %#v", err, dependencias)
	}
}

func TestConsultaReplayAsignacionFallaCerradaSinSegundaTransaccion(t *testing.T) {
	for _, caso := range []string{
		"error", "encontrado_sin_estado", "estado_sin_encontrar", "cancelacion",
	} {
		t.Run(caso, func(t *testing.T) {
			escenario := nuevoEscenarioAsignacion(t)
			servicio, dependencias := construirServicioAsignacion(t, escenario)
			ctx := context.Background()
			var cancelar context.CancelFunc
			switch caso {
			case "error":
				dependencias.consultas.err = errors.New("fallo privado")
			case "encontrado_sin_estado":
				dependencias.consultas.encontrado = true
			case "estado_sin_encontrar":
				dependencias.consultas.estado = configurarReplayAsignacionConfirmado(
					t,
					dependencias,
				)
				dependencias.consultas.encontrado = false
			case "cancelacion":
				configurarReplayAsignacionConfirmado(t, dependencias)
				ctx, cancelar = context.WithCancel(context.Background())
				dependencias.consultas.antes = cancelar
			}

			_, err := servicio.Asignar(ctx, escenario.solicitud)
			if err == nil || dependencias.consultas.llamadas != 1 ||
				dependencias.preparaciones.llamadas != 0 ||
				dependencias.transaccion.llamadas != 0 {
				t.Fatalf("consulta %s no falló cerrada: %v %#v", caso, err, dependencias)
			}
			if caso == "cancelacion" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelación ocultada: %v", err)
			}
		})
	}
}

func TestReplayAparecidoTrasConsultaNoAbreTransaccion(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	confirmada := dependencias.preparaciones.preparacion
	recibo := dependencias.transaccion.recibo
	confirmada.Estado = ports.PreparacionAsignacionConfirmada
	confirmada.ReciboConfirmado = &recibo
	dependencias.preparaciones.preparacion = confirmada

	_, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoAsignacionNoConfiable) ||
		dependencias.consultas.llamadas != 1 ||
		dependencias.preparaciones.llamadas != 1 ||
		dependencias.autorizador.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("terminal concurrente abrió segunda transacción: %v %#v", err, dependencias)
	}
}

func configurarReplayAsignacionConfirmado(
	t *testing.T,
	dependencias *dependenciasAsignacion,
) ports.EstadoCandidatoAsignacionIdempotente {
	t.Helper()
	confirmada := dependencias.preparaciones.preparacion
	recibo := dependencias.transaccion.recibo
	confirmada.Estado = ports.PreparacionAsignacionConfirmada
	confirmada.ReciboConfirmado = &recibo
	dependencias.preparaciones.preparacion = confirmada
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		dependencias.preparar,
	)
	if err != nil {
		t.Fatal(err)
	}
	estado := nuevoEstadoReplayAsignacion(t, dependencias, consulta)
	dependencias.consultas.estado = estado
	dependencias.consultas.encontrado = true
	return estado
}

func nuevoEstadoReplayAsignacion(
	t *testing.T,
	dependencias *dependenciasAsignacion,
	consulta ports.SolicitudConsultarAsignacionIdempotente,
) ports.EstadoCandidatoAsignacionIdempotente {
	t.Helper()
	confirmada := dependencias.preparaciones.preparacion
	if confirmada.Estado != ports.PreparacionAsignacionConfirmada {
		recibo := dependencias.transaccion.recibo
		confirmada.Estado = ports.PreparacionAsignacionConfirmada
		confirmada.ReciboConfirmado = &recibo
	}
	estado, err := ports.NuevoEstadoCandidatoAsignacionIdempotente(
		ports.DatosEstadoCandidatoAsignacionIdempotente{
			Consulta:                     consulta,
			Preparacion:                  confirmada,
			DestinoEvidenciaRef:          dependencias.destinos.destino.EvidenciaRef,
			DestinoEvidenciaHuellaSHA256: dependencias.destinos.destino.EvidenciaHuellaSHA256,
			PoliticaRef:                  "politica:asignacion-sintetica-001",
			PoliticaVersion:              2,
			PoliticaHuellaSHA256:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Finalidad:                    "gestionar_contratacion_temporal",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return estado
}
