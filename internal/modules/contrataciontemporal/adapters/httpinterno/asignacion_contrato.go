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
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoAsignacionBytes = 8 * 1024
	maximosTokensAsignacionJSON = 20
	esquemaReciboAsignacion     = "vec.contratacion-temporal.recibo-asignacion.v1"
)

var (
	errEntradaAsignacionInvalida = errors.New(
		"contratacion temporal http: entrada de asignacion invalida",
	)
	errContenidoAsignacionInvalido = errors.New(
		"contratacion temporal http: contenido de asignacion invalido",
	)
	errCuerpoAsignacionDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de asignacion demasiado grande",
	)
)

type asignacionEntradaJSON struct {
	ExpedienteRef     string  `json:"expediente_ref"`
	VersionEsperada   *uint64 `json:"version_esperada"`
	ClaveIdempotencia string  `json:"clave_idempotencia"`
	UnidadRef         string  `json:"unidad_ref"`
	ResponsableRef    string  `json:"responsable_ref"`
}

type reasignacionEntradaJSON struct {
	ExpedienteRef      string  `json:"expediente_ref"`
	VersionEsperada    *uint64 `json:"version_esperada"`
	ClaveIdempotencia  string  `json:"clave_idempotencia"`
	UnidadRef          string  `json:"unidad_ref"`
	ResponsableRef     string  `json:"responsable_ref"`
	MotivoReasignacion string  `json:"motivo_reasignacion_clave"`
	Observaciones      string  `json:"observaciones"`
}

type entradaAsignacionHTTP struct {
	expedienteRef      string
	versionEsperada    uint64
	claveIdempotencia  string
	unidadRef          string
	responsableRef     string
	motivoReasignacion domain.ClaveCatalogo
	observaciones      string
}

func (e entradaAsignacionHTTP) valida(
	operacion ports.TipoOperacionAsignacion,
) bool {
	if !operacion.Valida() ||
		!domain.ReferenciaOpacaValida(e.expedienteRef) ||
		!ports.VersionOperacionAnalisisConIncrementoValida(
			e.versionEsperada,
		) ||
		!ports.ClaveIdempotenciaValida(e.claveIdempotencia) ||
		!domain.ReferenciaOpacaValida(e.unidadRef) ||
		!domain.ReferenciaOpacaValida(e.responsableRef) {
		return false
	}
	if operacion == ports.OperacionRegistrarAsignacion {
		return e.motivoReasignacion == "" && e.observaciones == ""
	}
	return e.motivoReasignacion.Valida() &&
		textoAsignacionHTTPValido(e.observaciones, 1000, false)
}

func (e entradaAsignacionHTTP) solicitudAsignar(
	c ContextoCanalAsignacion,
) application.SolicitudAsignarUnidad {
	return application.SolicitudAsignarUnidad{
		AutenticacionRef:  c.AutenticacionRef,
		SesionRef:         c.SesionRef,
		PerfilRef:         c.PerfilRef,
		OrganizacionRef:   c.OrganizacionRef,
		ExpedienteRef:     e.expedienteRef,
		VersionEsperada:   e.versionEsperada,
		ClaveIdempotencia: e.claveIdempotencia,
		UnidadRef:         e.unidadRef,
		ResponsableRef:    e.responsableRef,
	}
}

func (e entradaAsignacionHTTP) solicitudReasignar(
	c ContextoCanalAsignacion,
) application.SolicitudReasignarUnidad {
	return application.SolicitudReasignarUnidad{
		AutenticacionRef:        c.AutenticacionRef,
		SesionRef:               c.SesionRef,
		PerfilRef:               c.PerfilRef,
		OrganizacionRef:         c.OrganizacionRef,
		ExpedienteRef:           e.expedienteRef,
		VersionEsperada:         e.versionEsperada,
		ClaveIdempotencia:       e.claveIdempotencia,
		UnidadRef:               e.unidadRef,
		ResponsableRef:          e.responsableRef,
		MotivoReasignacionClave: e.motivoReasignacion,
		Observaciones:           e.observaciones,
	}
}

