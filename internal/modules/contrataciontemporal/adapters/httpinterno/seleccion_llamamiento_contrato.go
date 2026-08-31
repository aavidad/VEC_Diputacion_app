package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const EsquemaReciboSeleccionLlamamiento = "" +
	"vec.contratacion-temporal.recibo-seleccion-llamamiento.v1"

var errCanonSeleccionLlamamientoInvalido = errors.New(
	"contratacion temporal http: canon de seleccion y llamamiento invalido",
)

type seleccionLlamamientoEntradaJSON struct {
	ClaveIdempotencia string `json:"clave_idempotencia"`
}

func seleccionLlamamientoDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (seleccionLlamamientoEntradaJSON, error) {
	var entrada seleccionLlamamientoEntradaJSON
	var contenido bytes.Buffer
	cuerpoOriginal := r.Body
	r.Body = io.NopCloser(io.TeeReader(cuerpoOriginal, &contenido))
	defer func() {
		r.Body = cuerpoOriginal
	}()
	if err := decodificarCobertura(w, r, &entrada); err != nil {
		return seleccionLlamamientoEntradaJSON{}, err
	}
	canon, err := json.Marshal(entrada)
	if err != nil || !bytes.Equal(contenido.Bytes(), canon) {
		return seleccionLlamamientoEntradaJSON{},
			errCanonSeleccionLlamamientoInvalido
	}
	if !ports.ClaveIdempotenciaValida(entrada.ClaveIdempotencia) {
		return seleccionLlamamientoEntradaJSON{},
			errContenidoCoberturaNoValido
	}
	return entrada, nil
}

type envoltorioReciboSeleccionLlamamiento struct {
	Data reciboSeleccionLlamamientoJSON `json:"data"`
}

// reciboSeleccionLlamamientoJSON omite organización, expediente, correlación,
// selección seudonimizada, posición, auditoría, evento y toda evidencia.
type reciboSeleccionLlamamientoJSON struct {
	Esquema      string `json:"esquema"`
	Estado       string `json:"estado"`
	ReciboRef    string `json:"recibo_ref"`
	ConfirmadaEn string `json:"confirmada_en"`
}

func proyectarReciboSeleccionLlamamiento(
	recibo application.DatosReciboSeleccionLlamamientoParaAdaptador,
) (reciboSeleccionLlamamientoJSON, bool) {
	if !domain.ReferenciaOpacaValida(recibo.ReciboRef) ||
		!domain.InstanteUTCCanonico(recibo.ConfirmadaEn) {
		return reciboSeleccionLlamamientoJSON{}, false
	}
	return reciboSeleccionLlamamientoJSON{
		Esquema:      EsquemaReciboSeleccionLlamamiento,
		Estado:       "confirmado",
		ReciboRef:    recibo.ReciboRef,
		ConfirmadaEn: recibo.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
	}, true
}

var (
	errorPeticionSeleccionLlamamientoNoValida = nuevoErrorSeleccionLlamamiento(
		http.StatusBadRequest,
		"peticion_no_valida",
	)
	errorPeticionSeleccionLlamamientoNoPermitida = nuevoErrorSeleccionLlamamiento(
		http.StatusBadRequest,
		"peticion_no_permitida",
	)
	errorRecursoSeleccionLlamamientoNoEncontrado = nuevoErrorSeleccionLlamamiento(
		http.StatusNotFound,
		"recurso_no_encontrado",
	)
	errorMetodoSeleccionLlamamientoNoPermitido = nuevoErrorSeleccionLlamamiento(
		http.StatusMethodNotAllowed,
		"metodo_no_permitido",
	)
	errorTipoSeleccionLlamamientoNoAdmitido = nuevoErrorSeleccionLlamamiento(
		http.StatusUnsupportedMediaType,
		"tipo_contenido_no_admitido",
	)
	errorRepresentacionSeleccionLlamamientoNoAceptable = nuevoErrorSeleccionLlamamiento(
		http.StatusNotAcceptable,
		"representacion_no_aceptable",
	)
	errorCuerpoSeleccionLlamamientoDemasiadoGrande = nuevoErrorSeleccionLlamamiento(
		http.StatusRequestEntityTooLarge,
		"peticion_demasiado_grande",
	)
	errorContenidoSeleccionLlamamientoInvalido = nuevoErrorSeleccionLlamamiento(
		http.StatusUnprocessableEntity,
		"contenido_no_valido",
	)
	errorConflictoSeleccionLlamamientoNoReintentable = nuevoErrorSeleccionLlamamiento(
		http.StatusConflict,
		"conflicto_no_reintentable",
	)
	errorSeleccionLlamamientoNoDisponible = nuevoErrorSeleccionLlamamiento(
		http.StatusConflict,
		"seleccion_no_disponible",
	)
	errorResultadoSeleccionLlamamientoNoConfiable = nuevoErrorSeleccionLlamamiento(
		http.StatusBadGateway,
		"resultado_no_confiable",
	)
	errorServicioSeleccionLlamamientoNoDisponible = nuevoErrorSeleccionLlamamiento(
		http.StatusServiceUnavailable,
		"servicio_no_disponible",
	)
	errorCancelacionSeleccionLlamamiento = nuevoErrorSeleccionLlamamiento(
		http.StatusRequestTimeout,
		"peticion_cancelada",
	)
	errorPlazoSeleccionLlamamiento = nuevoErrorSeleccionLlamamiento(
		http.StatusGatewayTimeout,
		"plazo_agotado",
	)
)

