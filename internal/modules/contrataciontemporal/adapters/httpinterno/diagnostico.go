package httpinterno

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// registrarFalloContratacion solo registra metadatos cerrados, nunca el texto de la causa.
func registrarFalloContratacion(r *http.Request, estado int, codigo, correlacion string, causa error) {
	if estado < http.StatusInternalServerError &&
		(estado != http.StatusConflict || codigo != "conflicto_estado") {
		return
	}
	ruta, operacion := "ruta_no_reconocida", "contratacion"
	if r != nil && r.URL != nil {
		switch r.URL.Path {
		case RutaAnotacionesAdministrativas:
			ruta, operacion = RutaAnotacionesAdministrativas, "anotaciones_administrativas"
		case RutaAsignaciones:
			ruta, operacion = RutaAsignaciones, "asignaciones"
		case RutaCerrarAdministrativamente:
			ruta, operacion = RutaCerrarAdministrativamente, "cerrar_administrativamente"
		case RutaCerrarAdministrativamenteSinCese:
			ruta, operacion = RutaCerrarAdministrativamenteSinCese, "cerrar_administrativamente_sin_cese"
		case RutaConsultaSeguimientoV2:
			ruta, operacion = RutaConsultaSeguimientoV2, "consulta_seguimiento_v2"
		case RutaContinuacionLlamamiento:
			ruta, operacion = RutaContinuacionLlamamiento, "continuacion_llamamiento"
		case RutaDecisionCobertura:
			ruta, operacion = RutaDecisionCobertura, "decision_cobertura"
		case RutaDocumentacionFormalizacion:
			ruta, operacion = RutaDocumentacionFormalizacion, "documentacion_formalizacion"
		case RutaFirmaDocumento:
			ruta, operacion = RutaFirmaDocumento, "firma_documento"
		case RutaConsultaFirmaDocumento:
			ruta, operacion = RutaConsultaFirmaDocumento, "consulta_firma_documento"
		case RutaFichaGINPIXV2:
			ruta, operacion = RutaFichaGINPIXV2, "ficha_ginpixv2"
		case RutaIncorporacionEjercicioV2:
			ruta, operacion = RutaIncorporacionEjercicioV2, "incorporacion_ejercicio_v2"
		case RutaPreparacionCierreSinCese:
			ruta, operacion = RutaPreparacionCierreSinCese, "preparacion_cierre_sin_cese"
		case RutaPreparacionesInformeJuridico:
			ruta, operacion = RutaPreparacionesInformeJuridico, "preparaciones_informe_juridico"
		case RutaPropuestaCobertura:
			ruta, operacion = RutaPropuestaCobertura, "propuesta_cobertura"
		case RutaPropuestaFormalizacion:
			ruta, operacion = RutaPropuestaFormalizacion, "propuesta_formalizacion"
		case RutaReabrirExcepcionalmente:
			ruta, operacion = RutaReabrirExcepcionalmente, "reabrir_excepcionalmente"
		case RutaReasignaciones:
			ruta, operacion = RutaReasignaciones, "reasignaciones"
		case RutaRectificacionAnalisisRRHH:
			ruta, operacion = RutaRectificacionAnalisisRRHH, "rectificacion_analisis_rrhh"
		case RutaRectificacionCobertura:
			ruta, operacion = RutaRectificacionCobertura, "rectificacion_cobertura"
		case RutaRecuperacionAnotacionesAdministrativas:
			ruta, operacion = RutaRecuperacionAnotacionesAdministrativas, "recuperacion_anotaciones_administrativas"
		case RutaRegistroAnalisisRRHH:
			ruta, operacion = RutaRegistroAnalisisRRHH, "registro_analisis_rrhh"
		case RutaRegistroComunicacionLlamamiento:
			ruta, operacion = RutaRegistroComunicacionLlamamiento, "registro_comunicacion_llamamiento"
		case RutaRegistroRespuestaRecibida:
			ruta, operacion = RutaRegistroRespuestaRecibida, "registro_respuesta_recibida"
		case RutaEventoPlazoLlamamiento:
			ruta, operacion = RutaEventoPlazoLlamamiento, "evento_plazo_llamamiento"
		case RutaResolucionComunicacionLlamamiento:
			ruta, operacion = RutaResolucionComunicacionLlamamiento, "resolucion_comunicacion_llamamiento"
		case RutaResolucionFormalizacion:
			ruta, operacion = RutaResolucionFormalizacion, "resolucion_formalizacion"
		case RutaResultadoCobertura:
			ruta, operacion = RutaResultadoCobertura, "resultado_cobertura"
		case RutaResultadosFiscalizacion:
			ruta, operacion = RutaResultadosFiscalizacion, "resultados_fiscalizacion"
		case RutaSeleccionLlamamiento:
			ruta, operacion = RutaSeleccionLlamamiento, "seleccion_llamamiento"
		case RutaSubsanacionReparos:
			ruta, operacion = RutaSubsanacionReparos, "subsanacion_reparos"
		case RutaAltaSolicitudes:
			ruta, operacion = RutaAltaSolicitudes, "alta_solicitud"
		case RutaConsultaCuadroRRHH:
			ruta, operacion = RutaConsultaCuadroRRHH, "consulta_cuadro_rrhh"
		case RutaConsultaDetalleRRHH:
			ruta, operacion = RutaConsultaDetalleRRHH, "consulta_detalle_rrhh"
		}
	}
	etapa, sqlstate := "desconocida", ""
	var falloInterno *diagnostico.FalloConsultaRRHH
	if errors.As(causa, &falloInterno) && falloInterno != nil {
		etapa = falloInterno.EtapaSegura()
		if codigoSQLSeguro(falloInterno.CodigoSQL) {
			sqlstate = falloInterno.CodigoSQL
		}
	}
	// La lectura de la instantánea original puede llegar como causa SQL sin
	// FalloConsultaRRHH. Se clasifica solo en el alta y bajo su centinela.
	if ruta == RutaAltaSolicitudes && etapa == "desconocida" &&
		errors.Is(causa, ports.ErrFuenteNecesidadesAltaNoDisponible) {
		var estadoSQL interface{ SQLState() string }
		if errors.As(causa, &estadoSQL) {
			codigo := estadoSQL.SQLState()
			if codigoSQLSeguro(codigo) {
				etapa, sqlstate = string(diagnostico.EtapaSQL), codigo
			}
		}
	}
	if etapaCobertura, ok := application.EtapaDiagnosticoDePresentacionPropuestaCobertura(causa); ok {
		etapa = string(etapaCobertura)
	}
	if etapa == "desconocida" && errors.Is(causa, ports.ErrComposicionIncorporacionAplicacion) {
		etapa = "incorporacion_ejercicio.composicion"
	}
	atributos := []any{"operacion", operacion, "ruta", ruta,
		"estado_http", estado, "codigo", codigo,
		"etapa", etapa, "sqlstate", sqlstate,
		"centinela", centinelaSeguroContratacion(causa), "correlacion_ref", correlacion}
	if ruta == RutaSeleccionLlamamiento &&
		errors.Is(causa, application.ErrEjecucionSeleccionLlamamientoIndeterminada) {
		atributos = append(atributos, "causa_interna", causaInternaSeleccionIndeterminada(causa))
	}
	slog.Error("operación de Contratación fallida", atributos...)
}

