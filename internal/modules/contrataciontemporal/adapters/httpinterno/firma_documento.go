package httpinterno

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Rutas del registro de firmas de prueba de los borradores. La firma llega
// ya hecha por AutoFirma en el equipo de la persona; el servidor la verifica
// y la registra. Ninguna respuesta presenta la firma como eficaz: todas
// llevan firma_eficaz=false hasta el portafirmas corporativo.
const (
	RutaFirmaDocumento          = "/api/vec/contratacion-temporal/firmas-documento"
	RutaConsultaFirmaDocumento  = "/api/vec/contratacion-temporal/firmas-documento/consultas"
	maximoCuerpoFirmaDocumento  = 2 << 20
	maximoCuerpoConsultaFirmas  = 4 << 10
	esquemaReciboFirmaDocumento = "vec.contratacion-temporal.recibo-firma-documento.v1"
	esquemaEstadoFirmaDocumento = "vec.contratacion-temporal.estado-firmas-documento.v1"
)

// AutoridadCanalFirmaDocumento resuelve la organización del canal
// autenticado. Nunca se toma del cuerpo de la petición.
type AutoridadCanalFirmaDocumento interface {
	ResolverOrganizacionFirmaDocumento(context.Context) (string, error)
}

// ServicioFirmaDocumentoHTTP es el caso de uso que usa el manejador.
type ServicioFirmaDocumentoHTTP interface {
	Firmar(context.Context, application.SolicitudFirmaDocumento) (application.ResultadoFirmaDocumento, error)
	Consultar(ctx context.Context, organizacionRef, expedienteRef string) (application.EstadoFirmasExpediente, error)
	VerificacionDisponible() bool
}

type manejadorFirmaDocumento struct {
	autoridad AutoridadCanalFirmaDocumento
	servicio  ServicioFirmaDocumentoHTTP
}

// NuevoManejadorFirmaDocumento atiende las dos rutas exactas.
func NuevoManejadorFirmaDocumento(a AutoridadCanalFirmaDocumento, s ServicioFirmaDocumentoHTTP) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(s) {
		return nil, errors.New("contratacion temporal http: firma de documentos no disponible")
	}
	return &manejadorFirmaDocumento{autoridad: a, servicio: s}, nil
}

type errorFirmaDocumentoJSON struct {
	Codigo         string `json:"codigo"`
	ClaveI18n      string `json:"clave_i18n"`
	Motivo         string `json:"motivo,omitempty"`
	CorrelacionRef string `json:"correlacion_ref"`
}

func responderErrorFirmaDocumento(w http.ResponseWriter, r *http.Request, estado int, codigo, motivo string, causas ...error) {
	responderJSONCobertura(w, r, estado, map[string]errorFirmaDocumentoJSON{"error": {
		Codigo: codigo, ClaveI18n: "api.contratacion_temporal.firma_documento.error." + codigo,
		Motivo: motivo, CorrelacionRef: nuevaCorrelacionCobertura(),
	}}, causas...)
}

func (h *manejadorFirmaDocumento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		(r.URL.Path != RutaFirmaDocumento && r.URL.Path != RutaConsultaFirmaDocumento) {
		responderErrorFirmaDocumento(w, r, http.StatusNotFound, "recurso_no_encontrado", "")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorFirmaDocumento(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido", "")
		return
	}
	if !tipoContenidoJSON(r.Header) || cabeceraCoberturaProhibida(r.Header) {
		responderErrorFirmaDocumento(w, r, http.StatusBadRequest, "peticion_no_permitida", "")
		return
	}
	limite := int64(maximoCuerpoFirmaDocumento)
	if r.URL.Path == RutaConsultaFirmaDocumento {
		limite = maximoCuerpoConsultaFirmas
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limite+1))
	if err != nil || len(contenido) == 0 || int64(len(contenido)) > limite {
		responderErrorFirmaDocumento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", "")
		return
	}
	organizacion, err := h.autoridad.ResolverOrganizacionFirmaDocumento(r.Context())
	if err != nil || !domain.ReferenciaOpacaValida(organizacion) {
		responderErrorFirmaDocumento(w, r, http.StatusForbidden, "acceso_denegado", "")
		return
	}
	if r.URL.Path == RutaConsultaFirmaDocumento {
		h.consultar(w, r, organizacion, contenido)
		return
	}
	h.firmar(w, r, organizacion, contenido)
}

func decodificarCerradoFirma(contenido []byte, v any) bool {
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && dec.Decode(&struct{}{}) == io.EOF
}

