package httpinterno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type errorPublicoCobertura struct {
	estado    int
	codigo    string
	claveI18n string
}

var (
	errorPeticionCoberturaNoValida          = nuevoErrorCobertura(http.StatusBadRequest, "peticion_no_valida")
	errorPeticionCoberturaNoPermitida       = nuevoErrorCobertura(http.StatusBadRequest, "peticion_no_permitida")
	errorRecursoCoberturaNoEncontrado       = nuevoErrorCobertura(http.StatusNotFound, "recurso_no_encontrado")
	errorMetodoCoberturaNoPermitido         = nuevoErrorCobertura(http.StatusMethodNotAllowed, "metodo_no_permitido")
	errorTipoCoberturaNoAdmitido            = nuevoErrorCobertura(http.StatusUnsupportedMediaType, "tipo_contenido_no_admitido")
	errorRepresentacionCoberturaNoAceptable = nuevoErrorCobertura(http.StatusNotAcceptable, "representacion_no_aceptable")
	errorContenidoCoberturaInvalido         = nuevoErrorCobertura(http.StatusUnprocessableEntity, "contenido_no_valido")
	errorAccesoCoberturaDenegado            = nuevoErrorCobertura(http.StatusForbidden, "acceso_denegado")
	errorConflictoCobertura                 = nuevoErrorCobertura(http.StatusConflict, "conflicto")
	errorResultadoCoberturaNoConfiable      = nuevoErrorCobertura(http.StatusBadGateway, "resultado_no_confiable")
	errorServicioCoberturaNoDisponible      = nuevoErrorCobertura(http.StatusServiceUnavailable, "servicio_no_disponible")
	errorCancelacionCobertura               = nuevoErrorCobertura(http.StatusRequestTimeout, "peticion_cancelada")
	errorPlazoCobertura                     = nuevoErrorCobertura(http.StatusGatewayTimeout, "plazo_agotado")
	errorInternoCobertura                   = nuevoErrorCobertura(http.StatusInternalServerError, "error_interno")
)

func nuevoErrorCobertura(estado int, codigo string) errorPublicoCobertura {
	return errorPublicoCobertura{estado: estado, codigo: codigo, claveI18n: "api.contratacion_temporal.cobertura.error." + codigo}
}
func errorEntradaCobertura(err error) errorPublicoCobertura {
	if errors.Is(err, errCuerpoCoberturaDemasiadoGrande) {
		return nuevoErrorCobertura(http.StatusRequestEntityTooLarge, "peticion_demasiado_grande")
	}
	if errors.Is(err, errContenidoCoberturaNoValido) {
		return errorContenidoCoberturaInvalido
	}
	return errorPeticionCoberturaNoValida
}
func clasificarErrorCobertura(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionCobertura
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoCobertura
	case errors.Is(err, ports.ErrAutorizacionDenegada), errors.Is(err, application.ErrPresentacionPropuestaCoberturaDenegada), errors.Is(err, application.ErrConfirmacionDecisionCoberturaDenegada):
		return errorAccesoCoberturaDenegado
	case errors.Is(err, application.ErrPresentacionPropuestaCoberturaEnConflicto), errors.Is(err, application.ErrConfirmacionDecisionCoberturaEnConflicto), errors.Is(err, application.ErrConfirmacionDecisionCoberturaOcupada):
		return errorConflictoCobertura
	case errors.Is(err, application.ErrPresentacionPropuestaCoberturaNoConfiable), errors.Is(err, application.ErrConfirmacionDecisionCoberturaNoConfiable), errors.Is(err, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida):
		return errorResultadoCoberturaNoConfiable
	case errors.Is(err, application.ErrPresentacionPropuestaCoberturaNoDisponible), errors.Is(err, application.ErrConfirmacionDecisionCoberturaNoDisponible):
		return errorServicioCoberturaNoDisponible
	default:
		return errorInternoCobertura
	}
}

type envoltorioErrorCobertura struct {
	Error detalleErrorCobertura `json:"error"`
}
type detalleErrorCobertura struct {
	Codigo         string `json:"codigo"`
	ClaveI18n      string `json:"clave_i18n"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func responderErrorCobertura(w http.ResponseWriter, problema errorPublicoCobertura) {
	responderJSONCobertura(w, problema.estado, envoltorioErrorCobertura{Error: detalleErrorCobertura{Codigo: problema.codigo, ClaveI18n: problema.claveI18n, CorrelacionRef: nuevaCorrelacionCobertura()}})
}
func responderJSONCobertura(w http.ResponseWriter, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > 16*1024 {
		estado = http.StatusInternalServerError
		contenido, _ = json.Marshal(envoltorioErrorCobertura{Error: detalleErrorCobertura{Codigo: errorInternoCobertura.codigo, ClaveI18n: errorInternoCobertura.claveI18n, CorrelacionRef: nuevaCorrelacionCobertura()}})
	}
	aplicarCabecerasCobertura(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}
func aplicarCabecerasCobertura(w http.ResponseWriter) {
	for _, nombre := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Access-Control-Allow-Headers", "Access-Control-Allow-Methods", "Access-Control-Expose-Headers", "Content-Encoding", "Location", "Retry-After"} {
		w.Header().Del(nombre)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("X-Frame-Options", "DENY")
}
func nuevaCorrelacionCobertura() string {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return "corr_no_disponible"
	}
	return "corr_" + hex.EncodeToString(aleatorio)
}
