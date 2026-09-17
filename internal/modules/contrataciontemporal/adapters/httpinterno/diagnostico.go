package httpinterno

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// registrarFalloContratacion solo registra metadatos cerrados, nunca el texto de la causa.
func registrarFalloContratacion(r *http.Request, estado int, codigo, correlacion string, causa error) {
	if estado < http.StatusInternalServerError {
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
		if len(falloInterno.CodigoSQL) == 5 && strings.IndexFunc(falloInterno.CodigoSQL, func(c rune) bool {
			return !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z')
		}) == -1 {
			sqlstate = falloInterno.CodigoSQL
		}
	}
	if etapaCobertura, ok := application.EtapaDiagnosticoDePresentacionPropuestaCobertura(causa); ok {
		etapa = string(etapaCobertura)
	}
	slog.Error("operación de Contratación fallida", "operacion", operacion, "ruta", ruta,
		"estado_http", estado, "codigo", codigo,
		"etapa", etapa, "sqlstate", sqlstate,
		"centinela", centinelaSeguroContratacion(causa), "correlacion_ref", correlacion)
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
	case errors.Is(causa, ports.ErrFlujoNoDisponible):
		return "flujo_no_disponible"
	case errors.Is(causa, ports.ErrMotivoAutorizacionNoDisponible):
		return "motivo_autorizacion_no_disponible"
	case errors.Is(causa, application.ErrServicioRegistroInvalido):
		return "servicio_registro_invalido"
	case errors.Is(causa, application.ErrConsultaRRHHNoDisponible):
		return "consulta_rrhh_no_disponible"
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