func validarMetadatosAsignacion(
	r *http.Request,
) *errorPublicoCobertura {
	if r != nil && r.ContentLength > MaximoCuerpoAsignacionBytes {
		problema := errorCuerpoAsignacionDemasiadoGrande
		return &problema
	}
	if r == nil || r.ContentLength < -1 || r.ContentLength == 0 ||
		r.Body == nil || r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaAsignacionPermitida(r) {
		problema := errorPeticionAsignacionNoValida
		return &problema
	}
	if !cabecerasAsignacionPermitidas(r.Header) {
		problema := errorPeticionAsignacionNoPermitida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoAsignacionNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionAsignacionNoAceptable
		return &problema
	}
	return nil
}

func transferenciaAsignacionPermitida(r *http.Request) bool {
	if r == nil || len(r.TransferEncoding) == 0 {
		return r != nil
	}
	return len(r.TransferEncoding) == 1 && r.ContentLength <= 0 &&
		strings.EqualFold(r.TransferEncoding[0], "chunked")
}

func cabecerasAsignacionPermitidas(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		if !strings.EqualFold(nombre, "Content-Type") &&
			!strings.EqualFold(nombre, "Accept") {
			return false
		}
	}
	return true
}

func asignacionDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
	operacion ports.TipoOperacionAsignacion,
) (entradaAsignacionHTTP, error) {
	if operacion == ports.OperacionRegistrarAsignacion {
		var entrada asignacionEntradaJSON
		if err := decodificarAsignacion(w, r, operacion, &entrada); err != nil {
			return entradaAsignacionHTTP{}, err
		}
		if entrada.VersionEsperada == nil {
			return entradaAsignacionHTTP{}, errContenidoAsignacionInvalido
		}
		resultado := entradaAsignacionHTTP{
			expedienteRef:     entrada.ExpedienteRef,
			versionEsperada:   *entrada.VersionEsperada,
			claveIdempotencia: entrada.ClaveIdempotencia,
			unidadRef:         entrada.UnidadRef,
			responsableRef:    entrada.ResponsableRef,
		}
		if !resultado.valida(operacion) {
			return entradaAsignacionHTTP{}, errContenidoAsignacionInvalido
		}
		return resultado, nil
	}
	var entrada reasignacionEntradaJSON
	if err := decodificarAsignacion(w, r, operacion, &entrada); err != nil {
		return entradaAsignacionHTTP{}, err
	}
	if entrada.VersionEsperada == nil {
		return entradaAsignacionHTTP{}, errContenidoAsignacionInvalido
	}
	resultado := entradaAsignacionHTTP{
		expedienteRef:      entrada.ExpedienteRef,
		versionEsperada:    *entrada.VersionEsperada,
		claveIdempotencia:  entrada.ClaveIdempotencia,
		unidadRef:          entrada.UnidadRef,
		responsableRef:     entrada.ResponsableRef,
		motivoReasignacion: domain.ClaveCatalogo(entrada.MotivoReasignacion),
		observaciones:      entrada.Observaciones,
	}
	if !resultado.valida(operacion) {
		return entradaAsignacionHTTP{}, errContenidoAsignacionInvalido
	}
	return resultado, nil
}

func decodificarAsignacion(
	w http.ResponseWriter,
	r *http.Request,
	operacion ports.TipoOperacionAsignacion,
	destino any,
) error {
	contenido, err := io.ReadAll(http.MaxBytesReader(
		w,
		r.Body,
		MaximoCuerpoAsignacionBytes+1,
	))
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return errCuerpoAsignacionDemasiadoGrande
		}
		return errEntradaAsignacionInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return errEntradaAsignacionInvalida
	}
	if len(contenido) > MaximoCuerpoAsignacionBytes {
		return errCuerpoAsignacionDemasiadoGrande
	}
	if err := validarJSONAsignacionCerrado(contenido, operacion); err != nil {
		return err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaAsignacionInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaAsignacionInvalida
	}
	return nil
}

