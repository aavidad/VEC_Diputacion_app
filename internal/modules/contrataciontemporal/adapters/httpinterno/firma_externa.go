package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	RutaRegistroFirmaExterna         = "/api/vec/contratacion-temporal/firmas-documento/registro-externo"
	EsquemaRegistroFirmaExterna      = "vec.contratacion-temporal.registro-firma-externa.v1"
	EsquemaRegistroFirmaExternaV2    = "vec.contratacion-temporal.registro-firma-externa.v2"
	maximoCuerpoRegistroFirmaExterna = ((ports.MaximoDocumentoFirmaBytes+2)/3)*4 + (64 << 10)
)

// La autoridad obtiene la organización del canal interno autenticado. Nunca
// recibe la petición HTTP ni acepta identidad enviada en cabeceras o JSON.
type AutoridadCanalRegistroFirmaExterna interface {
	ResolverOrganizacionFirmaExterna(context.Context) (string, error)
}

type ServicioRegistroFirmaExternaHTTP interface {
	Registrar(context.Context, application.SolicitudFirmaExterna) (application.ResultadoFirmaExterna, error)
}

type entradaRegistroFirmaExterna struct {
	ExpedienteRef                  string `json:"expediente_ref"`
	VersionExpediente              uint64 `json:"version_expediente"`
	Documento                      string `json:"documento"`
	PasoOrden                      int    `json:"paso_orden"`
	OriginalRef                    string `json:"original_ref"`
	OriginalVersion                uint64 `json:"original_version"`
	FirmadoBase64                  string `json:"firmado_base64"`
	ReferenciaPortafirmasDeclarada string `json:"referencia_portafirmas_declarada"`
	FechaPortafirmasDeclarada      string `json:"fecha_portafirmas_declarada"`
	ClaveIdempotencia              string `json:"clave_idempotencia"`
}

type manejadorRegistroFirmaExterna struct {
	autoridad AutoridadCanalRegistroFirmaExterna
	servicio  ServicioRegistroFirmaExternaHTTP
	exigirV2  bool
}

func NuevoManejadorRegistroFirmaExterna(a AutoridadCanalRegistroFirmaExterna, s ServicioRegistroFirmaExternaHTTP) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(s) {
		return nil, ports.ErrRegistroFirmaExternaNoDisponible
	}
	return &manejadorRegistroFirmaExterna{autoridad: a, servicio: s}, nil
}

// El montaje multifirma exige evidencia V2 también en recuperaciones. El
// constructor anterior conserva el transporte V1 para consumidores anteriores.
func NuevoManejadorRegistroFirmaExternaV2(a AutoridadCanalRegistroFirmaExterna, s ServicioRegistroFirmaExternaHTTP) (http.Handler, error) {
	h, err := NuevoManejadorRegistroFirmaExterna(a, s)
	if err != nil {
		return nil, err
	}
	h.(*manejadorRegistroFirmaExterna).exigirV2 = true
	return h, nil
}

func responderErrorRegistroFirmaExterna(w http.ResponseWriter, r *http.Request, estado int, codigo string, causas ...error) {
	responderJSONFirmaNominal(w, r, estado, map[string]any{"error": map[string]string{
		"codigo":          codigo,
		"clave_i18n":      "api.contratacion_temporal.registro_firma_externa.error." + codigo,
		"correlacion_ref": correlacionPeticionFirma(r),
	}}, MaximoRespuestaConsultaRRHHBytes, causas...)
}

func rutaRegistroFirmaExternaExacta(r *http.Request) bool {
	return rutaRegistroFirmaNominalExacta(r, RutaRegistroFirmaExterna)
}

func rutaRegistroFirmaNominalExacta(r *http.Request, ruta string) bool {
	return r != nil && r.URL != nil && r.URL.Path == ruta &&
		r.URL.RawPath == "" && r.URL.RawQuery == "" && !r.URL.ForceQuery &&
		r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil &&
		r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" &&
		r.URL.EscapedPath() == r.URL.Path
}