func (h *manejadorFirmaDocumento) consultar(w http.ResponseWriter, r *http.Request, organizacion string, contenido []byte) {
	var in struct {
		ExpedienteRef string `json:"expediente_ref"`
	}
	if !decodificarCerradoFirma(contenido, &in) || !domain.ReferenciaOpacaValida(in.ExpedienteRef) {
		responderErrorFirmaDocumento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", "")
		return
	}
	estado, err := h.servicio.Consultar(r.Context(), organizacion, in.ExpedienteRef)
	if err != nil {
		h.responderError(w, r, err)
		return
	}
	responderJSONCobertura(w, r, http.StatusOK, map[string]any{"data": vistaEstadoFirmas(estado, h.servicio.VerificacionDisponible())})
}

type pasoEstadoFirmaJSON struct {
	Orden            int    `json:"orden"`
	Cargo            string `json:"cargo"`
	Accion           string `json:"accion"`
	Devolucion       string `json:"devolucion"`
	Estado           string `json:"estado"`
	MotivoDevolucion string `json:"motivo_devolucion,omitempty"`
	ReciboRef        string `json:"recibo_ref,omitempty"`
	RegistradaEn     string `json:"registrada_en,omitempty"`
}

type documentoEstadoFirmaJSON struct {
	Documento       string                `json:"documento"`
	Etiqueta        string                `json:"etiqueta"`
	PasoPendiente   int                   `json:"paso_pendiente"`
	Completo        bool                  `json:"completo"`
	UltimaSecuencia int                   `json:"ultima_secuencia"`
	OriginalSHA256  string                `json:"original_esperado_sha256,omitempty"`
	Pasos           []pasoEstadoFirmaJSON `json:"pasos"`
}

func vistaEstadoFirmas(e application.EstadoFirmasExpediente, verificacion bool) map[string]any {
	documentos := make([]documentoEstadoFirmaJSON, 0, len(e.Documentos))
	for _, d := range e.Documentos {
		c, _ := e.Circuito.Documento(d.Documento)
		salida := documentoEstadoFirmaJSON{Documento: d.Documento, Etiqueta: c.Etiqueta, PasoPendiente: d.PasoPendiente,
			Completo: d.Completo, UltimaSecuencia: d.UltimaSecuencia, OriginalSHA256: d.OriginalEsperadoHuella}
		for i, p := range d.Pasos {
			paso := pasoEstadoFirmaJSON{Orden: p.Orden, Estado: string(p.Estado), MotivoDevolucion: p.MotivoDevolucion, ReciboRef: p.ReciboRef}
			if i < len(c.Pasos) {
				paso.Cargo, paso.Accion, paso.Devolucion = c.Pasos[i].Cargo, c.Pasos[i].Accion, string(c.Pasos[i].Devolucion)
			}
			if !p.RegistradaEn.IsZero() {
				paso.RegistradaEn = p.RegistradaEn.UTC().Format(time.RFC3339Nano)
			}
			salida.Pasos = append(salida.Pasos, paso)
		}
		documentos = append(documentos, salida)
	}
	return map[string]any{
		"esquema": esquemaEstadoFirmaDocumento, "catalogo_ref": e.Circuito.CatalogoRef, "huella_sha256": e.Circuito.HuellaCatalogo,
		"ejemplo": e.Circuito.Ejemplo, "firma_eficaz": false, "verificacion_disponible": verificacion, "documentos": documentos,
	}
}

