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
	MaximoCuerpoFiscalizacionBytes = 16 * 1024
	maximosTokensFiscalizacionJSON = 12
	esquemaReciboFiscalizacion     = "vec.contratacion-temporal.recibo-fiscalizacion.v1"
	operacionFiscalizacion         = "registrar_resultado"
)

var (
	errEntradaFiscalizacionInvalida = errors.New(
		"contratacion temporal http: entrada de fiscalizacion invalida",
	)
	errContenidoFiscalizacionInvalido = errors.New(
		"contratacion temporal http: contenido de fiscalizacion invalido",
	)
	errCuerpoFiscalizacionDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de fiscalizacion demasiado grande",
	)
)

type fiscalizacionEntradaJSON struct {
	ExpedienteRef     string  `json:"expediente_ref"`
	VersionEsperada   *uint64 `json:"version_esperada"`
	ClaveIdempotencia string  `json:"clave_idempotencia"`
	Resultado         string  `json:"resultado"`
	Observaciones     string  `json:"observaciones"`
}

type entradaFiscalizacionHTTP struct {
	expedienteRef     string
	versionEsperada   uint64
	claveIdempotencia string
	resultado         domain.ResultadoFiscalizacion
	observaciones     string
}

func (e entradaFiscalizacionHTTP) valida() bool {
	return domain.ReferenciaOpacaValida(e.expedienteRef) &&
		ports.VersionOperacionAnalisisConIncrementoValida(e.versionEsperada) &&
		ports.ClaveIdempotenciaValida(e.claveIdempotencia) &&
		resultadoFiscalizacionHTTPValido(string(e.resultado)) &&
		ports.ValidarResultadoFiscalizacion(e.resultado, e.observaciones) == nil
}

func (e entradaFiscalizacionHTTP) solicitud(
	c ContextoCanalFiscalizacion,
) application.SolicitudRegistrarResultadoFiscalizacion {
	return application.SolicitudRegistrarResultadoFiscalizacion{
		AutenticacionRef:  c.AutenticacionRef,
		SesionRef:         c.SesionRef,
		PerfilRef:         c.PerfilRef,
		OrganizacionRef:   c.OrganizacionRef,
		ExpedienteRef:     e.expedienteRef,
		VersionEsperada:   e.versionEsperada,
		ClaveIdempotencia: e.claveIdempotencia,
		Resultado:         e.resultado,
		Observaciones:     e.observaciones,
	}
}

func validarMetadatosFiscalizacion(r *http.Request) *errorPublicoCobertura {
	if r != nil && r.ContentLength > MaximoCuerpoFiscalizacionBytes {
		problema := errorCuerpoFiscalizacionDemasiadoGrande
		return &problema
	}
	if r == nil || r.ContentLength < -1 || r.ContentLength == 0 ||
		r.Body == nil || r.Body == http.NoBody || len(r.Trailer) != 0 ||
		!transferenciaFiscalizacionPermitida(r) {
		problema := errorPeticionFiscalizacionNoValida
		return &problema
	}
	if cabeceraCoberturaProhibida(r.Header) {
		problema := errorPeticionFiscalizacionNoPermitida
		return &problema
	}
	if !tipoContenidoJSON(r.Header) {
		problema := errorTipoFiscalizacionNoAdmitido
		return &problema
	}
	if !acceptCompatibleJSON(r.Header) {
		problema := errorRepresentacionFiscalizacionNoAceptable
		return &problema
	}
	return nil
}

func transferenciaFiscalizacionPermitida(r *http.Request) bool {
	if r == nil || len(r.TransferEncoding) == 0 {
		return r != nil
	}
	return len(r.TransferEncoding) == 1 && r.ContentLength <= 0 &&
		strings.EqualFold(r.TransferEncoding[0], "chunked")
}

func fiscalizacionDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (entradaFiscalizacionHTTP, error) {
	contenido, err := io.ReadAll(http.MaxBytesReader(
		w,
		r.Body,
		MaximoCuerpoFiscalizacionBytes+1,
	))
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return entradaFiscalizacionHTTP{}, errCuerpoFiscalizacionDemasiadoGrande
		}
		return entradaFiscalizacionHTTP{}, errEntradaFiscalizacionInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return entradaFiscalizacionHTTP{}, errEntradaFiscalizacionInvalida
	}
	if len(contenido) > MaximoCuerpoFiscalizacionBytes {
		return entradaFiscalizacionHTTP{}, errCuerpoFiscalizacionDemasiadoGrande
	}
	if err := validarJSONFiscalizacionCerrado(contenido); err != nil {
		return entradaFiscalizacionHTTP{}, err
	}
	var entrada fiscalizacionEntradaJSON
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil {
		return entradaFiscalizacionHTTP{}, errEntradaFiscalizacionInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF ||
		entrada.VersionEsperada == nil {
		return entradaFiscalizacionHTTP{}, errEntradaFiscalizacionInvalida
	}
	resultado := entradaFiscalizacionHTTP{
		expedienteRef:     entrada.ExpedienteRef,
		versionEsperada:   *entrada.VersionEsperada,
		claveIdempotencia: entrada.ClaveIdempotencia,
		resultado:         domain.ResultadoFiscalizacion(entrada.Resultado),
		observaciones:     entrada.Observaciones,
	}
	if !resultado.valida() {
		return entradaFiscalizacionHTTP{}, errContenidoFiscalizacionInvalido
	}
	return resultado, nil
}

func validarJSONFiscalizacionCerrado(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('{') {
		return errEntradaFiscalizacionInvalida
	}
	vistas := make(map[string]struct{}, 5)
	tokens := 1
	for decodificador.More() {
		tokens++
		claveToken, err := decodificador.Token()
		clave, correcta := claveToken.(string)
		if err != nil || !correcta || tokens > maximosTokensFiscalizacionJSON ||
			!claveFiscalizacionJSONExacta(clave) {
			return errEntradaFiscalizacionInvalida
		}
		if _, repetida := vistas[clave]; repetida {
			return errEntradaFiscalizacionInvalida
		}
		vistas[clave] = struct{}{}
		tokens++
		valor, err := decodificador.Token()
		if err != nil || valor == nil || tokens > maximosTokensFiscalizacionJSON {
			return errEntradaFiscalizacionInvalida
		}
		if _, compuesto := valor.(json.Delim); compuesto {
			return errEntradaFiscalizacionInvalida
		}
	}
	cierre, err := decodificador.Token()
	if err != nil || cierre != json.Delim('}') || len(vistas) != 5 {
		return errEntradaFiscalizacionInvalida
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errEntradaFiscalizacionInvalida
	}
	return nil
}

func claveFiscalizacionJSONExacta(clave string) bool {
	switch clave {
	case "expediente_ref", "version_esperada", "clave_idempotencia",
		"resultado", "observaciones":
		return true
	default:
		return false
	}
}

func resultadoFiscalizacionHTTPValido(resultado string) bool {
	switch resultado {
	case "favorable", "favorable_con_observaciones", "desfavorable":
		return true
	default:
		return false
	}
}

func textoFiscalizacionHTTPValido(valor string, maximo int, permiteVacio bool) bool {
	if valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) || utf8.RuneCountInString(valor) > maximo ||
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

func reciboFiscalizacionVacio(r ports.ReciboFiscalizacion) bool {
	return r.Operacion == "" && r.OrganizacionRef == "" && r.ExpedienteRef == "" &&
		r.VersionAnterior == 0 && r.VersionResultante == 0 && r.Resultado == "" &&
		r.FaseResultante == "" && r.EstadoResultante == "" && r.ReciboRef == "" &&
		r.AuditoriaRef == "" && r.EventoRef == "" && r.ActorRef == "" &&
		r.UnidadRetornoRef == "" && r.ResponsableRetornoRef == "" &&
		r.RegistradaEn.IsZero()
}

