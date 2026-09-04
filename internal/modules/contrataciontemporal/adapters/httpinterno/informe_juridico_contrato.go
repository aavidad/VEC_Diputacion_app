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
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoInformeJuridicoBytes = 4 * 1024
	maximosTokensInformeJuridicoJSON = 8
	esquemaReciboInformeJuridico     = "vec.contratacion-temporal.recibo-informe-juridico.v1"
)

var (
	errEntradaInformeJuridicoInvalida = errors.New(
		"contratacion temporal http: entrada de informe juridico invalida",
	)
	errContenidoInformeJuridicoInvalido = errors.New(
		"contratacion temporal http: contenido de informe juridico invalido",
	)
	errCuerpoInformeJuridicoDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de informe juridico demasiado grande",
	)
)

type informeJuridicoEntradaJSON struct {
	ExpedienteRef     string  `json:"expediente_ref"`
	VersionEsperada   *uint64 `json:"version_esperada"`
	ClaveIdempotencia string  `json:"clave_idempotencia"`
}

type entradaInformeJuridicoHTTP struct {
	expedienteRef     string
	versionEsperada   uint64
	claveIdempotencia string
}

func (e entradaInformeJuridicoHTTP) valida() bool {
	return domain.ReferenciaOpacaValida(e.expedienteRef) &&
		ports.VersionOperacionAnalisisConIncrementoValida(e.versionEsperada) &&
		ports.ClaveIdempotenciaValida(e.claveIdempotencia)
}

func (e entradaInformeJuridicoHTTP) solicitud(
	c ContextoCanalInformeJuridico,
) application.SolicitudEmitirInformeJuridico {
	return application.SolicitudEmitirInformeJuridico{
		AutenticacionRef:  c.AutenticacionRef,
		SesionRef:         c.SesionRef,
		PerfilRef:         c.PerfilRef,
		OrganizacionRef:   c.OrganizacionRef,
		ExpedienteRef:     e.expedienteRef,
		VersionEsperada:   e.versionEsperada,
		ClaveIdempotencia: e.claveIdempotencia,
	}
}

func validarMetadatosInformeJuridico(
	r *http.Request,
) *errorPublicoCobertura {
	if r != nil && r.ContentLength > MaximoCuerpoInformeJuridicoBytes {
		problema := errorCuerpoInformeJuridicoDemasiadoGrande
		return &problema
	}
	if r == nil || r.ContentLength < -1 || r.ContentLength == 0 ||
		r.Body == nil || r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaInformeJuridicoPermitida(r) {
		problema := errorPeticionInformeJuridicoNoValida
		return &problema
	}
	if cabeceraCoberturaProhibida(r.Header) {
		problema := errorPeticionInformeJuridicoNoPermitida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoInformeJuridicoNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionInformeJuridicoNoAceptable
		return &problema
	}
	return nil
}

func transferenciaInformeJuridicoPermitida(r *http.Request) bool {
	if r == nil || len(r.TransferEncoding) == 0 {
		return r != nil
	}
	return len(r.TransferEncoding) == 1 && r.ContentLength <= 0 &&
		strings.EqualFold(r.TransferEncoding[0], "chunked")
}

func informeJuridicoDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (entradaInformeJuridicoHTTP, error) {
	contenido, err := io.ReadAll(http.MaxBytesReader(
		w,
		r.Body,
		MaximoCuerpoInformeJuridicoBytes+1,
	))
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return entradaInformeJuridicoHTTP{}, errCuerpoInformeJuridicoDemasiadoGrande
		}
		return entradaInformeJuridicoHTTP{}, errEntradaInformeJuridicoInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return entradaInformeJuridicoHTTP{}, errEntradaInformeJuridicoInvalida
	}
	if len(contenido) > MaximoCuerpoInformeJuridicoBytes {
		return entradaInformeJuridicoHTTP{}, errCuerpoInformeJuridicoDemasiadoGrande
	}
	if err := validarJSONInformeJuridicoCerrado(contenido); err != nil {
		return entradaInformeJuridicoHTTP{}, err
	}
	var entrada informeJuridicoEntradaJSON
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil {
		return entradaInformeJuridicoHTTP{}, errEntradaInformeJuridicoInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return entradaInformeJuridicoHTTP{}, errEntradaInformeJuridicoInvalida
	}
	if entrada.VersionEsperada == nil {
		return entradaInformeJuridicoHTTP{}, errContenidoInformeJuridicoInvalido
	}
	resultado := entradaInformeJuridicoHTTP{
		expedienteRef:     entrada.ExpedienteRef,
		versionEsperada:   *entrada.VersionEsperada,
		claveIdempotencia: entrada.ClaveIdempotencia,
	}
	if !resultado.valida() {
		return entradaInformeJuridicoHTTP{}, errContenidoInformeJuridicoInvalido
	}
	return resultado, nil
}