// El resultado público indeterminado es un envoltorio: el diagnóstico busca
// la causa CT más profunda. Una causa desconocida aporta solo su tipo Go.
func causaInternaSeleccionIndeterminada(causa error) string {
	conocidos := []struct {
		error
		codigo string
	}{
		{ports.ErrPeticionIntegracionBolsaInvalida, "peticion_integracion_bolsa_invalida"},
		{ports.ErrRespuestaBolsaNoConfiable, "respuesta_bolsa_no_confiable"},
		{ports.ErrIntegracionBolsaNoDisponible, "integracion_bolsa_no_disponible"},
		{ports.ErrEstadoExpedienteNoSeleccionable, "expediente_no_seleccionable"},
		{ports.ErrLimiteIntegracionBolsaExcedido, "limite_integracion_bolsa_excedido"},
		{ports.ErrEvidenciaBolsaNoAutenticada, "evidencia_bolsa_no_autenticada"},
		{ports.ErrCriteriosOfrecimientoPendientesRRHH, "criterios_ofrecimiento_pendientes_rrhh"},
		{ports.ErrAutorizacionDenegada, "autorizacion_denegada"},
		{ports.ErrEjecucionSeleccionLlamamientoInvalida, "ejecucion_seleccion_invalida"},
		{ports.ErrEjecucionSeleccionLlamamientoPendiente, "ejecucion_seleccion_pendiente"},
		{application.ErrServicioSeleccionLlamamientoInvalido, "servicio_seleccion_invalido"},
		{application.ErrSolicitudSeleccionLlamamientoInvalida, "solicitud_seleccion_invalida"},
		{application.ErrSeleccionLlamamientoNoDisponible, "seleccion_no_disponible"},
		{application.ErrResultadoSeleccionLlamamientoNoConfiable, "resultado_seleccion_no_confiable"},
		{application.ErrClaveSeleccionLlamamientoEnColision, "clave_seleccion_en_colision"},
		{application.ErrEjecucionSeleccionLlamamientoConcurrente, "ejecucion_seleccion_concurrente"},
	}
	mejor, profundidadConocida := "", -1
	var interno error
	var marcadorPublico error
	profundidadInterna := -1
	profundidadMarcador := -1
	var visitar func(error, int)
	visitar = func(actual error, profundidad int) {
		if actual == nil {
			return
		}
		descendio := false
		switch envuelto := actual.(type) {
		case interface{ Unwrap() []error }:
			for _, siguiente := range envuelto.Unwrap() {
				if siguiente != nil {
					descendio = true
					visitar(siguiente, profundidad+1)
				}
			}
		case interface{ Unwrap() error }:
			if siguiente := envuelto.Unwrap(); siguiente != nil {
				descendio = true
				visitar(siguiente, profundidad+1)
			}
		}
		if !descendio {
			if errors.Is(actual, application.ErrEjecucionSeleccionLlamamientoIndeterminada) {
				if profundidad > profundidadMarcador {
					marcadorPublico, profundidadMarcador = actual, profundidad
				}
			} else if profundidad > profundidadInterna {
				interno, profundidadInterna = actual, profundidad
			}
		}
		if profundidad > profundidadConocida {
			for _, conocido := range conocidos {
				if errors.Is(actual, conocido.error) {
					mejor, profundidadConocida = conocido.codigo, profundidad
					break
				}
			}
		}
	}
	visitar(causa, 0)
	if mejor != "" {
		return mejor
	}
	if interno == nil {
		interno = marcadorPublico
	}
	return fmt.Sprintf("%T", interno)
}

