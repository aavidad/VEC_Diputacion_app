package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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
	problema errorPublicoConsultaRRHH,
) {
	responderJSONConsultaRRHH(
		w,
		problema.estado,
		envoltorioErrorConsultaRRHH{Error: detalleErrorConsultaRRHH{
			Codigo: problema.codigo, ClaveI18n: problema.claveI18n,
			CorrelacionRef: nuevaCorrelacionCobertura(),
		}},
	)
}

func responderJSONConsultaRRHH(w http.ResponseWriter, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > MaximoRespuestaConsultaRRHHBytes {
		estado = http.StatusInternalServerError
		contenido, _ = json.Marshal(envoltorioErrorConsultaRRHH{
			Error: detalleErrorConsultaRRHH{
				Codigo:         errorInternoConsultaRRHH.codigo,
				ClaveI18n:      errorInternoConsultaRRHH.claveI18n,
				CorrelacionRef: nuevaCorrelacionCobertura(),
			},
		})
	}
	aplicarCabecerasCobertura(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}
