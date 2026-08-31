package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoCierreAdministrativoBytes     = 4 * 1024
	profundidadMaximaJSONCierreAdministrativo = 4
	tokensMaximosJSONCierreAdministrativo     = 32
)

var (
	errEntradaCierreAdministrativoInvalida = errors.New(
		"contratacion temporal http: entrada de cierre administrativo invalida",
	)
	errCuerpoCierreAdministrativoDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de cierre administrativo demasiado grande",
	)
)

// cierreAdministrativoEntradaJSON contiene solo intencion. Organizacion,
// actor, perfil, unidad y autorizacion no forman parte del contrato HTTP.
type cierreAdministrativoEntradaJSON struct {
	ExpedienteRef  string `json:"expediente_ref"`
	SeguimientoRef string `json:"seguimiento_ref"`
	// El puntero conserva la presencia explicita y no confunde ausencia con cero.
	VersionEsperada   *uint64 `json:"version_esperada"`
	ClaveIdempotencia string  `json:"clave_idempotencia"`
	TransicionClave   string  `json:"transicion_clave"`
	MotivoClave       string  `json:"motivo_clave"`
}

func (e cierreAdministrativoEntradaJSON) valida() bool {
	return domain.ReferenciaOpacaValida(e.ExpedienteRef) &&
		domain.ReferenciaOpacaValida(e.SeguimientoRef) &&
		e.VersionEsperada != nil &&
		ports.VersionOperacionAnalisisConIncrementoValida(
			*e.VersionEsperada,
		) &&
		ports.ClaveIdempotenciaValida(e.ClaveIdempotencia) &&
		domain.ClaveCatalogo(e.TransicionClave).Valida() &&
		domain.ClaveCatalogo(e.MotivoClave).Valida()
}

func (e cierreAdministrativoEntradaJSON) solicitudPuerto(
	organizacionRef string,
	operacion ports.OperacionCierreAdministrativo,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         operacion,
		OrganizacionRef:   organizacionRef,
		ExpedienteRef:     e.ExpedienteRef,
		SeguimientoRef:    e.SeguimientoRef,
		VersionEsperada:   *e.VersionEsperada,
		ClaveIdempotencia: e.ClaveIdempotencia,
		TransicionClave:   domain.ClaveCatalogo(e.TransicionClave),
		MotivoClave:       domain.ClaveCatalogo(e.MotivoClave),
	}
}

func validarMetadatosCierreAdministrativo(
	r *http.Request,
) *errorPublicoCobertura {
	if r != nil && r.ContentLength > MaximoCuerpoCierreAdministrativoBytes {
		problema := errorCuerpoCierreAdministrativoDemasiadoGrande
		return &problema
	}
	if r == nil || r.ContentLength < -1 || r.ContentLength == 0 ||
		r.Body == nil || r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaCierreAdministrativoPermitida(r) {
		problema := errorPeticionCierreAdministrativoNoValida
		return &problema
	}
	if !cabecerasCierreAdministrativoPermitidas(r.Header) {
		problema := errorPeticionCierreAdministrativoNoPermitida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoCierreAdministrativoNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionCierreAdministrativoNoAceptable
		return &problema
	}
	return nil
}

func transferenciaCierreAdministrativoPermitida(r *http.Request) bool {
	if r == nil || len(r.TransferEncoding) == 0 {
		return r != nil
	}
	return len(r.TransferEncoding) == 1 && r.ContentLength <= 0 &&
		strings.EqualFold(r.TransferEncoding[0], "chunked")
}

func cabecerasCierreAdministrativoPermitidas(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		switch {
		case strings.EqualFold(nombre, "Content-Type"),
			strings.EqualFold(nombre, "Accept"):
		default:
			return false
		}
	}
	return true
}

func cierreAdministrativoDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (cierreAdministrativoEntradaJSON, error) {
	var entrada cierreAdministrativoEntradaJSON
	contenido, err := io.ReadAll(http.MaxBytesReader(
		w,
		r.Body,
		MaximoCuerpoCierreAdministrativoBytes+1,
	))
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return cierreAdministrativoEntradaJSON{},
				errCuerpoCierreAdministrativoDemasiadoGrande
		}
		return cierreAdministrativoEntradaJSON{},
			errEntradaCierreAdministrativoInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return cierreAdministrativoEntradaJSON{},
			errEntradaCierreAdministrativoInvalida
	}
	if len(contenido) > MaximoCuerpoCierreAdministrativoBytes {
		return cierreAdministrativoEntradaJSON{},
			errCuerpoCierreAdministrativoDemasiadoGrande
	}
	if err := validarJSONCierreAdministrativoSinDuplicados(contenido); err != nil {
		return cierreAdministrativoEntradaJSON{}, err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil {
		return cierreAdministrativoEntradaJSON{},
			errEntradaCierreAdministrativoInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return cierreAdministrativoEntradaJSON{},
			errEntradaCierreAdministrativoInvalida
	}
	return entrada, nil
}

func validarJSONCierreAdministrativoSinDuplicados(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	tokens := 0
	var recorrer func(int) (json.Delim, error)
	recorrer = func(profundidad int) (json.Delim, error) {
		tokens++
		if profundidad > profundidadMaximaJSONCierreAdministrativo ||
			tokens > tokensMaximosJSONCierreAdministrativo {
			return 0, errCuerpoCierreAdministrativoDemasiadoGrande
		}
		token, err := decodificador.Token()
		if err != nil || token == nil {
			return 0, errEntradaCierreAdministrativoInvalida
		}
		delimitador, compuesto := token.(json.Delim)
		if !compuesto {
			return 0, nil
		}
		switch delimitador {
		case '{':
			vistas := make(map[string]struct{})
			for decodificador.More() {
				tokens++
				if tokens > tokensMaximosJSONCierreAdministrativo {
					return 0, errCuerpoCierreAdministrativoDemasiadoGrande
				}
				claveToken, err := decodificador.Token()
				clave, correcta := claveToken.(string)
				if err != nil || !correcta {
					return 0, errEntradaCierreAdministrativoInvalida
				}
				if _, repetida := vistas[clave]; repetida {
					return 0, errEntradaCierreAdministrativoInvalida
				}
				vistas[clave] = struct{}{}
				if _, err := recorrer(profundidad + 1); err != nil {
					return 0, err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim('}') {
				return 0, errEntradaCierreAdministrativoInvalida
			}
		case '[':
			for decodificador.More() {
				if _, err := recorrer(profundidad + 1); err != nil {
					return 0, err
				}
			}
			cierre, err := decodificador.Token()
			if err != nil || cierre != json.Delim(']') {
				return 0, errEntradaCierreAdministrativoInvalida
			}
		default:
			return 0, errEntradaCierreAdministrativoInvalida
		}
		return delimitador, nil
	}
	raiz, err := recorrer(0)
	if err != nil || raiz != json.Delim('{') {
		return errEntradaCierreAdministrativoInvalida
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaCierreAdministrativoInvalida
	}
	return nil
}

type envoltorioCierreAdministrativo struct {
	Data cierreAdministrativoSalidaJSON `json:"data"`
}

// cierreAdministrativoSalidaJSON omite operacion, organizacion, expediente,
// seguimiento, actor, perfil, unidad, autorizacion, auditoria y correlacion.
type cierreAdministrativoSalidaJSON struct {
	ReciboRef          string `json:"recibo_ref"`
	VersionSeguimiento uint64 `json:"version_seguimiento"`
}

func proyectarCierreAdministrativo(
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	resultado ports.ResultadoCierreAdministrativo,
) (cierreAdministrativoSalidaJSON, int, bool) {
	if resultado.ValidarPara(solicitud) != nil {
		return cierreAdministrativoSalidaJSON{}, 0, false
	}
	estadoHTTP := http.StatusCreated
	if resultado.EsReplayConfirmado() {
		estadoHTTP = http.StatusOK
	}
	return cierreAdministrativoSalidaJSON{
		ReciboRef:          resultado.ReciboRef(),
		VersionSeguimiento: resultado.VersionSeguimiento(),
	}, estadoHTTP, true
}

var (
	errorPeticionCierreAdministrativoNoValida = nuevoErrorCierreAdministrativo(
		http.StatusBadRequest,
		"peticion_no_valida",
	)
	errorPeticionCierreAdministrativoNoPermitida = nuevoErrorCierreAdministrativo(
		http.StatusBadRequest,
		"peticion_no_permitida",
	)
	errorRecursoCierreAdministrativoNoEncontrado = nuevoErrorCierreAdministrativo(
		http.StatusNotFound,
		"recurso_no_encontrado",
	)
	errorMetodoCierreAdministrativoNoPermitido = nuevoErrorCierreAdministrativo(
		http.StatusMethodNotAllowed,
		"metodo_no_permitido",
	)
	errorTipoCierreAdministrativoNoAdmitido = nuevoErrorCierreAdministrativo(
		http.StatusUnsupportedMediaType,
		"tipo_contenido_no_admitido",
	)
	errorRepresentacionCierreAdministrativoNoAceptable = nuevoErrorCierreAdministrativo(
		http.StatusNotAcceptable,
		"representacion_no_aceptable",
	)
	errorCuerpoCierreAdministrativoDemasiadoGrande = nuevoErrorCierreAdministrativo(
		http.StatusRequestEntityTooLarge,
		"peticion_demasiado_grande",
	)
	errorContenidoCierreAdministrativoInvalido = nuevoErrorCierreAdministrativo(
		http.StatusUnprocessableEntity,
		"contenido_no_valido",
	)
	errorAccesoCierreAdministrativoDenegado = nuevoErrorCierreAdministrativo(
		http.StatusForbidden,
		"acceso_denegado",
	)
	errorVersionCierreAdministrativoEnConflicto = nuevoErrorCierreAdministrativo(
		http.StatusConflict,
		"version_en_conflicto",
	)
	errorClaveCierreAdministrativoReutilizada = nuevoErrorCierreAdministrativo(
		http.StatusConflict,
		"clave_idempotencia_reutilizada",
	)
	errorResultadoCierreAdministrativoNoConfiable = nuevoErrorCierreAdministrativo(
		http.StatusBadGateway,
		"resultado_no_confiable",
	)
	errorServicioCierreAdministrativoNoDisponible = nuevoErrorCierreAdministrativo(
		http.StatusServiceUnavailable,
		"servicio_no_disponible",
	)
	errorCancelacionCierreAdministrativo = nuevoErrorCierreAdministrativo(
		http.StatusRequestTimeout,
		"peticion_cancelada",
	)
	errorPlazoCierreAdministrativo = nuevoErrorCierreAdministrativo(
		http.StatusGatewayTimeout,
		"plazo_agotado",
	)
)

func nuevoErrorCierreAdministrativo(
	estado int,
	codigo string,
) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado: estado,
		codigo: codigo,
		claveI18n: "api.contratacion_temporal.cierre_administrativo.error." +
			codigo,
	}
}

