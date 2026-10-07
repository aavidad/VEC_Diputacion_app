package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	RutaRegistroFirmaVec    = "/api/vec/contratacion-temporal/firmas-documento/registro-vec"
	EsquemaRegistroFirmaVec = "vec.contratacion-temporal.registro-firma-vec.v2"
)

// La organización procede del contexto autenticado. La autoridad de aplicación
// resuelve y revalida actor, certificado, perfil y competencia antes del efecto.
type AutoridadCanalRegistroFirmaVec interface {
	ResolverOrganizacionFirmaVec(context.Context) (string, error)
}

type ServicioRegistroFirmaVecHTTP interface {
	Firmar(context.Context, application.SolicitudFirmaVec) (application.ResultadoFirmaVec, error)
}

type entradaRegistroFirmaVec struct {
	ExpedienteRef     string `json:"expediente_ref"`
	VersionExpediente uint64 `json:"version_expediente"`
	Documento         string `json:"documento"`
	PasoOrden         int    `json:"paso_orden"`
	OriginalRef       string `json:"original_ref"`
	OriginalVersion   uint64 `json:"original_version"`
	FirmadoBase64     string `json:"firmado_base64"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
}

type manejadorRegistroFirmaVec struct {
	autoridad AutoridadCanalRegistroFirmaVec
	servicio  ServicioRegistroFirmaVecHTTP
}

func NuevoManejadorRegistroFirmaVec(a AutoridadCanalRegistroFirmaVec, s ServicioRegistroFirmaVecHTTP) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(s) {
		return nil, ports.ErrRegistroFirmaVecNoDisponible
	}
	return &manejadorRegistroFirmaVec{autoridad: a, servicio: s}, nil
}

func responderErrorRegistroFirmaVec(w http.ResponseWriter, r *http.Request, estado int, codigo string, causas ...error) {
	responderJSONFirmaNominal(w, r, estado, map[string]any{"error": map[string]string{
		"codigo": codigo, "clave_i18n": "api.contratacion_temporal.registro_firma_vec.error." + codigo,
		"correlacion_ref": correlacionPeticionFirma(r),
	}}, MaximoRespuestaConsultaRRHHBytes, causas...)
}

func (h *manejadorRegistroFirmaVec) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rutaRegistroFirmaNominalExacta(r, RutaRegistroFirmaVec) {
		responderErrorRegistroFirmaVec(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorRegistroFirmaVec(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaVec(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	contenido, err := leerCuerpoRegistroFirma(w, r)
	if err != nil {
		responderErrorRegistroFirmaVec(w, r, http.StatusBadRequest, "peticion_no_valida", err)
		return
	}
	defer clear(contenido)
	var entrada entradaRegistroFirmaVec
	if !camposRegistroFirmaExactos(contenido, false) || !decodificarCerradoFirma(contenido, &entrada) {
		responderErrorRegistroFirmaVec(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	if !entrada.valida() {
		responderErrorRegistroFirmaVec(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	firmado, err := decodificarPDFRegistroFirma(entrada.FirmadoBase64)
	if err != nil {
		responderErrorRegistroFirmaVec(w, r, http.StatusUnprocessableEntity, "contenido_no_valido", err)
		return
	}
	defer clear(firmado)
	organizacion, err := h.autoridad.ResolverOrganizacionFirmaVec(r.Context())
	if err != nil || !domain.ReferenciaOpacaValida(organizacion) {
		responderErrorAutoridadRegistroFirma(w, r, err, responderErrorRegistroFirmaVec)
		return
	}
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaVec(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	solicitud := application.SolicitudFirmaVec{
		OrganizacionRef: organizacion, ExpedienteRef: entrada.ExpedienteRef, VersionExpediente: entrada.VersionExpediente,
		Documento: entrada.Documento, PasoOrden: entrada.PasoOrden, OriginalRef: entrada.OriginalRef,
		OriginalVersion: entrada.OriginalVersion, PDFFirmado: firmado, ClaveIdempotencia: entrada.ClaveIdempotencia,
	}
	huella := sha256.Sum256(firmado)
	resultado, err := h.servicio.Firmar(r.Context(), solicitud)
	if r.Context().Err() != nil {
		responderErrorRegistroFirmaVec(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", r.Context().Err())
		return
	}
	if err != nil {
		responderErrorServicioRegistroFirma(w, r, err, responderErrorRegistroFirmaVec)
		return
	}
	if !resultadoRegistroFirmaVecConfiable(resultado, solicitud, hex.EncodeToString(huella[:])) {
		responderErrorRegistroFirmaVec(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	data := vistaRegistroFirmaVerificada(resultado.MaterialMultiple.MaterialFirmaExterna, resultado.Recibo, resultado.Custodiado, resultado.MotivoVerificacion)
	data["esquema"] = EsquemaRegistroFirmaVec
	completarVistaRegistroFirmaV2(data, resultado.MaterialMultiple)
	estado := http.StatusCreated
	if resultado.Recibo.YaRegistrada {
		estado = http.StatusOK
	}
	responderJSONFirmaNominal(w, r, estado, map[string]any{"data": data}, MaximoRespuestaConsultaRRHHBytes)
}

func (e entradaRegistroFirmaVec) valida() bool {
	return domain.ReferenciaOpacaValida(e.ExpedienteRef) && e.VersionExpediente > 0 && e.VersionExpediente <= 9007199254740991 &&
		domain.ClaveDocumentoFirmaValida(e.Documento) && e.PasoOrden >= 1 && e.PasoOrden <= domain.MaximoPasosCircuitoFirma &&
		domain.ReferenciaOpacaValida(e.OriginalRef) && e.OriginalVersion > 0 && e.OriginalVersion <= 9007199254740991 &&
		ports.ClaveIdempotenciaFirmaValida(e.ClaveIdempotencia) && len(e.FirmadoBase64) > 0 &&
		len(e.FirmadoBase64) <= base64.StdEncoding.EncodedLen(ports.MaximoDocumentoFirmaBytes)
}

func decodificarPDFRegistroFirma(contenido string) ([]byte, error) {
	firmado, err := base64.StdEncoding.Strict().DecodeString(contenido)
	if err != nil {
		clear(firmado)
		return nil, err
	}
	if len(firmado) == 0 || len(firmado) > ports.MaximoDocumentoFirmaBytes ||
		base64.StdEncoding.EncodeToString(firmado) != contenido || !bytes.HasPrefix(firmado, []byte("%PDF-")) || !bytes.Contains(firmado, []byte("%%EOF")) {
		clear(firmado)
		return nil, errContenidoConsultaRRHHNoValido
	}
	return firmado, nil
}

func resultadoRegistroFirmaVecConfiable(r application.ResultadoFirmaVec, s application.SolicitudFirmaVec, huella string) bool {
	if r.MaterialMultiple == nil || r.Material != (ports.MaterialFirmaVec{}) {
		return false
	}
	m := r.MaterialMultiple
	return m.Validar() == nil && m.Via == ports.ViaFirmaCertificadoVEC &&
		m.OrganizacionRef == s.OrganizacionRef && m.ExpedienteRef == s.ExpedienteRef && m.VersionExpediente == s.VersionExpediente &&
		m.Documento == s.Documento && m.PasoOrden == s.PasoOrden && m.OriginalRef == s.OriginalRef && m.OriginalVersion == s.OriginalVersion &&
		m.ClaveIdempotencia == s.ClaveIdempotencia && m.FirmadoHuella == huella &&
		resultadoRegistroFirmaV2Confiable(m, r.Recibo, r.Custodiado, r.MotivoVerificacion, len(s.PDFFirmado))
}

func resultadoRegistroFirmaV2Confiable(m *ports.MaterialFirmaVerificadaV2, rec ports.ReciboFirmaDocumento, c ports.DocumentoCustodiado, motivo docports.MotivoVerificacionFirma, longitud int) bool {
	if longitud < 1 || longitud > ports.MaximoDocumentoFirmaBytes {
		return false
	}
	huella, err := m.HuellaSHA256()
	return err == nil && rec.SolicitudHuella == huella && m.RevisionLongitud == uint64(longitud) &&
		resultadoRegistroFirmaComunConfiable(m.MaterialFirmaExterna, rec, c, motivo)
}

func resultadoRegistroFirmaComunConfiable(m ports.MaterialFirmaExterna, rec ports.ReciboFirmaDocumento, c ports.DocumentoCustodiado, motivo docports.MotivoVerificacionFirma) bool {
	return c.Ref == m.DocumentoCustodiaRef && c.Version == m.DocumentoCustodiaVersion && c.HuellaSHA256 == m.FirmadoHuella &&
		domain.ReferenciaOpacaValida(rec.FirmaRef) && domain.ReferenciaOpacaValida(rec.ReciboRef) && rec.Secuencia == m.Secuencia &&
		rec.Resultado == domain.ResultadoFirmaFirmado && rec.ExpedienteVersion == m.VersionExpediente &&
		rec.DocumentoCustodiaRef == m.DocumentoCustodiaRef && rec.DocumentoCustodiaVersion == m.DocumentoCustodiaVersion &&
		!rec.RegistradaEn.IsZero() && motivo == docports.MotivoFirmaVerificada
}

func vistaRegistroFirmaVerificada(m ports.MaterialFirmaExterna, rec ports.ReciboFirmaDocumento, c ports.DocumentoCustodiado, motivo docports.MotivoVerificacionFirma) map[string]any {
	return map[string]any{
		"recibo_ref": rec.ReciboRef, "firma_ref": rec.FirmaRef, "ya_registrada": rec.YaRegistrada,
		"expediente_ref": m.ExpedienteRef, "version_expediente": rec.ExpedienteVersion, "documento": m.Documento,
		"paso_orden": m.PasoOrden, "paso_ref": m.PasoRef, "secuencia": rec.Secuencia, "registrada_en": rec.RegistradaEn.UTC().Format(time.RFC3339Nano),
		"documento_custodiado": documentoCustodiado(m.OrganizacionRef, m.ExpedienteRef, c.Ref, c.Version, c.HuellaSHA256),
		"verificacion_tecnica": map[string]any{"estado": "valida", "motivo": string(motivo), "politica": m.PoliticaVerificacion,
			"revocacion": m.RevocacionEstado, "sello_tiempo": m.SelloTiempoEstado, "original_sha256": m.OriginalHuella, "firmado_sha256": m.FirmadoHuella},
		"firma_eficaz": false,
	}
}

func completarVistaRegistroFirmaV2(data map[string]any, m *ports.MaterialFirmaVerificadaV2) {
	huella, _ := m.HuellaSHA256() // La respuesta ya ha superado la validación cerrada.
	data["material_root_sha256"] = huella
	data["revision_pdf"] = map[string]any{"orden_firma": m.OrdenFirmaPDF, "entrada_sha256": m.EntradaDocumentoHuella,
		"revision_sha256": m.RevisionHuellaSHA256, "evidencia_sha256": m.EvidenciaFirmasHuellaSHA256}
}

func camposRegistroFirmaExactos(contenido []byte, externa bool) bool {
	var campos map[string]json.RawMessage
	if json.Unmarshal(contenido, &campos) != nil {
		return false
	}
	esperados := []string{"expediente_ref", "version_expediente", "documento", "paso_orden", "original_ref", "original_version", "firmado_base64", "clave_idempotencia"}
	if externa {
		esperados = append(esperados, "referencia_portafirmas_declarada", "fecha_portafirmas_declarada")
	}
	if len(campos) != len(esperados) {
		return false
	}
	for _, nombre := range esperados {
		if _, ok := campos[nombre]; !ok {
			return false
		}
	}
	return true
}
