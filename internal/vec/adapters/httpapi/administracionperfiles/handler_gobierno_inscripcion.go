package administracionperfiles

import (
	"context"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaGobiernoInscripcionProponer = PrefijoV1 + "/gobierno-inscripcion/propuestas"
	RutaGobiernoInscripcionCerrar   = PrefijoV1 + "/gobierno-inscripcion/cierres"
)

type ServicioGobiernoInscripcionADMIN interface {
	ProponerVersionInscripcion(context.Context, domain.SolicitudPropuestaVersionInscripcion) (domain.PropuestaVersionInscripcion, bool, error)
	CerrarVersionInscripcion(context.Context, domain.SolicitudCierreVersionInscripcion) (domain.CierreVersionInscripcion, bool, error)
}

type solicitudGobiernoInscripcionProponerDTO struct {
	OperacionRef string                        `json:"operacion_ref"`
	Plan         domain.PlanVersionInscripcion `json:"plan"`
}

type solicitudGobiernoInscripcionCerrarDTO struct {
	OperacionRef          string                           `json:"operacion_ref"`
	PropuestaRef          string                           `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                           `json:"propuesta_huella_sha256"`
	Motivo                domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

// El plan enviado por HTTP es intención de selección, nunca fuente de
// permisos. Aplicación y AUT68 resuelven catálogo/preimagen exactos y V3.
func (h *Handler) ConGobiernoInscripcion(s ServicioGobiernoInscripcionADMIN) error {
	if h == nil || dependenciaNula(s) || !h.soloMetadatos || h.gobiernoInscripcion != nil {
		return ErrConfiguracionIncompleta
	}
	h.gobiernoInscripcion, h.soloLectura = s, false
	return nil
}

func (h *Handler) postGobiernoInscripcionProponer(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	if dependenciaNula(h.gobiernoInscripcion) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", "escribir", "")
		return
	}
	var dto solicitudGobiernoInscripcionProponerDTO
	if err := decodificarLimitado(w, r, &dto, 196608); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, sesion, estado, "solicitud_invalida", "escribir", "")
		return
	}
	if !domain.ReferenciaAdministracionPerfilesValida(dto.OperacionRef, "propuesta_admin:") ||
		dto.Plan.ValidarEstructura() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", "escribir", dto.OperacionRef)
		return
	}
	solicitud := domain.SolicitudPropuestaVersionInscripcion{OperacionRef: dto.OperacionRef,
		Actor: sesion.Actor, Evidencia: sesion.Evidencia,
		InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Plan:                    dto.Plan, CorrelacionRef: sesion.CorrelacionRef}
	propuesta, replay, err := h.gobiernoInscripcion.ProponerVersionInscripcion(r.Context(), solicitud)
	if err != nil {
		h.responderErrorGobiernoInscripcion(w, r, sesion, err, dto.OperacionRef)
		return
	}
	if propuesta.Material.OperacionRef != dto.OperacionRef ||
		propuesta.Material.Plan.VersionRolObjetivoRef != dto.Plan.VersionRolObjetivoRef {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	estado := http.StatusCreated
	if replay {
		estado = http.StatusOK
	}
	jsonRespuesta(w, estado, struct {
		PropuestaRef       string `json:"propuesta_ref"`
		HuellaSHA256       string `json:"huella_sha256"`
		VersionRolObjetivo string `json:"version_rol_objetivo_ref"`
		CaducaEn           any    `json:"caduca_en"`
		Replay             bool   `json:"replay"`
		AuditoriaAccesoRef string `json:"auditoria_acceso_ref"`
	}{propuesta.Material.OperacionRef, propuesta.HuellaSHA256,
		propuesta.Material.Plan.VersionRolObjetivoRef, propuesta.CaducaEn, replay, propuesta.AuditoriaAccesoRef})
}

func (h *Handler) postGobiernoInscripcionCerrar(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	if dependenciaNula(h.gobiernoInscripcion) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", "escribir", "")
		return
	}
	var dto solicitudGobiernoInscripcionCerrarDTO
	if err := decodificar(w, r, &dto); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, sesion, estado, "solicitud_invalida", "escribir", "")
		return
	}
	solicitud := domain.SolicitudCierreVersionInscripcion{OperacionRef: dto.OperacionRef,
		PropuestaRef: dto.PropuestaRef, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
		Aprobador: sesion.Actor, Evidencia: sesion.Evidencia,
		InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil, Motivo: dto.Motivo,
		CorrelacionRef: sesion.CorrelacionRef}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", "escribir", dto.PropuestaRef)
		return
	}
	cierre, replay, err := h.gobiernoInscripcion.CerrarVersionInscripcion(r.Context(), solicitud)
	if err != nil {
		h.responderErrorGobiernoInscripcion(w, r, sesion, err, dto.PropuestaRef)
		return
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.Recibo == nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	estado := http.StatusCreated
	if replay {
		estado = http.StatusOK
	}
	jsonRespuesta(w, estado, struct {
		OperacionRef       string                          `json:"operacion_ref"`
		PropuestaRef       string                          `json:"propuesta_ref"`
		VersionRolObjetivo string                          `json:"version_rol_objetivo_ref"`
		AuditoriaAccesoRef string                          `json:"auditoria_acceso_ref"`
		Replay             bool                            `json:"replay"`
		Recibo             domain.ReciboVersionInscripcion `json:"recibo"`
	}{cierre.OperacionRef, cierre.Material.OperacionRef,
		cierre.Material.Plan.VersionRolObjetivoRef, cierre.AuditoriaAccesoRef, replay, *cierre.Recibo})
}

func (h *Handler) responderErrorGobiernoInscripcion(w http.ResponseWriter, r *http.Request,
	sesion SesionConfiable, err error, recurso string) {
	if errors.Is(err, ports.ErrGobiernoInscripcionIntentoAuditado) {
		falloError(w, err)
		return
	}
	estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
	if errors.Is(err, domain.ErrAutorizacionDenegada) {
		estado, codigo = http.StatusForbidden, "acceso_denegado"
	}
	h.denegarActor(w, r, sesion, estado, codigo, "escribir", recurso)
}
