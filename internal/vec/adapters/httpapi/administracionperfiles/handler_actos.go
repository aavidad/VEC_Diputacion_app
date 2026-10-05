package administracionperfiles

import (
	"errors"
	"mime"
	"net/http"
	"strings"
	"vec-diputacion-granada/internal/vec/domain"
)

func (h *Handler) post(w http.ResponseWriter, r *http.Request, s SesionConfiable) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "escribir", "")
		return
	}
	p := r.URL.Path
	if r.URL.RawQuery != "" {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "escribir", "")
		return
	}
	if p == RutaGobiernoPlanFirma {
		h.postGobiernoPlanFirma(w, r, s)
		return
	}
	// Con sólo la autoridad del lote montada, cualquier otra escritura no existe.
	if h.lotes != nil && h.actos == nil && p != PrefijoV1+"/lotes-ordinarios" {
		h.denegarActor(w, r, s, http.StatusNotFound, "recurso_no_encontrado", "escribir", "")
		return
	}
	switch {
	case p == PrefijoV1+"/lotes-ordinarios":
		h.postLoteOrdinario(w, r, s)
	case p == PrefijoV1+"/actos-ordinarios" || p == PrefijoV1+"/propuestas":
		var dto SolicitudActo
		if err := decodificar(w, r, &dto); err != nil {
			estado := http.StatusBadRequest
			if errors.Is(err, errCuerpoExcesivo) {
				estado = http.StatusRequestEntityTooLarge
			}
			h.denegarActor(w, r, s, estado, "solicitud_invalida", "escribir", "")
			return
		}
		prefijoOperacion := "acto_admin:"
		accion := "aplicar_ordinario"
		if p == PrefijoV1+"/propuestas" {
			prefijoOperacion, accion = "propuesta_admin:", "proponer"
		}
		if !domain.ReferenciaAdministracionPerfilesValida(dto.OperacionRef, prefijoOperacion) {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
			return
		}
		clase, err := h.catalogo.ResolverRolAdministrable(r.Context(), dto.RolVersionRef)
		if err != nil {
			falloError(w, err)
			return
		}
		solicitud := domain.SolicitudActoAdministracionPerfiles{ReferenciaActo: dto.ReferenciaActo, OperacionRef: dto.OperacionRef,
			Actor: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Operacion: domain.OperacionAdministracionPerfiles(dto.Operacion), Clase: clase.Clase,
			RolVersionRef: dto.RolVersionRef, Objetivo: dto.Objetivo.dominio(),
			Motivo: dto.Motivo.dominio(), CorrelacionRef: s.CorrelacionRef}
		if solicitud.Validar() != nil || (clase.UnidadRequerida && solicitud.Objetivo.UnidadRef == "") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
			return
		}
		if p == PrefijoV1+"/actos-ordinarios" {
			if clase.Clase != domain.ClaseControlPerfilOrdinario {
				h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
				return
			}
			recibo, err := h.actos.AplicarOrdinario(r.Context(), solicitud)
			if err != nil {
				falloError(w, err)
				return
			}
			if recibo.ValidarPara(solicitud) != nil {
				fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
				return
			}
			jsonRespuesta(w, http.StatusOK, struct {
				Recibo Recibo `json:"recibo"`
			}{reciboDTO(recibo)})
			return
		}
		if !clase.Clase.RequiereDobleControl() {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
			return
		}
		propuesta, err := h.actos.ProponerSensible(r.Context(), solicitud)
		if err != nil {
			falloError(w, err)
			return
		}
		if propuesta.ValidarPara(solicitud) != nil {
			fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		propuestaDTO := struct {
			PropuestaRef string `json:"propuesta_ref"`
			HuellaSHA256 string `json:"huella_sha256"`
			CaducaEn     any    `json:"caduca_en"`
		}{propuesta.PropuestaRef, propuesta.HuellaSHA256, propuesta.CaducaEn}
		jsonRespuesta(w, http.StatusOK, struct {
			Propuesta any `json:"propuesta"`
		}{propuestaDTO})
	case strings.HasPrefix(p, PrefijoV1+"/propuestas/") && strings.HasSuffix(p, "/cierre"):
		ref := strings.TrimSuffix(strings.TrimPrefix(p, PrefijoV1+"/propuestas/"), "/cierre")
		if !domain.ReferenciaAdministracionPerfilesValida(ref, "propuesta_admin:") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "cerrar_propuesta", "")
			return
		}
		var dto SolicitudCierre
		if err := decodificar(w, r, &dto); err != nil {
			estado := http.StatusBadRequest
			if errors.Is(err, errCuerpoExcesivo) {
				estado = http.StatusRequestEntityTooLarge
			}
			h.denegarActor(w, r, s, estado, "solicitud_invalida", "cerrar_propuesta", ref)
			return
		}
		propuesta, err := h.lecturas.ConsultarPropuesta(r.Context(), s.Actor, s.Evidencia, ref)
		if err != nil {
			falloError(w, err)
			return
		}
		if propuesta.PropuestaRef != ref || propuesta.HuellaSHA256 != dto.PropuestaHuellaSHA256 {
			h.denegarActor(w, r, s, http.StatusConflict, "conflicto_estado", "cerrar_propuesta", ref)
			return
		}
		solicitud := domain.SolicitudCierrePropuestaAdministracionPerfiles{
			OperacionRef: dto.OperacionRef, PropuestaRef: ref, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
			ProponentePersonaRef: propuesta.ProponentePersonaRef, ObjetivoPersonaRef: propuesta.ObjetivoPersonaRef,
			Aprobador: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Decision: domain.DecisionPropuestaAdministracionPerfiles(dto.Decision), Motivo: dto.Motivo.dominio(),
			CorrelacionRef: s.CorrelacionRef}
		if solicitud.Validar() != nil {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "cerrar_propuesta", ref)
			return
		}
		cierre, err := h.actos.CerrarPropuestaSensible(r.Context(), solicitud)
		if err != nil {
			falloError(w, err)
			return
		}
		if cierre.ValidarPara(solicitud) != nil {
			fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		var recibo *Recibo
		if cierre.Recibo != nil {
			r := reciboDTO(*cierre.Recibo)
			recibo = &r
		}
		cierreDTO := struct {
			OperacionRef          string  `json:"operacion_ref"`
			PropuestaRef          string  `json:"propuesta_ref"`
			PropuestaHuellaSHA256 string  `json:"propuesta_huella_sha256"`
			Decision              string  `json:"decision"`
			HuellaCierreSHA256    string  `json:"huella_cierre_sha256"`
			ConfirmadoEn          any     `json:"confirmado_en"`
			Recibo                *Recibo `json:"recibo,omitempty"`
		}{
			cierre.OperacionRef, cierre.PropuestaRef, cierre.PropuestaHuellaSHA256, string(cierre.Decision), cierre.HuellaCierreSHA256, cierre.ConfirmadoEn, recibo}
		jsonRespuesta(w, http.StatusOK, struct {
			Cierre any `json:"cierre"`
		}{cierreDTO})
	default:
		h.denegarActor(w, r, s, http.StatusNotFound, "recurso_no_encontrado", "escribir", "")
	}
}