func nuevoErrorSeleccionLlamamiento(
	estado int,
	codigo string,
) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado:    estado,
		codigo:    codigo,
		claveI18n: "api.contratacion_temporal.seleccion_llamamiento.error." + codigo,
	}
}

func validarMetadatosSeleccionLlamamiento(
	r *http.Request,
) *errorPublicoCobertura {
	problema := validarMetadatosCobertura(r)
	if problema == nil {
		return nil
	}
	traducido := nuevoErrorSeleccionLlamamiento(
		problema.estado,
		problema.codigo,
	)
	return &traducido
}

func errorEntradaSeleccionLlamamiento(err error) errorPublicoCobertura {
	base := errorEntradaCobertura(err)
	if errors.Is(err, errCanonSeleccionLlamamientoInvalido) {
		base = errorContenidoCoberturaInvalido
	}
	return nuevoErrorSeleccionLlamamiento(base.estado, base.codigo)
}

func clasificarErrorSeleccionLlamamiento(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, application.ErrClaveSeleccionLlamamientoEnColision),
		errors.Is(err, application.ErrEjecucionSeleccionLlamamientoConcurrente),
		errors.Is(err, application.ErrEjecucionSeleccionLlamamientoIndeterminada):
		// La indeterminación puede venir unida a cancelación o plazo; nunca se
		// degrada a un fallo que sugiera repetir el efecto.
		return errorConflictoSeleccionLlamamientoNoReintentable
	case errors.Is(err, context.Canceled):
		return errorCancelacionSeleccionLlamamiento
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoSeleccionLlamamiento
	case errors.Is(err, application.ErrSolicitudSeleccionLlamamientoInvalida):
		return errorContenidoSeleccionLlamamientoInvalido
	case errors.Is(err, application.ErrSeleccionLlamamientoNoDisponible):
		return errorSeleccionLlamamientoNoDisponible
	case errors.Is(err, application.ErrResultadoSeleccionLlamamientoNoConfiable),
		errors.Is(err, ports.ErrRespuestaBolsaNoConfiable):
		return errorResultadoSeleccionLlamamientoNoConfiable
	case errors.Is(err, application.ErrServicioSeleccionLlamamientoInvalido),
		errors.Is(err, ports.ErrIntegracionBolsaNoDisponible):
		return errorServicioSeleccionLlamamientoNoDisponible
	default:
		return errorServicioSeleccionLlamamientoNoDisponible
	}
}

func responderErrorSeleccionLlamamiento(
	w http.ResponseWriter,
	problema errorPublicoCobertura,
) {
	responderJSONCobertura(
		w,
		problema.estado,
		envoltorioErrorCobertura{Error: detalleErrorCobertura{
			Codigo:         problema.codigo,
			ClaveI18n:      problema.claveI18n,
			CorrelacionRef: nuevaCorrelacionCobertura(),
		}},
	)
}