func (h *manejadorFirmaDocumento) firmar(w http.ResponseWriter, r *http.Request, organizacion string, contenido []byte) {
	var in struct {
		ExpedienteRef     string `json:"expediente_ref"`
		VersionExpediente uint64 `json:"version_expediente"`
		Documento         string `json:"documento"`
		PasoOrden         int    `json:"paso_orden"`
		Resultado         string `json:"resultado"`
		MotivoDevolucion  string `json:"motivo_devolucion"`
		OriginalBase64    string `json:"original_base64"`
		FirmadoBase64     string `json:"firmado_base64"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}
	if !decodificarCerradoFirma(contenido, &in) {
		responderErrorFirmaDocumento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", "")
		return
	}
	original, errOriginal := base64.StdEncoding.Strict().DecodeString(in.OriginalBase64)
	firmado, errFirmado := base64.StdEncoding.Strict().DecodeString(in.FirmadoBase64)
	if errOriginal != nil || errFirmado != nil {
		responderErrorFirmaDocumento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", "")
		return
	}
	resultado, err := h.servicio.Firmar(r.Context(), application.SolicitudFirmaDocumento{
		OrganizacionRef: organizacion, ExpedienteRef: in.ExpedienteRef, VersionExpediente: in.VersionExpediente,
		Documento: in.Documento, PasoOrden: in.PasoOrden, Resultado: domain.ResultadoFirmaDocumento(in.Resultado),
		MotivoDevolucion: in.MotivoDevolucion, Original: original, Firmado: firmado, ClaveIdempotencia: in.ClaveIdempotencia,
	})
	clear(original)
	clear(firmado)
	if err != nil {
		h.responderError(w, r, err)
		return
	}
	rec, m := resultado.Recibo, resultado.Material
	salida := map[string]any{
		"esquema": esquemaReciboFirmaDocumento, "firma_ref": rec.FirmaRef, "recibo_ref": rec.ReciboRef,
		"expediente_ref": m.ExpedienteRef, "expediente_version": rec.ExpedienteVersion, "documento": m.Documento,
		"paso_orden": m.PasoOrden, "paso_ref": m.PasoRef, "catalogo_ref": m.CatalogoRef, "catalogo_huella_sha256": m.CatalogoHuella,
		"secuencia": rec.Secuencia, "resultado": string(rec.Resultado), "perfil_ref": rec.PerfilRef,
		"registrada_en": rec.RegistradaEn.UTC().Format(time.RFC3339Nano), "ya_registrada": rec.YaRegistrada,
		// Verificada por el validador, pero sin eficacia administrativa.
		"firma_verificada": rec.Resultado == domain.ResultadoFirmaFirmado, "firma_eficaz": false,
	}
	if rec.Resultado == domain.ResultadoFirmaFirmado {
		salida["verificacion"] = map[string]string{
			"estado": "valida", "motivo": string(resultado.MotivoVerificacion), "politica": m.PoliticaVerificacion,
			"certificado_sha256": m.CertificadoHuella, "revocacion": m.RevocacionEstado, "sello_tiempo": m.SelloTiempoEstado,
			"original_sha256": m.OriginalHuella, "firmado_sha256": m.FirmadoHuella,
		}
	} else {
		salida["motivo_devolucion"] = m.MotivoDevolucion
	}
	estado := http.StatusCreated
	if rec.YaRegistrada {
		estado = http.StatusOK
	}
	responderJSONCobertura(w, r, estado, map[string]any{"data": salida})
}

func (h *manejadorFirmaDocumento) responderError(w http.ResponseWriter, r *http.Request, err error) {
	var rechazo application.DictamenRechazado
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		responderErrorFirmaDocumento(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", "", err)
	case errors.Is(err, application.ErrVerificacionFirmaApagada):
		responderErrorFirmaDocumento(w, r, http.StatusServiceUnavailable, "verificacion_no_disponible", "")
	case errors.As(err, &rechazo):
		responderErrorFirmaDocumento(w, r, http.StatusUnprocessableEntity, "firma_no_verificada", string(rechazo.Motivo))
	case errors.Is(err, application.ErrPasoFirmaNoPendiente):
		responderErrorFirmaDocumento(w, r, http.StatusConflict, "paso_no_pendiente", "")
	case errors.Is(err, ports.ErrCadenaFirmaDocumentoRota):
		responderErrorFirmaDocumento(w, r, http.StatusConflict, "cadena_rota", "")
	case errors.Is(err, ports.ErrFirmaDocumentoEnConflicto), errors.Is(err, ports.ErrClaveFirmaDocumentoUsada):
		responderErrorFirmaDocumento(w, r, http.StatusConflict, "conflicto", "")
	case errors.Is(err, ports.ErrFirmaDocumentoDenegada), errors.Is(err, ports.ErrAutorizacionDenegada):
		responderErrorFirmaDocumento(w, r, http.StatusForbidden, "acceso_denegado", "")
	case errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida):
		responderErrorFirmaDocumento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", "")
	case errors.Is(err, application.ErrCircuitoFirmaNoDisponible), errors.Is(err, domain.ErrHistoriaFirmaIncoherente),
		errors.Is(err, domain.ErrCircuitoFirmaIncoherente):
		responderErrorFirmaDocumento(w, r, http.StatusServiceUnavailable, "circuito_no_disponible", "", err)
	case errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido):
		responderErrorFirmaDocumento(w, r, http.StatusBadGateway, "resultado_no_confiable", "", err)
	default:
		responderErrorFirmaDocumento(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", "", err)
	}
}