func validarJSONInformeJuridicoCerrado(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('{') {
		return errEntradaInformeJuridicoInvalida
	}
	vistas := make(map[string]struct{}, 3)
	tokens := 1
	for decodificador.More() {
		tokens++
		claveToken, err := decodificador.Token()
		clave, correcta := claveToken.(string)
		if err != nil || !correcta || tokens > maximosTokensInformeJuridicoJSON ||
			!claveInformeJuridicoJSONExacta(clave) {
			return errEntradaInformeJuridicoInvalida
		}
		if _, repetida := vistas[clave]; repetida {
			return errEntradaInformeJuridicoInvalida
		}
		vistas[clave] = struct{}{}
		tokens++
		valor, err := decodificador.Token()
		if err != nil || valor == nil || tokens > maximosTokensInformeJuridicoJSON {
			return errEntradaInformeJuridicoInvalida
		}
		if _, compuesto := valor.(json.Delim); compuesto {
			return errEntradaInformeJuridicoInvalida
		}
	}
	cierre, err := decodificador.Token()
	if err != nil || cierre != json.Delim('}') {
		return errEntradaInformeJuridicoInvalida
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaInformeJuridicoInvalida
	}
	return nil
}

func claveInformeJuridicoJSONExacta(clave string) bool {
	switch clave {
	case "expediente_ref", "version_esperada", "clave_idempotencia":
		return true
	default:
		return false
	}
}

func reciboInformeJuridicoSeguro(
	r ports.ReciboInformeJuridico,
	c ContextoCanalInformeJuridico,
	e entradaInformeJuridicoHTTP,
) bool {
	return r.Operacion == "preparar" && r.OrganizacionRef == c.OrganizacionRef &&
		r.ExpedienteRef == e.expedienteRef && r.VersionAnterior == e.versionEsperada &&
		ports.VersionOperacionAnalisisConIncrementoValida(r.VersionAnterior) &&
		r.VersionResultante == r.VersionAnterior+1 &&
		r.VersionResultante <= ports.MaximoEnteroSeguroOperacionAnalisis &&
		domain.ReferenciaOpacaValida(r.InformeRef) &&
		domain.ReferenciaOpacaValida(r.DocumentoRef) &&
		ports.VersionOperacionAnalisisValida(r.VersionDocumento) &&
		r.Formato == ports.FormatoInformeJuridicoDesarrollo &&
		strings.TrimSpace(r.Nombre) != "" &&
		huellaSHA256InformeJuridicoValida(r.HuellaDocumentoSHA256) &&
		huellaSHA256InformeJuridicoValida(r.HuellaBorradorSHA256) &&
		domain.ReferenciaOpacaValida(r.ReciboRef) &&
		domain.ReferenciaOpacaValida(r.AuditoriaRef) &&
		domain.ReferenciaOpacaValida(r.EventoRef) &&
		domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) &&
		ports.SelloHMACSHA256Valido(r.AmbitoIdempotenciaHMAC) &&
		ports.SelloHMACSHA256Valido(r.HuellaPeticionHMAC) &&
		len(r.ContenidoDesarrollo) > 0 && len(r.ContenidoDesarrollo) <= 256*1024 &&
		utf8.ValidString(r.ContenidoDesarrollo) &&
		strings.Contains(r.ContenidoDesarrollo, "DOCUMENTO DE DESARROLLO") &&
		strings.Contains(r.ContenidoDesarrollo, "SIN FIRMA NI VALIDEZ JURIDICA") &&
		domain.InstanteUTCCanonico(r.ConfirmadaEn)
}

