package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoComunicacionLlamamientoBytes = 4 * 1024
	EsquemaRegistroComunicacionLlamamiento   = "" +
		"vec.contratacion-temporal.registro-comunicacion-llamamiento.v1"
	EsquemaResolucionComunicacionLlamamiento = "" +
		"vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1"
)

var (
	errEntradaComunicacionLlamamientoInvalida = errors.New(
		"contratacion temporal http: entrada de comunicacion de llamamiento invalida",
	)
	errContenidoComunicacionLlamamientoNoValido = errors.New(
		"contratacion temporal http: contenido de comunicacion de llamamiento no valido",
	)
	errCuerpoComunicacionLlamamientoDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de comunicacion de llamamiento demasiado grande",
	)
)

type registroComunicacionLlamamientoJSON struct {
	ClaveIdempotencia string `json:"clave_idempotencia"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	LlamamientoRef    string `json:"llamamiento_ref"`
	VersionEsperada   uint64 `json:"version_esperada"`
	PruebaEntregaRef  string `json:"prueba_entrega_ref"`
}

func (e registroComunicacionLlamamientoJSON) solicitud() (
	ports.SolicitudRegistrarComunicacionLlamamiento,
	error,
) {
	solicitud := ports.SolicitudRegistrarComunicacionLlamamiento{
		ClaveIdempotencia: e.ClaveIdempotencia,
		OrganizacionRef:   e.OrganizacionRef,
		ExpedienteRef:     e.ExpedienteRef,
		LlamamientoRef:    e.LlamamientoRef,
		VersionEsperada:   e.VersionEsperada,
		PruebaEntregaRef:  e.PruebaEntregaRef,
	}
	if solicitud.Validar() != nil {
		return ports.SolicitudRegistrarComunicacionLlamamiento{},
			errContenidoComunicacionLlamamientoNoValido
	}
	return solicitud, nil
}

type resolucionComunicacionLlamamientoJSON struct {
	ClaveIdempotencia     string `json:"clave_idempotencia"`
	OrganizacionRef       string `json:"organizacion_ref"`
	ExpedienteRef         string `json:"expediente_ref"`
	LlamamientoRef        string `json:"llamamiento_ref"`
	ComunicacionRef       string `json:"comunicacion_ref"`
	VersionEsperada       uint64 `json:"version_esperada"`
	Respuesta             string `json:"respuesta"`
	PruebaRespuestaRef    string `json:"prueba_respuesta_ref"`
	RevisionRespuestaRRHH bool   `json:"revision_respuesta_rrhh,omitempty"`
	RevisionPlazoRRHH     bool   `json:"revision_plazo_rrhh,omitempty"`
	CriterioValidacionRef string `json:"criterio_validacion_ref,omitempty"`
}

func (e resolucionComunicacionLlamamientoJSON) solicitud() (
	ports.SolicitudResolverLlamamiento,
	error,
) {
	solicitud := ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia:     e.ClaveIdempotencia,
		OrganizacionRef:       e.OrganizacionRef,
		ExpedienteRef:         e.ExpedienteRef,
		LlamamientoRef:        e.LlamamientoRef,
		ComunicacionRef:       e.ComunicacionRef,
		VersionEsperada:       e.VersionEsperada,
		Respuesta:             ports.RespuestaLlamamiento(e.Respuesta),
		PruebaRespuestaRef:    e.PruebaRespuestaRef,
		RevisionRespuestaRRHH: e.RevisionRespuestaRRHH,
		RevisionPlazoRRHH:     e.RevisionPlazoRRHH,
		CriterioValidacionRef: e.CriterioValidacionRef,
	}
	if solicitud.Validar() != nil {
		return ports.SolicitudResolverLlamamiento{},
			errContenidoComunicacionLlamamientoNoValido
	}
	return solicitud, nil
}

func validarMetadatosComunicacionLlamamiento(
	r *http.Request,
) *errorPublicoCobertura {
	if r != nil && r.ContentLength > MaximoCuerpoComunicacionLlamamientoBytes {
		problema := errorCuerpoComunicacionLlamamientoDemasiadoGrande
		return &problema
	}
	problema := validarMetadatosCobertura(r)
	if problema == nil {
		return nil
	}
	traducido := nuevoErrorComunicacionLlamamiento(
		problema.estado,
		problema.codigo,
	)
	return &traducido
}

func solicitudRegistroComunicacionDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (ports.SolicitudRegistrarComunicacionLlamamiento, error) {
	var entrada registroComunicacionLlamamientoJSON
	if err := decodificarComunicacionLlamamiento(w, r, &entrada); err != nil {
		return ports.SolicitudRegistrarComunicacionLlamamiento{}, err
	}
	return entrada.solicitud()
}

func solicitudResolucionLlamamientoDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (ports.SolicitudResolverLlamamiento, error) {
	var entrada resolucionComunicacionLlamamientoJSON
	if err := decodificarComunicacionLlamamiento(w, r, &entrada); err != nil {
		return ports.SolicitudResolverLlamamiento{}, err
	}
	return entrada.solicitud()
}

func decodificarComunicacionLlamamiento(
	w http.ResponseWriter,
	r *http.Request,
	destino any,
) error {
	lector := http.MaxBytesReader(
		w,
		r.Body,
		MaximoCuerpoComunicacionLlamamientoBytes+1,
	)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return errCuerpoComunicacionLlamamientoDemasiadoGrande
		}
		return errEntradaComunicacionLlamamientoInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return errEntradaComunicacionLlamamientoInvalida
	}
	if len(contenido) > MaximoCuerpoComunicacionLlamamientoBytes {
		return errCuerpoComunicacionLlamamientoDemasiadoGrande
	}
	if err := validarJSONAltaSinDuplicados(contenido); err != nil {
		if errors.Is(err, errCuerpoAltaDemasiadoGrande) {
			return errCuerpoComunicacionLlamamientoDemasiadoGrande
		}
		return errEntradaComunicacionLlamamientoInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaComunicacionLlamamientoInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaComunicacionLlamamientoInvalida
	}
	canon, err := json.Marshal(destino)
	if err != nil || !bytes.Equal(contenido, canon) {
		return errContenidoComunicacionLlamamientoNoValido
	}
	return nil
}

type envoltorioRegistroComunicacionLlamamiento struct {
	Data registroComunicacionLlamamientoSalidaJSON `json:"data"`
}

type registroComunicacionLlamamientoSalidaJSON struct {
	Esquema           string `json:"esquema"`
	EstadoLocal       string `json:"estado_local"`
	ComunicacionRef   string `json:"comunicacion_ref"`
	ReciboRef         string `json:"recibo_ref"`
	AuditoriaRef      string `json:"auditoria_ref"`
	VersionResultante uint64 `json:"version_resultante"`
	RespuestaHasta    string `json:"respuesta_hasta,omitempty"`
	RegistradaEn      string `json:"registrada_en,omitempty"`
	IntencionEnvioRef string `json:"intencion_envio_ref,omitempty"`
}

func proyectarRegistroComunicacionLlamamiento(
	solicitud ports.SolicitudRegistrarComunicacionLlamamiento,
	resultado ports.ComunicacionProbatoria,
) (registroComunicacionLlamamientoSalidaJSON, int, bool) {
	if resultado.ValidarPara(solicitud) != nil {
		return registroComunicacionLlamamientoSalidaJSON{}, 0, false
	}
	estadoHTTP := http.StatusCreated
	if resultado.EsReplayConfirmado() {
		estadoHTTP = http.StatusOK
	}
	salida := registroComunicacionLlamamientoSalidaJSON{
		Esquema:           EsquemaRegistroComunicacionLlamamiento,
		EstadoLocal:       string(resultado.Estado),
		ComunicacionRef:   resultado.ComunicacionRef,
		ReciboRef:         resultado.ReciboRef,
		AuditoriaRef:      resultado.AuditoriaRef,
		VersionResultante: resultado.VersionResultante,
	}
	if resultado.EsRegistroLocal() {
		salida.RegistradaEn = resultado.RegistradaEn.UTC().Format(time.RFC3339Nano)
		salida.IntencionEnvioRef = resultado.IntencionEnvioRef
	} else {
		salida.RespuestaHasta = resultado.RespuestaHasta.UTC().Format(time.RFC3339Nano)
	}
	return salida, estadoHTTP, true
}

type envoltorioResolucionComunicacionLlamamiento struct {
	Data resolucionComunicacionLlamamientoSalidaJSON `json:"data"`
}

type resolucionComunicacionLlamamientoSalidaJSON struct {
	Esquema            string                             `json:"esquema"`
	Respuesta          string                             `json:"respuesta"`
	EstadoPlazo        string                             `json:"estado_plazo"`
	EstadoLocal        string                             `json:"estado_local"`
	ResolucionRef      string                             `json:"resolucion_ref"`
	ReciboLocalRef     string                             `json:"recibo_local_ref"`
	AuditoriaRef       string                             `json:"auditoria_ref"`
	IntencionSiguiente *intencionSiguienteLlamamientoJSON `json:"intencion_siguiente,omitempty"`
	VersionResultante  uint64                             `json:"version_resultante"`
	ResueltaEn         string                             `json:"resuelta_en"`
}

// intencionSiguienteLlamamientoJSON solo refleja el registro local de outbox.
// Incluso "despachada" es un estado local y no acredita recepción ni efecto.
type intencionSiguienteLlamamientoJSON struct {
	Referencia    string `json:"referencia"`
	EstadoLocal   string `json:"estado_local"`
	ActualizadaEn string `json:"actualizada_en"`
}

func proyectarResolucionComunicacionLlamamiento(
	solicitud ports.SolicitudResolverLlamamiento,
	resultado ports.ResultadoResolucionLlamamiento,
) (resolucionComunicacionLlamamientoSalidaJSON, int, bool) {
	if resultado.ValidarPara(solicitud) != nil {
		return resolucionComunicacionLlamamientoSalidaJSON{}, 0, false
	}
	var intencion *intencionSiguienteLlamamientoJSON
	if resultado.IntencionSiguiente != (ports.IntencionOutboxSiguienteCandidato{}) {
		intencion = &intencionSiguienteLlamamientoJSON{
			Referencia:  resultado.IntencionSiguiente.IntencionRef,
			EstadoLocal: string(resultado.IntencionSiguiente.Estado),
			ActualizadaEn: resultado.IntencionSiguiente.ActualizadaEn.UTC().Format(
				time.RFC3339Nano,
			),
		}
	}
	estadoHTTP := http.StatusCreated
	if resultado.EsReplayConfirmado() {
		estadoHTTP = http.StatusOK
	}
	return resolucionComunicacionLlamamientoSalidaJSON{
		Esquema:            EsquemaResolucionComunicacionLlamamiento,
		Respuesta:          string(solicitud.Respuesta),
		EstadoPlazo:        string(resultado.EstadoPlazo),
		EstadoLocal:        string(resultado.Estado),
		ResolucionRef:      resultado.ResolucionRef,
		ReciboLocalRef:     resultado.ReciboLocalRef,
		AuditoriaRef:       resultado.AuditoriaRef,
		IntencionSiguiente: intencion,
		VersionResultante:  resultado.VersionResultante,
		ResueltaEn: resultado.ResueltaEn.UTC().Format(
			time.RFC3339Nano,
		),
	}, estadoHTTP, true
}

var (
	errorPeticionComunicacionLlamamientoNoValida = nuevoErrorComunicacionLlamamiento(
		http.StatusBadRequest,
		"peticion_no_valida",
	)
	errorPeticionComunicacionLlamamientoNoPermitida = nuevoErrorComunicacionLlamamiento(
		http.StatusBadRequest,
		"peticion_no_permitida",
	)
	errorRecursoComunicacionLlamamientoNoEncontrado = nuevoErrorComunicacionLlamamiento(
		http.StatusNotFound,
		"recurso_no_encontrado",
	)
	errorMetodoComunicacionLlamamientoNoPermitido = nuevoErrorComunicacionLlamamiento(
		http.StatusMethodNotAllowed,
		"metodo_no_permitido",
	)
	errorTipoComunicacionLlamamientoNoAdmitido = nuevoErrorComunicacionLlamamiento(
		http.StatusUnsupportedMediaType,
		"tipo_contenido_no_admitido",
	)
	errorRepresentacionComunicacionLlamamientoNoAceptable = nuevoErrorComunicacionLlamamiento(
		http.StatusNotAcceptable,
		"representacion_no_aceptable",
	)
	errorCuerpoComunicacionLlamamientoDemasiadoGrande = nuevoErrorComunicacionLlamamiento(
		http.StatusRequestEntityTooLarge,
		"peticion_demasiado_grande",
	)
	errorContenidoComunicacionLlamamientoInvalido = nuevoErrorComunicacionLlamamiento(
		http.StatusUnprocessableEntity,
		"contenido_no_valido",
	)
	errorAccesoComunicacionLlamamientoDenegado = nuevoErrorComunicacionLlamamiento(
		http.StatusForbidden,
		"acceso_denegado",
	)
	errorVersionComunicacionLlamamientoEnConflicto = nuevoErrorComunicacionLlamamiento(
		http.StatusConflict,
		"version_en_conflicto",
	)
	errorClaveComunicacionLlamamientoReutilizada = nuevoErrorComunicacionLlamamiento(
		http.StatusConflict,
		"clave_idempotencia_reutilizada",
	)
	errorResultadoComunicacionLlamamientoNoConfiable = nuevoErrorComunicacionLlamamiento(
		http.StatusBadGateway,
		"resultado_no_confiable",
	)
	errorServicioComunicacionLlamamientoNoDisponible = nuevoErrorComunicacionLlamamiento(
		http.StatusServiceUnavailable,
		"servicio_no_disponible",
	)
	errorCancelacionComunicacionLlamamiento = nuevoErrorComunicacionLlamamiento(
		http.StatusRequestTimeout,
		"peticion_cancelada",
	)
	errorPlazoComunicacionLlamamiento = nuevoErrorComunicacionLlamamiento(
		http.StatusGatewayTimeout,
		"plazo_agotado",
	)
)

func nuevoErrorComunicacionLlamamiento(
	estado int,
	codigo string,
) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado: estado,
		codigo: codigo,
		claveI18n: "api.contratacion_temporal.comunicacion_llamamiento.error." +
			codigo,
	}
}

func errorEntradaComunicacionLlamamiento(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, errCuerpoComunicacionLlamamientoDemasiadoGrande):
		return errorCuerpoComunicacionLlamamientoDemasiadoGrande
	case errors.Is(err, errContenidoComunicacionLlamamientoNoValido):
		return errorContenidoComunicacionLlamamientoInvalido
	default:
		return errorPeticionComunicacionLlamamientoNoValida
	}
}

func clasificarErrorComunicacionLlamamientoHTTP(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, application.ErrValidacionRespuestaLlamamientoPendiente):
		return nuevoErrorComunicacionLlamamiento(http.StatusConflict, "validacion_respuesta_pendiente")
	case errors.Is(err, application.ErrClaveComunicacionLlamamientoEnColision):
		return errorClaveComunicacionLlamamientoReutilizada
	case errors.Is(err, application.ErrVersionComunicacionLlamamientoEnConflicto):
		return errorVersionComunicacionLlamamientoEnConflicto
	case errors.Is(err, application.ErrComunicacionLlamamientoDenegada):
		return errorAccesoComunicacionLlamamientoDenegado
	case errors.Is(err, context.Canceled):
		return errorCancelacionComunicacionLlamamiento
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoComunicacionLlamamiento
	case errors.Is(err, application.ErrSolicitudComunicacionLlamamientoInvalida):
		return errorContenidoComunicacionLlamamientoInvalido
	case errors.Is(err, application.ErrResultadoComunicacionLlamamientoNoConfiable):
		return errorResultadoComunicacionLlamamientoNoConfiable
	case errors.Is(err, application.ErrServicioComunicacionLlamamientoInvalido),
		errors.Is(err, application.ErrComunicacionLlamamientoNoDisponible):
		return errorServicioComunicacionLlamamientoNoDisponible
	default:
		return errorServicioComunicacionLlamamientoNoDisponible
	}
}

func responderErrorComunicacionLlamamiento(
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
