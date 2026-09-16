package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

type errorPublicoConsultaRRHH struct {
	estado    int
	codigo    string
	claveI18n string
}

var (
	errorPeticionConsultaRRHHNoValida          = nuevoErrorConsultaRRHH(http.StatusBadRequest, "peticion_no_valida")
	errorPeticionConsultaRRHHNoPermitida       = nuevoErrorConsultaRRHH(http.StatusBadRequest, "peticion_no_permitida")
	errorRecursoConsultaRRHHNoEncontrado       = nuevoErrorConsultaRRHH(http.StatusNotFound, "recurso_no_encontrado")
	errorMetodoConsultaRRHHNoPermitido         = nuevoErrorConsultaRRHH(http.StatusMethodNotAllowed, "metodo_no_permitido")
	errorTipoConsultaRRHHNoAdmitido            = nuevoErrorConsultaRRHH(http.StatusUnsupportedMediaType, "tipo_contenido_no_admitido")
	errorRepresentacionConsultaRRHHNoAceptable = nuevoErrorConsultaRRHH(http.StatusNotAcceptable, "representacion_no_aceptable")
	errorCuerpoConsultaRRHHDemasiadoGrande     = nuevoErrorConsultaRRHH(http.StatusRequestEntityTooLarge, "peticion_demasiado_grande")
	errorContenidoConsultaRRHHNoValido         = nuevoErrorConsultaRRHH(http.StatusUnprocessableEntity, "contenido_no_valido")
	errorResultadoConsultaRRHHNoConfiable      = nuevoErrorConsultaRRHH(http.StatusBadGateway, "resultado_no_confiable")
	errorServicioConsultaRRHHNoDisponible      = nuevoErrorConsultaRRHH(http.StatusServiceUnavailable, "servicio_no_disponible")
	errorCancelacionConsultaRRHH               = nuevoErrorConsultaRRHH(http.StatusRequestTimeout, "peticion_cancelada")
	errorPlazoConsultaRRHH                     = nuevoErrorConsultaRRHH(http.StatusGatewayTimeout, "plazo_agotado")
	errorInternoConsultaRRHH                   = nuevoErrorConsultaRRHH(http.StatusInternalServerError, "error_interno")
)

func nuevoErrorConsultaRRHH(
	estado int,
	codigo string,
) errorPublicoConsultaRRHH {
	return errorPublicoConsultaRRHH{
		estado:    estado,
		codigo:    codigo,
		claveI18n: "api.contratacion_temporal.consulta_rrhh.error." + codigo,
	}
}

func errorEntradaConsultaRRHH(err error) errorPublicoConsultaRRHH {
	if errors.Is(err, errCuerpoConsultaRRHHDemasiadoGrande) {
		return errorCuerpoConsultaRRHHDemasiadoGrande
	}
	if errors.Is(err, errContenidoConsultaRRHHNoValido) {
		return errorContenidoConsultaRRHHNoValido
	}
	return errorPeticionConsultaRRHHNoValida
}

func clasificarErrorConsultaRRHH(err error) errorPublicoConsultaRRHH {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionConsultaRRHH
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoConsultaRRHH
	case errors.Is(err, application.ErrConsultaRRHHNoObservable):
		return errorRecursoConsultaRRHHNoEncontrado
	case errors.Is(err, application.ErrResultadoConsultaRRHHNoConfiable):
		return errorResultadoConsultaRRHHNoConfiable
	case errors.Is(err, application.ErrConsultaRRHHNoDisponible):
		return errorServicioConsultaRRHHNoDisponible
	case errors.Is(err, application.ErrServicioConsultaRRHHInvalido):
		return errorServicioConsultaRRHHNoDisponible
	case errors.Is(err, application.ErrSolicitudConsultaRRHHInvalida):
		return errorContenidoConsultaRRHHNoValido
	default:
		return errorInternoConsultaRRHH
	}
}

type envoltorioErrorConsultaRRHH struct {
	Error detalleErrorConsultaRRHH `json:"error"`
}

type detalleErrorConsultaRRHH struct {
	Codigo         string `json:"codigo"`
	ClaveI18n      string `json:"clave_i18n"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func responderErrorConsultaRRHH(
	w http.ResponseWriter,
	r *http.Request,
	causa error,
	problema errorPublicoConsultaRRHH,
) {
	responderJSONConsultaRRHH(
		w, r,
		problema.estado,
		envoltorioErrorConsultaRRHH{Error: detalleErrorConsultaRRHH{
			Codigo: problema.codigo, ClaveI18n: problema.claveI18n,
			CorrelacionRef: nuevaCorrelacionCobertura(),
		}}, causa,
	)
}

func responderJSONConsultaRRHH(w http.ResponseWriter, r *http.Request, estado int, valor any, causas ...error) {
	var causa error
	if len(causas) > 0 {
		causa = causas[0]
	}
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > MaximoRespuestaConsultaRRHHBytes {
		estado = http.StatusInternalServerError
		causa = &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaSerializacion, Causa: err}
		valor = envoltorioErrorConsultaRRHH{
			Error: detalleErrorConsultaRRHH{
				Codigo:         errorInternoConsultaRRHH.codigo,
				ClaveI18n:      errorInternoConsultaRRHH.claveI18n,
				CorrelacionRef: nuevaCorrelacionCobertura(),
			},
		}
		contenido, _ = json.Marshal(valor)
	}
	if estado >= http.StatusInternalServerError {
		if fallo, ok := valor.(envoltorioErrorConsultaRRHH); ok {
			ruta, operacion := "ruta_no_reconocida", "consulta_rrhh"
			if r != nil && r.URL != nil {
				switch r.URL.Path {
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
			slog.Error("consulta RRHH fallida", "operacion", operacion, "ruta", ruta,
				"estado_http", estado, "codigo", fallo.Error.Codigo,
				"etapa", etapa, "sqlstate", sqlstate, "correlacion_ref", fallo.Error.CorrelacionRef)
		}
	}
	aplicarCabecerasCobertura(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}
