package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	RutaRegistroFirmaExterna         = "/api/vec/contratacion-temporal/firmas-documento/registro-externo"
	EsquemaRegistroFirmaExterna      = "vec.contratacion-temporal.registro-firma-externa.v1"
	maximoCuerpoRegistroFirmaExterna = 2 << 20
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
}

func NuevoManejadorRegistroFirmaExterna(a AutoridadCanalRegistroFirmaExterna, s ServicioRegistroFirmaExternaHTTP) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(s) {
		return nil, ports.ErrRegistroFirmaExternaNoDisponible
	}
	return &manejadorRegistroFirmaExterna{autoridad: a, servicio: s}, nil
}

func responderErrorRegistroFirmaExterna(w http.ResponseWriter, r *http.Request, estado int, codigo string, causas ...error) {
	responderJSONCobertura(w, r, estado, map[string]any{"error": map[string]string{
		"codigo":          codigo,
		"clave_i18n":      "api.contratacion_temporal.registro_firma_externa.error." + codigo,
		"correlacion_ref": nuevaCorrelacionCobertura(),
	}}, causas...)
}

func rutaRegistroFirmaExternaExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaRegistroFirmaExterna &&
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
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 || r.ContentLength > maximoCuerpoRegistroFirmaExterna ||
		len(r.Trailer) != 0 || !transferenciaAltaPermitida(r.TransferEncoding) ||
		!cabecerasPropuestaFormalizacionPermitidas(r) || !tipoContenidoJSON(r.Header) || !acceptCompatibleJSON(r.Header) {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoRegistroFirmaExterna+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoRegistroFirmaExterna ||
		(r.ContentLength >= 0 && r.ContentLength != int64(len(contenido))) ||
		validarJSONPropuestaFormalizacionSinDuplicados(contenido) != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	var entrada entradaRegistroFirmaExterna
	if !decodificarCerradoFirma(contenido, &entrada) {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(contenido, &campos) != nil || len(campos) != 10 || !entrada.valida() {
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	firmado, err := base64.StdEncoding.Strict().DecodeString(entrada.FirmadoBase64)
	if err != nil || len(firmado) == 0 || len(firmado) > ports.MaximoDocumentoFirmaBytes ||
		!bytes.HasPrefix(firmado, []byte("%PDF-")) || !bytes.Contains(firmado, []byte("%%EOF")) {
		clear(firmado)
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	organizacion, err := h.autoridad.ResolverOrganizacionFirmaExterna(r.Context())
	if err != nil || !domain.ReferenciaOpacaValida(organizacion) {
		clear(firmado)
		responderErrorAutoridadFirmaExterna(w, r, err)
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
	resultado, err := h.servicio.Registrar(r.Context(), solicitud)
	huellaFirmado := sha256.Sum256(firmado)
	clear(firmado)
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	if err != nil {
		responderErrorServicioFirmaExterna(w, r, err)
		return
	}
	if !resultadoRegistroFirmaExternaConfiable(resultado, solicitud) || resultado.Material.FirmadoHuella != hex.EncodeToString(huellaFirmado[:]) {
		responderErrorRegistroFirmaExterna(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	rec, m, c := resultado.Recibo, resultado.Material, resultado.Custodiado
	data := map[string]any{
		"esquema":    EsquemaRegistroFirmaExterna,
		"recibo_ref": rec.ReciboRef, "firma_ref": rec.FirmaRef, "ya_registrada": rec.YaRegistrada,
		"expediente_ref": m.ExpedienteRef, "version_expediente": rec.ExpedienteVersion,
		"documento": m.Documento, "paso_orden": m.PasoOrden, "paso_ref": m.PasoRef,
		"secuencia": rec.Secuencia, "registrada_en": rec.RegistradaEn.UTC().Format(time.RFC3339Nano),
		"documento_custodiado": documentoCustodiado(organizacion, m.ExpedienteRef, c.Ref, c.Version, c.HuellaSHA256),
		"verificacion_tecnica": map[string]any{"estado": "valida", "motivo": string(resultado.MotivoVerificacion),
			"politica": m.PoliticaVerificacion, "revocacion": m.RevocacionEstado,
			"sello_tiempo": m.SelloTiempoEstado, "original_sha256": m.OriginalHuella, "firmado_sha256": m.FirmadoHuella},
		"procedencia_portafirmas": map[string]any{"estado": "declarada_por_rrhh",
			"referencia_declarada": m.ReferenciaPortafirmasDeclarada,
			"fecha_declarada":      m.FechaPortafirmasDeclarada},
		"firma_eficaz": false,
	}
	estado := http.StatusCreated
	if rec.YaRegistrada {
		estado = http.StatusOK
	}
	responderJSONCobertura(w, r, estado, map[string]any{"data": data})
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
	m, rec, c := r.Material, r.Recibo, r.Custodiado
	return m.Validar() == nil && m.OrganizacionRef == s.OrganizacionRef && m.ExpedienteRef == s.ExpedienteRef &&
		m.VersionExpediente == s.VersionExpediente && m.Documento == s.Documento && m.PasoOrden == s.PasoOrden &&
		m.OriginalRef == s.OriginalRef && m.OriginalVersion == s.OriginalVersion &&
		m.ClaveIdempotencia == s.ClaveIdempotencia &&
		m.ReferenciaPortafirmasDeclarada == s.ReferenciaPortafirmasDeclarada &&
		m.FechaPortafirmasDeclarada == s.FechaPortafirmasDeclarada &&
		c.Ref == m.DocumentoCustodiaRef && c.Version == m.DocumentoCustodiaVersion && c.HuellaSHA256 == m.FirmadoHuella &&
		rec.FirmaRef != "" && rec.ReciboRef != "" && rec.Secuencia == m.Secuencia &&
		rec.Resultado == domain.ResultadoFirmaFirmado && rec.ExpedienteVersion == m.VersionExpediente &&
		rec.DocumentoCustodiaRef == m.DocumentoCustodiaRef && rec.DocumentoCustodiaVersion == m.DocumentoCustodiaVersion &&
		!rec.RegistradaEn.IsZero() && r.MotivoVerificacion == docports.MotivoFirmaVerificada
}

func responderErrorAutoridadFirmaExterna(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrContextoCanalAusente), errors.Is(err, ErrContextoCanalCaducado):
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnauthorized, "autenticacion_requerida", err)
	case errors.Is(err, ErrContextoCanalOrganizacionDenegada), errors.Is(err, ports.ErrAutorizacionDenegada):
		responderErrorRegistroFirmaExterna(w, r, http.StatusForbidden, "acceso_denegado", err)
	default:
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
	}
}

func responderErrorServicioFirmaExterna(w http.ResponseWriter, r *http.Request, err error) {
	var rechazo application.DictamenRechazado
	switch {
	case errors.Is(err, ports.ErrFirmaDocumentoDenegada), errors.Is(err, ports.ErrAutorizacionDenegada),
		errors.Is(err, ports.ErrOriginalFirmaNoAutorizado), errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada),
		errors.Is(err, ports.ErrCustodiaFirmadoDenegada):
		responderErrorRegistroFirmaExterna(w, r, http.StatusForbidden, "acceso_denegado", err)
	case errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida), errors.Is(err, ports.ErrCustodiaFirmadoInvalida):
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", err)
	case errors.Is(err, ports.ErrClaveFirmaDocumentoUsada), errors.Is(err, ports.ErrFirmaDocumentoEnConflicto),
		errors.Is(err, ports.ErrCustodiaFirmadoEnConflicto), errors.Is(err, ports.ErrCadenaFirmaDocumentoRota),
		errors.Is(err, application.ErrPasoFirmaNoPendiente):
		responderErrorRegistroFirmaExterna(w, r, http.StatusConflict, "conflicto", err)
	case errors.As(err, &rechazo):
		if rechazo.Motivo == docports.MotivoValidadorNoDisponible || rechazo.Motivo == docports.MotivoCredencialRechazada {
			responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "verificacion_no_disponible", err)
		} else {
			responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "firma_no_verificada", err)
		}
	case errors.Is(err, application.ErrFirmaNoVerificada):
		responderErrorRegistroFirmaExterna(w, r, http.StatusUnprocessableEntity, "firma_no_verificada", err)
	case errors.Is(err, application.ErrVerificacionFirmaApagada):
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "verificacion_no_disponible", err)
	default:
		responderErrorRegistroFirmaExterna(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
	}
}