func reciboFiscalizacionSeguro(
	r ports.ReciboFiscalizacion,
	c ContextoCanalFiscalizacion,
	e entradaFiscalizacionHTTP,
) bool {
	retornoCompleto := (r.UnidadRetornoRef == "") == (r.ResponsableRetornoRef == "")
	if r.UnidadRetornoRef != "" {
		retornoCompleto = retornoCompleto &&
			domain.ReferenciaOpacaValida(r.UnidadRetornoRef) &&
			domain.ReferenciaOpacaValida(r.ResponsableRetornoRef)
	}
	transicionValida := r.FaseResultante == domain.FaseFiscalizacion &&
		r.EstadoResultante == domain.EstadoEnCurso &&
		r.UnidadRetornoRef == "" && r.ResponsableRetornoRef == ""
	if r.Resultado == domain.FiscalizacionDesfavorable {
		transicionValida = r.FaseResultante == domain.FaseSubsanacionUnidad &&
			r.EstadoResultante == domain.EstadoIncidencia &&
			r.UnidadRetornoRef != "" && r.ResponsableRetornoRef != ""
	}
	return r.Operacion == operacionFiscalizacion &&
		r.OrganizacionRef == c.OrganizacionRef && r.ExpedienteRef == e.expedienteRef &&
		r.VersionAnterior == e.versionEsperada &&
		r.VersionResultante == r.VersionAnterior+1 &&
		r.VersionResultante <= ports.MaximoEnteroSeguroOperacionAnalisis &&
		string(r.Resultado) == string(e.resultado) && r.FaseResultante.Valida() &&
		r.EstadoResultante.Valido() && transicionValida &&
		domain.ReferenciaOpacaValida(r.ReciboRef) &&
		domain.ReferenciaOpacaValida(r.AuditoriaRef) &&
		domain.ReferenciaOpacaValida(r.EventoRef) &&
		domain.ReferenciaOpacaValida(r.ActorRef) && retornoCompleto &&
		domain.InstanteUTCCanonico(r.RegistradaEn)
}

type envoltorioReciboFiscalizacion struct {
	Data reciboFiscalizacionJSON `json:"data"`
}

type reciboFiscalizacionJSON struct {
	Esquema               string `json:"esquema"`
	Operacion             string `json:"operacion"`
	ExpedienteRef         string `json:"expediente_ref"`
	VersionResultante     uint64 `json:"version_resultante"`
	Resultado             string `json:"resultado"`
	FaseResultante        string `json:"fase_resultante"`
	EstadoResultante      string `json:"estado_resultante"`
	ReciboRef             string `json:"recibo_ref"`
	AuditoriaRef          string `json:"auditoria_ref"`
	EventoRef             string `json:"evento_ref"`
	ActorRef              string `json:"actor_ref"`
	UnidadRetornoRef      string `json:"unidad_retorno_ref,omitempty"`
	ResponsableRetornoRef string `json:"responsable_retorno_ref,omitempty"`
	RegistradaEn          string `json:"registrada_en"`
}

func responderExitoFiscalizacion(w http.ResponseWriter, r ports.ReciboFiscalizacion) {
	responderJSONCobertura(
		w,
		http.StatusCreated,
		envoltorioReciboFiscalizacion{Data: reciboFiscalizacionJSON{
			Esquema:               esquemaReciboFiscalizacion,
			Operacion:             operacionFiscalizacion,
			ExpedienteRef:         r.ExpedienteRef,
			VersionResultante:     r.VersionResultante,
			Resultado:             string(r.Resultado),
			FaseResultante:        string(r.FaseResultante),
			EstadoResultante:      string(r.EstadoResultante),
			ReciboRef:             r.ReciboRef,
			AuditoriaRef:          r.AuditoriaRef,
			EventoRef:             r.EventoRef,
			ActorRef:              r.ActorRef,
			UnidadRetornoRef:      r.UnidadRetornoRef,
			ResponsableRetornoRef: r.ResponsableRetornoRef,
			RegistradaEn:          r.RegistradaEn.UTC().Format(time.RFC3339Nano),
		}},
	)
}

