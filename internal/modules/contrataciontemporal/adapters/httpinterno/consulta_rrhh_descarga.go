package httpinterno

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// errorAccesoConsultaRRHHDenegado solo lo usa la descarga: la consulta del
// detalle ya se autorizó y lo que se deniega es la entrega del archivo.
var errorAccesoConsultaRRHHDenegado = nuevoErrorConsultaRRHH(http.StatusForbidden, "acceso_denegado")

// ConfigurarDescargaAuditadaConsultaDetalleRRHH hace que cada descarga de
// borrador consuma su propia decisión (contratacion_temporal.borrador_rrhh.descargar)
// antes de escribir el archivo, y que los intentos fallidos queden en la
// auditoría común. Sin él, el manejador conserva la conducta anterior.
func ConfigurarDescargaAuditadaConsultaDetalleRRHH(handler http.Handler, registrador ports.RegistradorDescargaBorradorRRHH) (http.Handler, error) {
	h, ok := handler.(*manejadorConsultaDetalleRRHH)
	if !ok || h == nil || dependenciaConsultaRRHHNula(registrador) {
		return nil, ErrManejadorConsultaRRHHInvalido
	}
	copia := *h
	copia.descargas = registrador
	return &copia, nil
}

// fallarDescargaRRHH registra un intento fallido de descarga (si la descarga
// está auditada) y responde con el error público de siempre. Si el intento no
// se pudo registrar, responde 503 en lugar del error original.
func (h *manejadorConsultaDetalleRRHH) fallarDescargaRRHH(w http.ResponseWriter, r *http.Request, expedienteRef string,
	causa error, problema errorPublicoConsultaRRHH,
) {
	if h.descargas != nil && expedienteRef != "" {
		motivo := causa
		if motivo == nil {
			// Sin causa interna (p. ej. archivo con firma de formato errónea):
			// cuenta como error técnico, nunca como denegación.
			motivo = ports.ErrDescargaBorradorRRHHNoDisponible
		}
		// ErrDescargaBorradorRRHHSinAnotar: no había nada que anotar (sin
		// registrador o sin actor); se responde con el error original.
		registrado := h.descargas.RegistrarFalloDescarga(r.Context(), expedienteRef, motivo)
		var auditado ports.FalloLecturaAuditado
		if !errors.As(registrado, &auditado) && !errors.Is(registrado, ports.ErrDescargaBorradorRRHHSinAnotar) {
			responderErrorConsultaRRHH(w, r, registrado, errorServicioConsultaRRHHNoDisponible)
			return
		}
		adjuntarAcuseAuditoriaLectura(w, registrado)
	}
	responderErrorConsultaRRHH(w, r, causa, problema)
}

// autorizarEntregaBorradorRRHH consume la decisión de descarga ligada al
// archivo exacto. Devuelve false si ya respondió con un error.
func (h *manejadorConsultaDetalleRRHH) autorizarEntregaBorradorRRHH(w http.ResponseWriter, r *http.Request,
	detalle ports.DetalleExpedienteRRHH, borrador representacionBorradorRRHH, formato string, contenido []byte,
) bool {
	if h.descargas == nil {
		return true
	}
	suma := sha256.Sum256(contenido)
	solicitud := ports.SolicitudDescargaBorradorRRHH{
		ExpedienteRef: detalle.Resumen.ExpedienteRef, VersionExpediente: detalle.Resumen.Version,
		Tipo: borrador.tipo, Formato: formato, DocumentoSHA256: hex.EncodeToString(suma[:]),
		TamanoBytes: len(contenido), ConsultaHuellaSHA256: detalle.Lectura.ConsultaHuellaSHA256(),
	}
	recibo, err := h.descargas.RegistrarDescarga(r.Context(), solicitud)
	if err != nil {
		adjuntarAcuseAuditoriaLectura(w, err)
		switch {
		case errors.Is(err, ports.ErrAutorizacionDenegada):
			responderErrorConsultaRRHH(w, r, err, errorAccesoConsultaRRHHDenegado)
		case errors.Is(err, ports.ErrDescargaBorradorRRHHVersionAusente):
			responderErrorConsultaRRHH(w, r, err, nuevoErrorConsultaRRHH(http.StatusConflict, "documento_no_disponible"))
		default:
			responderErrorConsultaRRHH(w, r, err, clasificarErrorDescargaRRHH(err))
		}
		return false
	}
	w.Header().Set("X-Audit-Ref", recibo.AuditoriaRef)
	w.Header().Set("X-VEC-Documento-SHA256", solicitud.DocumentoSHA256)
	return true
}

func clasificarErrorDescargaRRHH(err error) errorPublicoConsultaRRHH {
	if problema := clasificarErrorConsultaRRHH(err); problema != errorInternoConsultaRRHH {
		return problema
	}
	return errorServicioConsultaRRHHNoDisponible
}