func validarJSONAsignacionCerrado(
	contenido []byte,
	operacion ports.TipoOperacionAsignacion,
) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('{') {
		return errEntradaAsignacionInvalida
	}
	vistas := make(map[string]struct{}, 7)
	tokens := 1
	for decodificador.More() {
		tokens++
		claveToken, err := decodificador.Token()
		clave, correcta := claveToken.(string)
		if err != nil || !correcta || tokens > maximosTokensAsignacionJSON ||
			!claveAsignacionJSONExacta(clave, operacion) {
			return errEntradaAsignacionInvalida
		}
		if _, repetida := vistas[clave]; repetida {
			return errEntradaAsignacionInvalida
		}
		vistas[clave] = struct{}{}
		tokens++
		valor, err := decodificador.Token()
		if err != nil || valor == nil || tokens > maximosTokensAsignacionJSON {
			return errEntradaAsignacionInvalida
		}
		if _, compuesto := valor.(json.Delim); compuesto {
			return errEntradaAsignacionInvalida
		}
	}
	cierre, err := decodificador.Token()
	if err != nil || cierre != json.Delim('}') {
		return errEntradaAsignacionInvalida
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaAsignacionInvalida
	}
	return nil
}

func claveAsignacionJSONExacta(
	clave string,
	operacion ports.TipoOperacionAsignacion,
) bool {
	switch clave {
	case "expediente_ref", "version_esperada", "clave_idempotencia",
		"unidad_ref", "responsable_ref":
		return true
	case "motivo_reasignacion_clave", "observaciones":
		return operacion == ports.OperacionRegistrarReasignacion
	default:
		return false
	}
}

func textoAsignacionHTTPValido(
	valor string,
	maximo int,
	permiteVacio bool,
) bool {
	if valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) ||
		utf8.RuneCountInString(valor) > maximo ||
		(!permiteVacio && valor == "") {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) && caracter != '\n' && caracter != '\t' {
			return false
		}
	}
	return true
}

func reciboAsignacionSeguro(
	r ports.ReciboAsignacion,
	c ContextoCanalAsignacion,
	e entradaAsignacionHTTP,
	operacion ports.TipoOperacionAsignacion,
) bool {
	return r.Operacion == operacion && r.OrganizacionRef == c.OrganizacionRef &&
		r.ExpedienteRef == e.expedienteRef &&
		r.VersionAnterior == e.versionEsperada &&
		ports.VersionOperacionAnalisisConIncrementoValida(r.VersionAnterior) &&
		r.VersionResultante == r.VersionAnterior+1 &&
		r.VersionResultante <= ports.MaximoEnteroSeguroOperacionAnalisis &&
		r.UnidadRef == e.unidadRef && r.ResponsableRef == e.responsableRef &&
		domain.ReferenciaOpacaValida(r.ReciboRef) &&
		domain.ReferenciaOpacaValida(r.NotificacionRef) &&
		domain.ReferenciaOpacaValida(r.BandejaRef) &&
		domain.ReferenciaOpacaValida(r.AuditoriaRef) &&
		domain.ReferenciaOpacaValida(r.EventoRef) &&
		domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) &&
		ports.SelloHMACSHA256Valido(r.AmbitoIdempotenciaHMAC) &&
		ports.SelloHMACSHA256Valido(r.HuellaPeticionHMAC) &&
		domain.InstanteUTCCanonico(r.ConfirmadaEn)
}

type envoltorioReciboAsignacion struct {
	Data reciboAsignacionJSON `json:"data"`
}