func (h *manejadorRegistroFirmaExterna) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rutaRegistroFirmaExternaExacta(r) {
		responderErrorRegistroFirmaExterna(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorRegistroFirmaExterna(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	contenido, err := leerCuerpoRegistroFirma(w, r)
	if err != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadRequest, "peticion_no_valida", err)
		return
	}
	defer clear(contenido)
	var entrada entradaRegistroFirmaExterna
	if !camposRegistroFirmaExactos(contenido, true) || !decodificarCerradoFirma(contenido, &entrada) {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	if !entrada.valida() {
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	firmado, err := decodificarPDFRegistroFirma(entrada.FirmadoBase64)
	if err != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", err)
		return
	}
	defer clear(firmado)
	organizacion, err := h.autoridad.ResolverOrganizacionFirmaExterna(r.Context())
	if err != nil || !domain.ReferenciaOpacaValida(organizacion) {
		responderErrorAutoridadFirmaExterna(w, r, err)
		return
	}
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	solicitud := application.SolicitudFirmaExterna{
		OrganizacionRef: organizacion, ExpedienteRef: entrada.ExpedienteRef,
		VersionExpediente: entrada.VersionExpediente, Documento: entrada.Documento,
		PasoOrden: entrada.PasoOrden, OriginalRef: entrada.OriginalRef,
		OriginalVersion: entrada.OriginalVersion, PDFFirmado: firmado,
		ReferenciaPortafirmasDeclarada: entrada.ReferenciaPortafirmasDeclarada,
		FechaPortafirmasDeclarada:      entrada.FechaPortafirmasDeclarada,
		ClaveIdempotencia:              entrada.ClaveIdempotencia,
	}
	huellaFirmado := sha256.Sum256(firmado)
	resultado, err := h.servicio.Registrar(r.Context(), solicitud)
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	if err != nil {
		responderErrorServicioFirmaExterna(w, r, err)
		return
	}
	m := resultado.Material
	if resultado.MaterialMultiple != nil {
		m = resultado.MaterialMultiple.MaterialFirmaExterna
	}
	if (h.exigirV2 && resultado.MaterialMultiple == nil) || !resultadoRegistroFirmaExternaConfiable(resultado, solicitud) || m.FirmadoHuella != hex.EncodeToString(huellaFirmado[:]) {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	rec := resultado.Recibo
	data := vistaRegistroFirmaVerificada(m, rec, resultado.Custodiado, resultado.MotivoVerificacion)
	data["esquema"] = EsquemaRegistroFirmaExterna
	data["procedencia_portafirmas"] = map[string]any{"estado": "declarada_por_rrhh",
		"referencia_declarada": m.ReferenciaPortafirmasDeclarada, "fecha_declarada": m.FechaPortafirmasDeclarada}
	if resultado.MaterialMultiple != nil {
		data["esquema"] = EsquemaRegistroFirmaExternaV2
		completarVistaRegistroFirmaV2(data, resultado.MaterialMultiple)
	}
	estado := http.StatusCreated
	if rec.YaRegistrada {
		estado = http.StatusOK
	}
	responderJSONFirmaNominal(w, r, estado, map[string]any{"data": data}, MaximoRespuestaConsultaRRHHBytes)
}

func (e entradaRegistroFirmaExterna) valida() bool {
	if !domain.ReferenciaOpacaValida(e.ExpedienteRef) ||
		e.VersionExpediente == 0 || e.VersionExpediente > 9007199254740991 ||
		!domain.ClaveDocumentoFirmaValida(e.Documento) || e.PasoOrden < 1 || e.PasoOrden > domain.MaximoPasosCircuitoFirma ||
		!domain.ReferenciaOpacaValida(e.OriginalRef) || e.OriginalVersion == 0 || e.OriginalVersion > 9007199254740991 ||
		!ports.ClaveIdempotenciaFirmaValida(e.ClaveIdempotencia) ||
		len(e.ReferenciaPortafirmasDeclarada) == 0 || len(e.ReferenciaPortafirmasDeclarada) > 256 ||
		strings.TrimSpace(e.ReferenciaPortafirmasDeclarada) != e.ReferenciaPortafirmasDeclarada ||
		len(e.FirmadoBase64) == 0 || len(e.FirmadoBase64) > base64.StdEncoding.EncodedLen(ports.MaximoDocumentoFirmaBytes) {
		return false
	}
	for _, c := range e.ReferenciaPortafirmasDeclarada {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	fecha := e.FechaPortafirmasDeclarada
	if !strings.HasSuffix(fecha, "Z") || len(fecha) > len("2006-01-02T15:04:05.000000Z") {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, fecha)
	return err == nil && t.UTC().Format(time.RFC3339Nano) == fecha
}

func resultadoRegistroFirmaExternaConfiable(r application.ResultadoFirmaExterna, s application.SolicitudFirmaExterna) bool {
	m := r.Material
	if r.MaterialMultiple != nil {
		if r.Material != (ports.MaterialFirmaExterna{}) || r.MaterialMultiple.Validar() != nil ||
			!resultadoRegistroFirmaV2Confiable(r.MaterialMultiple, r.Recibo, r.Custodiado, r.MotivoVerificacion, len(s.PDFFirmado)) {
			return false
		}
		m = r.MaterialMultiple.MaterialFirmaExterna
	} else if m.Validar() != nil {
		return false
	}
	return m.Via == ports.ViaFirmaExternaPortafirmas && m.OrganizacionRef == s.OrganizacionRef && m.ExpedienteRef == s.ExpedienteRef &&
		m.VersionExpediente == s.VersionExpediente && m.Documento == s.Documento && m.PasoOrden == s.PasoOrden &&
		m.OriginalRef == s.OriginalRef && m.OriginalVersion == s.OriginalVersion && m.ClaveIdempotencia == s.ClaveIdempotencia &&
		m.ReferenciaPortafirmasDeclarada == s.ReferenciaPortafirmasDeclarada && m.FechaPortafirmasDeclarada == s.FechaPortafirmasDeclarada &&
		resultadoRegistroFirmaComunConfiable(m, r.Recibo, r.Custodiado, r.MotivoVerificacion)
}

type respuestaErrorRegistroFirma func(http.ResponseWriter, *http.Request, int, string, ...error)

func responderErrorAutoridadFirmaExterna(w http.ResponseWriter, r *http.Request, err error) {
	responderErrorAutoridadRegistroFirma(w, r, err, responderErrorRegistroFirmaExterna)
}

func responderErrorAutoridadRegistroFirma(w http.ResponseWriter, r *http.Request, err error, responder respuestaErrorRegistroFirma) {
	switch {
	case errors.Is(err, ErrContextoCanalAusente), errors.Is(err, ErrContextoCanalCaducado):
		responder(w, r, http.StatusUnauthorized, "autenticacion_requerida", err)
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada), errors.Is(err, ports.ErrAutorizacionDenegada):
		responder(w, r, http.StatusForbidden, "acceso_denegado", err)
	default:
		responder(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
	}
}

func responderErrorServicioFirmaExterna(w http.ResponseWriter, r *http.Request, err error) {
	responderErrorServicioRegistroFirma(w, r, err, responderErrorRegistroFirmaExterna)
}

func responderErrorServicioRegistroFirma(w http.ResponseWriter, r *http.Request, err error, responder respuestaErrorRegistroFirma) {
	var rechazo application.DictamenRechazado
	switch {
	case errors.Is(err, ports.ErrFirmaDocumentoDenegada), errors.Is(err, ports.ErrAutorizacionDenegada),
		errors.Is(err, ports.ErrOriginalFirmaNoAutorizado), errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada),
		errors.Is(err, ports.ErrCustodiaFirmadoDenegada):
		responder(w, r, http.StatusForbidden, "acceso_denegado", err)
	case errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida), errors.Is(err, ports.ErrCustodiaFirmadoInvalida):
		responder(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", err)
	case errors.Is(err, ports.ErrClaveFirmaDocumentoUsada), errors.Is(err, ports.ErrFirmaDocumentoEnConflicto),
		errors.Is(err, ports.ErrCustodiaFirmadoEnConflicto), errors.Is(err, ports.ErrCadenaFirmaDocumentoRota),
		errors.Is(err, ports.ErrAntecedenteFirmaR5NoAcreditado), errors.Is(err, ports.ErrOriginalTrasReparoNoNuevo),
		errors.Is(err, ports.ErrMismaPersonaEnOtroPasoR5), errors.Is(err, application.ErrPasoFirmaNoPendiente):
		responder(w, r, http.StatusConflict, "conflicto", err)
	case errors.As(err, &rechazo):
		if rechazo.Motivo == docports.MotivoValidadorNoDisponible || rechazo.Motivo == docports.MotivoCredencialRechazada {
			responder(w, r, http.StatusServiceUnavailable, "verificacion_no_disponible", err)
		} else {
			responder(w, r, http.StatusUnprocessableEntity, "firma_no_verificada", err)
		}
	case errors.Is(err, application.ErrFirmaNoVerificada):
		responder(w, r, http.StatusUnprocessableEntity, "firma_no_verificada", err)
	case errors.Is(err, application.ErrVerificacionFirmaApagada):
		responder(w, r, http.StatusServiceUnavailable, "verificacion_no_disponible", err)
	default:
		responder(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
	}
}

func leerCuerpoRegistroFirma(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 || r.ContentLength > maximoCuerpoRegistroFirmaExterna ||
		len(r.Trailer) != 0 || !transferenciaAltaPermitida(r.TransferEncoding) ||
		!cabecerasPropuestaFormalizacionPermitidas(r) || !tipoContenidoJSON(r.Header) || !acceptCompatibleJSON(r.Header) {
		return nil, errEntradaConsultaRRHHInvalida
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoRegistroFirmaExterna+1))
	if err != nil {
		clear(contenido)
		return nil, err
	}
	if len(contenido) == 0 || len(contenido) > maximoCuerpoRegistroFirmaExterna || !utf8.Valid(contenido) ||
		(r.ContentLength >= 0 && r.ContentLength != int64(len(contenido))) {
		clear(contenido)
		return nil, errEntradaConsultaRRHHInvalida
	}
	if err := validarJSONPropuestaFormalizacionSinDuplicados(contenido); err != nil {
		clear(contenido)
		return nil, err
	}
	return contenido, nil
}