var (
	errorPeticionFiscalizacionNoValida = nuevoErrorFiscalizacion(
		http.StatusBadRequest, "peticion_no_valida",
	)
	errorPeticionFiscalizacionNoPermitida = nuevoErrorFiscalizacion(
		http.StatusBadRequest, "peticion_no_permitida",
	)
	errorRecursoFiscalizacionNoEncontrado = nuevoErrorFiscalizacion(
		http.StatusNotFound, "recurso_no_encontrado",
	)
	errorMetodoFiscalizacionNoPermitido = nuevoErrorFiscalizacion(
		http.StatusMethodNotAllowed, "metodo_no_permitido",
	)
	errorTipoFiscalizacionNoAdmitido = nuevoErrorFiscalizacion(
		http.StatusUnsupportedMediaType, "tipo_contenido_no_admitido",
	)
	errorRepresentacionFiscalizacionNoAceptable = nuevoErrorFiscalizacion(
		http.StatusNotAcceptable, "representacion_no_aceptable",
	)
	errorCuerpoFiscalizacionDemasiadoGrande = nuevoErrorFiscalizacion(
		http.StatusRequestEntityTooLarge, "peticion_demasiado_grande",
	)
	errorContenidoFiscalizacionInvalido = nuevoErrorFiscalizacion(
		http.StatusUnprocessableEntity, "contenido_no_valido",
	)
	errorAutenticacionFiscalizacionRequerida = nuevoErrorFiscalizacion(
		http.StatusUnauthorized, "autenticacion_requerida",
	)
	errorAccesoFiscalizacionDenegado = nuevoErrorFiscalizacion(
		http.StatusForbidden, "acceso_denegado",
	)
	errorConflictoFiscalizacion = nuevoErrorFiscalizacion(
		http.StatusConflict, "conflicto",
	)
	errorResultadoFiscalizacionNoConfiable = nuevoErrorFiscalizacion(
		http.StatusBadGateway, "resultado_no_confiable",
	)
	errorServicioFiscalizacionNoDisponible = nuevoErrorFiscalizacion(
		http.StatusServiceUnavailable, "servicio_no_disponible",
	)
	errorCancelacionFiscalizacion = nuevoErrorFiscalizacion(
		http.StatusRequestTimeout, "peticion_cancelada",
	)
	errorPlazoFiscalizacion = nuevoErrorFiscalizacion(
		http.StatusGatewayTimeout, "plazo_agotado",
	)
)

func nuevoErrorFiscalizacion(estado int, codigo string) errorPublicoCobertura {
	return errorPublicoCobertura{
		estado:    estado,
		codigo:    codigo,
		claveI18n: "api.contratacion_temporal.fiscalizacion.error." + codigo,
	}
}

func errorEntradaFiscalizacion(err error) errorPublicoCobertura {
	if errors.Is(err, errCuerpoFiscalizacionDemasiadoGrande) {
		return errorCuerpoFiscalizacionDemasiadoGrande
	}
	if errors.Is(err, errContenidoFiscalizacionInvalido) {
		return errorContenidoFiscalizacionInvalido
	}
	return errorPeticionFiscalizacionNoValida
}

func clasificarErrorFiscalizacionHTTP(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, context.Canceled):
		return errorCancelacionFiscalizacion
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoFiscalizacion
	case errors.Is(err, ErrContextoCanalAusente),
		errors.Is(err, ErrContextoCanalCaducado):
		return errorAutenticacionFiscalizacionRequerida
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada),
		errors.Is(err, ports.ErrAutorizacionDenegada):
		return errorAccesoFiscalizacionDenegado
	case errors.Is(err, ports.ErrClaveIdempotenciaUsada),
		errors.Is(err, domain.ErrVersionEnConflicto),
		errors.Is(err, domain.ErrTransicionInvalida):
		return errorConflictoFiscalizacion
	case errors.Is(err, domain.ErrDatoInvalido):
		return errorContenidoFiscalizacionInvalido
	default:
		return errorServicioFiscalizacionNoDisponible
	}
}

func responderErrorFiscalizacion(w http.ResponseWriter, problema errorPublicoCobertura) {
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
