package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoPropuestaFormalizacionBytes = 128 * 1024
	EsquemaPropuestaFormalizacion           = "" +
		"vec.contratacion-temporal.propuesta-formalizacion-local.v1"
	profundidadMaximaJSONPropuestaFormalizacion = 8
	tokensMaximosJSONPropuestaFormalizacion     = 2048
)

var (
	errEntradaPropuestaFormalizacionInvalida = errors.New(
		"contratacion temporal http: entrada de propuesta de formalizacion invalida",
	)
	errContenidoPropuestaFormalizacionNoValido = errors.New(
		"contratacion temporal http: contenido de propuesta de formalizacion no valido",
	)
	errCuerpoPropuestaFormalizacionDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de propuesta de formalizacion demasiado grande",
	)
)

type snapshotPropuestaFormalizacionJSON struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (s snapshotPropuestaFormalizacionJSON) puerto() ports.SnapshotGobernadoFormalizacion {
	return ports.SnapshotGobernadoFormalizacion{
		Referencia: s.Referencia, Version: s.Version, HuellaSHA256: s.HuellaSHA256,
	}
}

type anexoPropuestaFormalizacionJSON struct {
	DocumentoRef string `json:"documento_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	TamanoBytes  uint64 `json:"tamano_bytes"`
}

func (a anexoPropuestaFormalizacionJSON) puerto() ports.AnexoPropuestaFormalizacion {
	return ports.AnexoPropuestaFormalizacion{
		DocumentoRef: a.DocumentoRef,
		Version:      a.Version,
		HuellaSHA256: a.HuellaSHA256,
		TamanoBytes:  a.TamanoBytes,
	}
}

// propuestaFormalizacionEntradaJSON no admite organizacion, identidad,
// perfil, actor ni permisos: esos datos pertenecen a la frontera confiable.
type propuestaFormalizacionEntradaJSON struct {
	ClaveIdempotencia                string                             `json:"clave_idempotencia"`
	ExpedienteRef                    string                             `json:"expediente_ref"`
	LlamamientoRef                   string                             `json:"llamamiento_ref"`
	ResolucionLlamamientoAceptadaRef string                             `json:"resolucion_llamamiento_aceptada_ref"`
	ReciboResolucionAceptadaRef      string                             `json:"recibo_resolucion_aceptada_ref"`
	VersionEsperada                  uint64                             `json:"version_esperada"`
	TipoFormalizacion                snapshotPropuestaFormalizacionJSON `json:"tipo_formalizacion"`
	Plantilla                        snapshotPropuestaFormalizacionJSON `json:"plantilla"`
	Anexos                           *[]anexoPropuestaFormalizacionJSON `json:"anexos"`
	PoliticaFirma                    snapshotPropuestaFormalizacionJSON `json:"politica_firma"`
	PlanFirma                        snapshotPropuestaFormalizacionJSON `json:"plan_firma"`
}

func (e propuestaFormalizacionEntradaJSON) solicitud(
	contexto ContextoServidorPropuestaFormalizacion,
) (ports.SolicitudPropuestaFormalizacion, error) {
	if e.Anexos == nil || len(*e.Anexos) > ports.MaximoAnexosPropuestaFormalizacion {
		return ports.SolicitudPropuestaFormalizacion{},
			errContenidoPropuestaFormalizacionNoValido
	}
	anexos := make([]ports.AnexoPropuestaFormalizacion, len(*e.Anexos))
	for indice, anexo := range *e.Anexos {
		anexos[indice] = anexo.puerto()
	}
	solicitud, err := (ports.SolicitudPropuestaFormalizacion{
		ClaveIdempotencia:                e.ClaveIdempotencia,
		OrganizacionRef:                  contexto.OrganizacionRef,
		ExpedienteRef:                    e.ExpedienteRef,
		LlamamientoRef:                   e.LlamamientoRef,
		ResolucionLlamamientoAceptadaRef: e.ResolucionLlamamientoAceptadaRef,
		ReciboResolucionAceptadaRef:      e.ReciboResolucionAceptadaRef,
		VersionEsperada:                  e.VersionEsperada,
		TipoFormalizacion:                e.TipoFormalizacion.puerto(),
		Plantilla:                        e.Plantilla.puerto(),
		Anexos:                           anexos,
		PoliticaFirma:                    e.PoliticaFirma.puerto(),
		PlanFirma:                        e.PlanFirma.puerto(),
	}).Normalizar()
	if err != nil || solicitud.Validar() != nil {
		return ports.SolicitudPropuestaFormalizacion{},
			errContenidoPropuestaFormalizacionNoValido
	}
	return solicitud, nil
}

func validarMetadatosPropuestaFormalizacion(
	r *http.Request,
) *errorPublicoCobertura {
	if r != nil && r.ContentLength > MaximoCuerpoPropuestaFormalizacionBytes {
		problema := errorCuerpoPropuestaFormalizacionDemasiadoGrande
		return &problema
	}
	if r == nil || r.ContentLength == 0 || r.Body == nil ||
		r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaAltaPermitida(r.TransferEncoding) {
		problema := errorPeticionPropuestaFormalizacionNoValida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoPropuestaFormalizacionNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionPropuestaFormalizacionNoAceptable
		return &problema
	}
	if cabeceraCoberturaProhibida(r.Header) {
		problema := errorPeticionPropuestaFormalizacionNoPermitida
		return &problema
	}
	return nil
}

func propuestaFormalizacionDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (propuestaFormalizacionEntradaJSON, error) {
	var entrada propuestaFormalizacionEntradaJSON
	contenido, err := io.ReadAll(http.MaxBytesReader(
		w,
		r.Body,
		MaximoCuerpoPropuestaFormalizacionBytes+1,
	))
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return propuestaFormalizacionEntradaJSON{},
				errCuerpoPropuestaFormalizacionDemasiadoGrande
		}
		return propuestaFormalizacionEntradaJSON{},
			errEntradaPropuestaFormalizacionInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return propuestaFormalizacionEntradaJSON{},
			errEntradaPropuestaFormalizacionInvalida
	}
	if len(contenido) > MaximoCuerpoPropuestaFormalizacionBytes {
		return propuestaFormalizacionEntradaJSON{},
			errCuerpoPropuestaFormalizacionDemasiadoGrande
	}
	if err := validarJSONPropuestaFormalizacionSinDuplicados(contenido); err != nil {
		return propuestaFormalizacionEntradaJSON{}, err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil {
		return propuestaFormalizacionEntradaJSON{},
			errEntradaPropuestaFormalizacionInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return propuestaFormalizacionEntradaJSON{},
			errEntradaPropuestaFormalizacionInvalida
	}
	canon, err := json.Marshal(entrada)
	if err != nil || !bytes.Equal(contenido, canon) {
		return propuestaFormalizacionEntradaJSON{},
			errContenidoPropuestaFormalizacionNoValido
	}
	return entrada, nil
}

func validarJSONPropuestaFormalizacionSinDuplicados(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	tokens := 0
	var recorrer func(int) error
	recorrer = func(profundidad int) error {
		tokens++
		if profundidad > profundidadMaximaJSONPropuestaFormalizacion ||
			tokens > tokensMaximosJSONPropuestaFormalizacion {
			return errCuerpoPropuestaFormalizacionDemasiadoGrande
		}
		token, err := decodificador.Token()
		if err != nil {
			return errEntradaPropuestaFormalizacionInvalida
		}
		delimitador, compuesto := token.(json.Delim)
		if !compuesto {
			if token == nil {
				return errEntradaPropuestaFormalizacionInvalida
			}
			if numero, esNumero := token.(json.Number); esNumero &&
				!patronEnteroJSONAlta.MatchString(numero.String()) {
				return errEntradaPropuestaFormalizacionInvalida
			}
			return nil
		}
		switch delimitador {
		case '{':
			vistas := map[string]struct{}{}
			for decodificador.More() {
				claveToken, err := decodificador.Token()
				clave, correcta := claveToken.(string)
				if err != nil || !correcta || clave != strings.ToLower(clave) {
					return errEntradaPropuestaFormalizacionInvalida
				}
				if _, repetida := vistas[clave]; repetida {
					return errEntradaPropuestaFormalizacionInvalida
				}
				vistas[clave] = struct{}{}
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim('}') {
				return errEntradaPropuestaFormalizacionInvalida
			}
		case '[':
			for decodificador.More() {
				if err := recorrer(profundidad + 1); err != nil {
					return err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim(']') {
				return errEntradaPropuestaFormalizacionInvalida
			}
		default:
			return errEntradaPropuestaFormalizacionInvalida
		}
		return nil
	}
	if err := recorrer(0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaPropuestaFormalizacionInvalida
	}
	return nil
}

type envoltorioPropuestaFormalizacion struct {
	Data propuestaFormalizacionSalidaJSON `json:"data"`
}

// propuestaFormalizacionSalidaJSON acredita solo el resultado local. Omite
// auditoria interna y no contiene documento, firma, registro ni descarga.
type propuestaFormalizacionSalidaJSON struct {
	Esquema           string `json:"esquema"`
	EstadoLocal       string `json:"estado_local"`
	PropuestaRef      string `json:"propuesta_ref"`
	ReciboLocalRef    string `json:"recibo_local_ref"`
	VersionResultante uint64 `json:"version_resultante"`
	ConfirmadaEn      string `json:"confirmada_en"`
}

func proyectarPropuestaFormalizacion(
	solicitud ports.SolicitudPropuestaFormalizacion,
	resultado ports.ResultadoPropuestaFormalizacion,
) (propuestaFormalizacionSalidaJSON, int, bool) {
	if resultado.ValidarPara(solicitud) != nil {
		return propuestaFormalizacionSalidaJSON{}, 0, false
	}
	estadoHTTP := http.StatusCreated
	if resultado.EsReplayConfirmado() {
		estadoHTTP = http.StatusOK
	}
	return propuestaFormalizacionSalidaJSON{
		Esquema:           EsquemaPropuestaFormalizacion,
		EstadoLocal:       string(resultado.Estado),
		PropuestaRef:      resultado.PropuestaRef,
		ReciboLocalRef:    resultado.ReciboLocalRef,
		VersionResultante: resultado.VersionResultante,
		ConfirmadaEn: resultado.ConfirmadaEn.UTC().Format(
			time.RFC3339Nano,
		),
	}, estadoHTTP, true
}

var (
	errorPeticionPropuestaFormalizacionNoValida = nuevoErrorPropuestaFormalizacion(
		http.StatusBadRequest,
		"peticion_no_valida",
	)
	errorPeticionPropuestaFormalizacionNoPermitida = nuevoErrorPropuestaFormalizacion(
		http.StatusBadRequest,
		"peticion_no_permitida",
	)
	errorRecursoPropuestaFormalizacionNoEncontrado = nuevoErrorPropuestaFormalizacion(
		http.StatusNotFound,
		"recurso_no_encontrado",
	)
	errorMetodoPropuestaFormalizacionNoPermitido = nuevoErrorPropuestaFormalizacion(
		http.StatusMethodNotAllowed,
		"metodo_no_permitido",
	)
	errorTipoPropuestaFormalizacionNoAdmitido = nuevoErrorPropuestaFormalizacion(
		http.StatusUnsupportedMediaType,
		"tipo_contenido_no_admitido",
	)
	errorRepresentacionPropuestaFormalizacionNoAceptable = nuevoErrorPropuestaFormalizacion(
		http.StatusNotAcceptable,
		"representacion_no_aceptable",
	)
	errorCuerpoPropuestaFormalizacionDemasiadoGrande = nuevoErrorPropuestaFormalizacion(
		http.StatusRequestEntityTooLarge,
		"peticion_demasiado_grande",
	)
	errorContenidoPropuestaFormalizacionInvalido = nuevoErrorPropuestaFormalizacion(
		http.StatusUnprocessableEntity,
		"contenido_no_valido",
	)
	errorAccesoPropuestaFormalizacionDenegado = nuevoErrorPropuestaFormalizacion(
		http.StatusForbidden,
		"acceso_denegado",
	)
	errorVersionPropuestaFormalizacionEnConflicto = nuevoErrorPropuestaFormalizacion(
		http.StatusConflict,
		"version_en_conflicto",
	)
	errorClavePropuestaFormalizacionReutilizada = nuevoErrorPropuestaFormalizacion(
		http.StatusConflict,
		"clave_idempotencia_reutilizada",
	)
	errorResolucionPropuestaFormalizacionNoAceptada = nuevoErrorPropuestaFormalizacion(
		http.StatusConflict,
		"resolucion_no_aceptada",
	)
	errorResultadoPropuestaFormalizacionNoConfiable = nuevoErrorPropuestaFormalizacion(
		http.StatusBadGateway,
		"resultado_no_confiable",
	)
	errorServicioPropuestaFormalizacionNoDisponible = nuevoErrorPropuestaFormalizacion(
		http.StatusServiceUnavailable,
		"servicio_no_disponible",
	)
	errorCancelacionPropuestaFormalizacion = nuevoErrorPropuestaFormalizacion(
		http.StatusRequestTimeout,
		"peticion_cancelada",
	)
	errorPlazoPropuestaFormalizacion = nuevoErrorPropuestaFormalizacion(
		http.StatusGatewayTimeout,
		"plazo_agotado",
	)
)

func nuevoErrorPropuestaFormalizacion(
	estado int,
	codigo string,
) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado: estado,
		codigo: codigo,
		claveI18n: "api.contratacion_temporal.propuesta_formalizacion.error." +
			codigo,
	}
}

func errorEntradaPropuestaFormalizacion(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, errCuerpoPropuestaFormalizacionDemasiadoGrande):
		return errorCuerpoPropuestaFormalizacionDemasiadoGrande
	case errors.Is(err, errContenidoPropuestaFormalizacionNoValido):
		return errorContenidoPropuestaFormalizacionInvalido
	default:
		return errorPeticionPropuestaFormalizacionNoValida
	}
}

func clasificarErrorPropuestaFormalizacionHTTP(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionPropuestaFormalizacion
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoPropuestaFormalizacion
	case errors.Is(err, application.ErrPropuestaFormalizacionDenegada):
		return errorAccesoPropuestaFormalizacionDenegado
	case errors.Is(err, application.ErrVersionPropuestaFormalizacionEnConflicto):
		return errorVersionPropuestaFormalizacionEnConflicto
	case errors.Is(err, application.ErrClavePropuestaFormalizacionEnColision):
		return errorClavePropuestaFormalizacionReutilizada
	case errors.Is(err, application.ErrResolucionFormalizacionNoAceptada):
		return errorResolucionPropuestaFormalizacionNoAceptada
	case errors.Is(err, application.ErrSolicitudPropuestaFormalizacionInvalida):
		return errorContenidoPropuestaFormalizacionInvalido
	case errors.Is(err, application.ErrResultadoPropuestaFormalizacionNoConfiable):
		return errorResultadoPropuestaFormalizacionNoConfiable
	case errors.Is(err, application.ErrServicioPropuestaFormalizacionInvalido),
		errors.Is(err, application.ErrPropuestaFormalizacionNoDisponible):
		return errorServicioPropuestaFormalizacionNoDisponible
	default:
		return errorServicioPropuestaFormalizacionNoDisponible
	}
}

func responderErrorPropuestaFormalizacion(
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