func errorEntradaCierreAdministrativo(err error) errorPublicoCobertura {
	if errors.Is(err, errCuerpoCierreAdministrativoDemasiadoGrande) {
		return errorCuerpoCierreAdministrativoDemasiadoGrande
	}
	return errorPeticionCierreAdministrativoNoValida
}

func clasificarErrorCierreAdministrativoHTTP(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionCierreAdministrativo
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoCierreAdministrativo
	case errors.Is(err, application.ErrSolicitudCierreAdministrativoInvalida),
		errors.Is(err, ports.ErrSolicitudCierreAdministrativoInvalida):
		return errorContenidoCierreAdministrativoInvalido
	case errors.Is(err, application.ErrCierreAdministrativoNoPermitido),
		errors.Is(err, ports.ErrCierreAdministrativoDenegado):
		return errorAccesoCierreAdministrativoDenegado
	case errors.Is(err, application.ErrVersionCierreAdministrativoEnConflicto):
		return errorVersionCierreAdministrativoEnConflicto
	case errors.Is(err, ports.ErrClaveIdempotenciaCierreAdministrativoUsada):
		return errorClaveCierreAdministrativoReutilizada
	case errors.Is(err, application.ErrResultadoCierreAdministrativoInvalido),
		errors.Is(err, ports.ErrResultadoCierreAdministrativoInvalido):
		return errorResultadoCierreAdministrativoNoConfiable
	case errors.Is(err, application.ErrServicioCierreAdministrativoInvalido),
		errors.Is(err, application.ErrCierreAdministrativoNoDisponible):
		return errorServicioCierreAdministrativoNoDisponible
	default:
		return errorServicioCierreAdministrativoNoDisponible
	}
}

func responderErrorCierreAdministrativo(
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