func huellaSHA256InformeJuridicoValida(valor string) bool {
	if len(valor) != 64 || valor == strings.Repeat("0", 64) {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') &&
			(caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

type envoltorioReciboInformeJuridico struct {
	Data reciboInformeJuridicoJSON `json:"data"`
}

type reciboInformeJuridicoJSON struct {
	Esquema               string `json:"esquema"`
	Operacion             string `json:"operacion"`
	ExpedienteRef         string `json:"expediente_ref"`
	VersionResultante     uint64 `json:"version_resultante"`
	InformeRef            string `json:"informe_ref"`
	DocumentoRef          string `json:"documento_ref"`
	VersionDocumento      uint64 `json:"version_documento"`
	Formato               string `json:"formato"`
	Nombre                string `json:"nombre"`
	HuellaDocumentoSHA256 string `json:"huella_documento_sha256"`
	ReciboRef             string `json:"recibo_ref"`
	AuditoriaRef          string `json:"auditoria_ref"`
	EventoRef             string `json:"evento_ref"`
	ContenidoDesarrollo   string `json:"contenido_desarrollo"`
	ConfirmadaEn          string `json:"confirmada_en"`
}

func responderExitoInformeJuridico(
	w http.ResponseWriter,
	r ports.ReciboInformeJuridico,
) {
	responderJSONCobertura(
		w,
		http.StatusCreated,
		envoltorioReciboInformeJuridico{Data: reciboInformeJuridicoJSON{
			Esquema:               esquemaReciboInformeJuridico,
			Operacion:             r.Operacion,
			ExpedienteRef:         r.ExpedienteRef,
			VersionResultante:     r.VersionResultante,
			InformeRef:            r.InformeRef,
			DocumentoRef:          r.DocumentoRef,
			VersionDocumento:      r.VersionDocumento,
			Formato:               r.Formato,
			Nombre:                r.Nombre,
			HuellaDocumentoSHA256: r.HuellaDocumentoSHA256,
			ReciboRef:             r.ReciboRef,
			AuditoriaRef:          r.AuditoriaRef,
			EventoRef:             r.EventoRef,
			ContenidoDesarrollo:   r.ContenidoDesarrollo,
			ConfirmadaEn:          r.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
		}},
	)
}

var (
	errorPeticionInformeJuridicoNoValida = nuevoErrorInformeJuridico(
		http.StatusBadRequest, "peticion_no_valida",
	)
	errorPeticionInformeJuridicoNoPermitida = nuevoErrorInformeJuridico(
		http.StatusBadRequest, "peticion_no_permitida",
	)
	errorRecursoInformeJuridicoNoEncontrado = nuevoErrorInformeJuridico(
		http.StatusNotFound, "recurso_no_encontrado",
	)
	errorMetodoInformeJuridicoNoPermitido = nuevoErrorInformeJuridico(
		http.StatusMethodNotAllowed, "metodo_no_permitido",
	)
	errorTipoInformeJuridicoNoAdmitido = nuevoErrorInformeJuridico(
		http.StatusUnsupportedMediaType, "tipo_contenido_no_admitido",
	)
	errorRepresentacionInformeJuridicoNoAceptable = nuevoErrorInformeJuridico(
		http.StatusNotAcceptable, "representacion_no_aceptable",
	)
	errorCuerpoInformeJuridicoDemasiadoGrande = nuevoErrorInformeJuridico(
		http.StatusRequestEntityTooLarge, "peticion_demasiado_grande",
	)
	errorContenidoInformeJuridicoInvalido = nuevoErrorInformeJuridico(
		http.StatusUnprocessableEntity, "contenido_no_valido",
	)
	errorAutenticacionInformeJuridicoRequerida = nuevoErrorInformeJuridico(
		http.StatusUnauthorized, "autenticacion_requerida",
	)
	errorAccesoInformeJuridicoDenegado = nuevoErrorInformeJuridico(
		http.StatusForbidden, "acceso_denegado",
	)
	errorConflictoInformeJuridico = nuevoErrorInformeJuridico(
		http.StatusConflict, "conflicto",
	)
	errorResultadoInformeJuridicoNoConfiable = nuevoErrorInformeJuridico(
		http.StatusBadGateway, "resultado_no_confiable",
	)
	errorServicioInformeJuridicoNoDisponible = nuevoErrorInformeJuridico(
		http.StatusServiceUnavailable, "servicio_no_disponible",
	)
	errorCancelacionInformeJuridico = nuevoErrorInformeJuridico(
		http.StatusRequestTimeout, "peticion_cancelada",
	)
	errorPlazoInformeJuridico = nuevoErrorInformeJuridico(
		http.StatusGatewayTimeout, "plazo_agotado",
	)
)

func nuevoErrorInformeJuridico(estado int, codigo string) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado:    estado,
		codigo:    codigo,
		claveI18n: "api.contratacion_temporal.informe_juridico.error." + codigo,
	}
}

func errorEntradaInformeJuridico(err error) errorPublicoCobertura {
	if errors.Is(err, errCuerpoInformeJuridicoDemasiadoGrande) {
		return errorCuerpoInformeJuridicoDemasiadoGrande
	}
	if errors.Is(err, errContenidoInformeJuridicoInvalido) {
		return errorContenidoInformeJuridicoInvalido
	}
	return errorPeticionInformeJuridicoNoValida
}

func clasificarErrorInformeJuridicoHTTP(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionInformeJuridico
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoInformeJuridico
	case errors.Is(err, ErrContextoCanalAusente),
		errors.Is(err, ErrContextoCanalCaducado):
		return errorAutenticacionInformeJuridicoRequerida
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada),
		errors.Is(err, application.ErrInformeJuridicoDenegado),
		errors.Is(err, ports.ErrAutorizacionDenegada):
		return errorAccesoInformeJuridicoDenegado
	case errors.Is(err, ports.ErrClaveIdempotenciaUsada),
		errors.Is(err, domain.ErrVersionEnConflicto):
		return errorConflictoInformeJuridico
	case errors.Is(err, application.ErrSolicitudInformeJuridicoInvalida),
		errors.Is(err, ports.ErrPreparacionInformeJuridicoInvalida),
		errors.Is(err, domain.ErrDatoInvalido):
		return errorContenidoInformeJuridicoInvalido
	case errors.Is(err, application.ErrResultadoInformeJuridicoNoConfiable),
		errors.Is(err, ports.ErrResultadoInformeJuridicoNoConfiable):
		return errorResultadoInformeJuridicoNoConfiable
	case errors.Is(err, ErrContextoCanalNoDisponible),
		errors.Is(err, application.ErrServicioInformeJuridicoInvalido),
		errors.Is(err, application.ErrGeneracionInformeJuridicoNoDisponible),
		errors.Is(err, ports.ErrPersistenciaInformeJuridicoNoDisponible):
		return errorServicioInformeJuridicoNoDisponible
	default:
		return errorServicioInformeJuridicoNoDisponible
	}
}

func responderErrorInformeJuridico(
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