func codigoSQLSeguro(codigo string) bool {
	return len(codigo) == 5 && strings.IndexFunc(codigo, func(c rune) bool {
		return !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z')
	}) == -1
}

// centinelaSeguroContratacion devuelve solo una clase fija de causas que la
// frontera ya trata como indisponibilidad. Nunca deriva texto ni tipo dinámico
// del error: los errores pueden envolver detalles de infraestructura privados.
func centinelaSeguroContratacion(causa error) string {
	switch {
	case errors.Is(causa, ErrContextoCanalNoDisponible):
		return "contexto_canal_no_disponible"
	case errors.Is(causa, ports.ErrPersistenciaNoDisponible):
		return "persistencia_no_disponible"
	case errors.Is(causa, ports.ErrFuenteNecesidadesAltaNoDisponible):
		return "fuente_necesidades_no_disponible"
	case errors.Is(causa, ports.ErrFlujoNoDisponible):
		return "flujo_no_disponible"
	case errors.Is(causa, ports.ErrMotivoAutorizacionNoDisponible):
		return "motivo_autorizacion_no_disponible"
	case errors.Is(causa, application.ErrServicioRegistroInvalido):
		return "servicio_registro_invalido"
	case errors.Is(causa, application.ErrConsultaRRHHNoDisponible):
		return "consulta_rrhh_no_disponible"
	case errors.Is(causa, application.ErrServicioOperacionAnalisisInvalido),
		errors.Is(causa, application.ErrDependenciaOperacionAnalisisNoDisponible),
		errors.Is(causa, ports.ErrPersistenciaOperacionAnalisisNoDisponible):
		return "analisis_no_disponible"
	case errors.Is(causa, application.ErrPresentacionPropuestaCoberturaNoDisponible),
		errors.Is(causa, application.ErrConfirmacionDecisionCoberturaNoDisponible):
		return "cobertura_no_disponible"
	case errors.Is(causa, application.ErrConfirmacionDecisionCoberturaPendiente):
		return "cobertura_operacion_pendiente"
	case errors.Is(causa, ports.ErrComposicionIncorporacionAplicacion):
		return "incorporacion_no_disponible"
	case errors.Is(causa, context.Canceled):
		return "context_canceled"
	case errors.Is(causa, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "no_clasificado"
	}
}

// Los formatos proceden de los respondedores del adaptador, nunca del cuerpo entrante.
func metadatosErrorContratacion(valor any) (string, string) {
	switch v := valor.(type) {
	case envoltorioErrorCobertura:
		return v.Error.Codigo, v.Error.CorrelacionRef
	case map[string]any:
		if e, ok := v["error"].(map[string]string); ok {
			return e["codigo"], e["correlacion_ref"]
		}
	}
	return "error_interno", "corr_no_disponible"
}