type reciboAsignacionJSON struct {
	Esquema           string `json:"esquema"`
	Operacion         string `json:"operacion"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionResultante uint64 `json:"version_resultante"`
	ReciboRef         string `json:"recibo_ref"`
	ConfirmadaEn      string `json:"confirmada_en"`
}

func responderExitoAsignacion(w http.ResponseWriter, r ports.ReciboAsignacion) {
	responderJSONCobertura(
		w,
		http.StatusCreated,
		envoltorioReciboAsignacion{Data: reciboAsignacionJSON{
			Esquema:           esquemaReciboAsignacion,
			Operacion:         string(r.Operacion),
			ExpedienteRef:     r.ExpedienteRef,
			VersionResultante: r.VersionResultante,
			ReciboRef:         r.ReciboRef,
			ConfirmadaEn:      r.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
		}},
	)
}

var (
	errorPeticionAsignacionNoValida = nuevoErrorAsignacion(
		http.StatusBadRequest, "peticion_no_valida",
	)
	errorPeticionAsignacionNoPermitida = nuevoErrorAsignacion(
		http.StatusBadRequest, "peticion_no_permitida",
	)
	errorRecursoAsignacionNoEncontrado = nuevoErrorAsignacion(
		http.StatusNotFound, "recurso_no_encontrado",
	)
	errorMetodoAsignacionNoPermitido = nuevoErrorAsignacion(
		http.StatusMethodNotAllowed, "metodo_no_permitido",
	)
	errorTipoAsignacionNoAdmitido = nuevoErrorAsignacion(
		http.StatusUnsupportedMediaType, "tipo_contenido_no_admitido",
	)
	errorRepresentacionAsignacionNoAceptable = nuevoErrorAsignacion(
		http.StatusNotAcceptable, "representacion_no_aceptable",
	)
	errorCuerpoAsignacionDemasiadoGrande = nuevoErrorAsignacion(
		http.StatusRequestEntityTooLarge, "peticion_demasiado_grande",
	)
	errorContenidoAsignacionInvalido = nuevoErrorAsignacion(
		http.StatusUnprocessableEntity, "contenido_no_valido",
	)
	errorAutenticacionAsignacionRequerida = nuevoErrorAsignacion(
		http.StatusUnauthorized, "autenticacion_requerida",
	)
	errorAccesoAsignacionDenegado = nuevoErrorAsignacion(
		http.StatusForbidden, "acceso_denegado",
	)
	errorConflictoAsignacion = nuevoErrorAsignacion(
		http.StatusConflict, "conflicto",
	)
	errorResultadoAsignacionNoConfiable = nuevoErrorAsignacion(
		http.StatusBadGateway, "resultado_no_confiable",
	)
	errorServicioAsignacionNoDisponible = nuevoErrorAsignacion(
		http.StatusServiceUnavailable, "servicio_no_disponible",
	)
	errorCancelacionAsignacion = nuevoErrorAsignacion(
		http.StatusRequestTimeout, "peticion_cancelada",
	)
	errorPlazoAsignacion = nuevoErrorAsignacion(
		http.StatusGatewayTimeout, "plazo_agotado",
	)
)

func nuevoErrorAsignacion(estado int, codigo string) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado:    estado,
		codigo:    codigo,
		claveI18n: "api.contratacion_temporal.asignacion.error." + codigo,
	}
}

func errorEntradaAsignacion(err error) errorPublicoCobertura {
	if errors.Is(err, errCuerpoAsignacionDemasiadoGrande) {
		return errorCuerpoAsignacionDemasiadoGrande
	}
	if errors.Is(err, errContenidoAsignacionInvalido) {
		return errorContenidoAsignacionInvalido
	}
	return errorPeticionAsignacionNoValida
}

func clasificarErrorAsignacionHTTP(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionAsignacion
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoAsignacion
	case errors.Is(err, ErrContextoCanalAusente),
		errors.Is(err, ErrContextoCanalCaducado):
		return errorAutenticacionAsignacionRequerida
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada),
		errors.Is(err, application.ErrAsignacionDenegada),
		errors.Is(err, ports.ErrAutorizacionDenegada):
		return errorAccesoAsignacionDenegado
	case errors.Is(err, ports.ErrClaveIdempotenciaUsada),
		errors.Is(err, domain.ErrVersionEnConflicto):
		return errorConflictoAsignacion
	case errors.Is(err, application.ErrSolicitudAsignacionInvalida),
		errors.Is(err, domain.ErrDatoInvalido),
		errors.Is(err, ports.ErrPreparacionAsignacionInvalida):
		return errorContenidoAsignacionInvalido
	case errors.Is(err, application.ErrResultadoAsignacionNoConfiable),
		errors.Is(err, ports.ErrResultadoAsignacionNoConfiable),
		errors.Is(err, ports.ErrOrdenAsignacionInvalida):
		return errorResultadoAsignacionNoConfiable
	case errors.Is(err, ErrContextoCanalNoDisponible),
		errors.Is(err, application.ErrServicioAsignacionInvalido),
		errors.Is(err, ports.ErrPersistenciaAsignacionNoDisponible),
		errors.Is(err, ports.ErrDestinoAsignacionNoDisponible),
		errors.Is(err, ports.ErrPoliticaAsignacionNoDisponible):
		return errorServicioAsignacionNoDisponible
	default:
		return errorServicioAsignacionNoDisponible
	}
}

func responderErrorAsignacion(
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
