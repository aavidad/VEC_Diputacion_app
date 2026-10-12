package administracionperfiles

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaVersionarRolBolsaB103Proponer = PrefijoV1 + "/gobierno-version-bolsa-b103/propuestas"
	RutaVersionarRolBolsaB103Cerrar   = PrefijoV1 + "/gobierno-version-bolsa-b103/cierres"
)

// El cuerpo sólo expresa selecciones y preimágenes CAS. La identidad, el
// catálogo autoritativo, el plan y su huella proceden de la frontera confiable.
type propuestaVersionarRolBolsaB103DTO struct {
	OperacionRef         string                                        `json:"operacion_ref"`
	CatalogoRef          string                                        `json:"catalogo_ref"`
	CatalogoVersion      int                                           `json:"catalogo_version"`
	CatalogoHuellaSHA256 string                                        `json:"catalogo_huella_sha256"`
	BaseRef              string                                        `json:"base_ref"`
	BaseHuellaSHA256     string                                        `json:"base_huella_sha256"`
	ControlRevision      uint64                                        `json:"control_revision"`
	ControlHuellaSHA256  string                                        `json:"control_huella_sha256"`
	Selecciones          []domain.SeleccionAccionAdministracionV1      `json:"selecciones"`
	Asignaciones         []domain.SeleccionAsignacionVersionarRolBolsa `json:"asignaciones"`
	Motivo               domain.ReferenciaEntradaCatalogo              `json:"motivo"`
	ReferenciaActo       string                                        `json:"referencia_acto,omitempty"`
}

type ServicioVersionarRolBolsaB103ADMIN interface {
	ProponerVersionarRolBolsaB103(context.Context, domain.SolicitudPropuestaVersionarRolBolsaB103) (ports.ResultadoPropuestaVersionarRolBolsaB103, error)
	CerrarVersionarRolBolsaB103PorReferencia(context.Context, domain.SolicitudCierreVersionarRolBolsaB103) (domain.CierreVersionarRolBolsaB103, error)
}

func (h *Handler) ConVersionarRolBolsaB103(s ServicioVersionarRolBolsaB103ADMIN,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, reloj ports.Reloj) error {
	if h == nil || dependenciaNula(s) || dependenciaNula(fuente) || dependenciaNula(reloj) ||
		!h.soloMetadatos || h.versionBolsaB103 != nil || h.fuenteVersionBolsaB103 != nil {
		return ErrConfiguracionIncompleta
	}
	h.versionBolsaB103, h.fuenteVersionBolsaB103, h.relojVersionBolsaB103, h.soloLectura = s, fuente, reloj, false
	return nil
}

func (h *Handler) postVersionarRolBolsaB103Proponer(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.versionBolsaB103) || dependenciaNula(h.fuenteVersionBolsaB103) || dependenciaNula(h.relojVersionBolsaB103) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto propuestaVersionarRolBolsaB103DTO
	if err := decodificarLimitado(w, r, &dto, 64*1024); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, sesion, estado, "solicitud_invalida", accion, "")
		return
	}
	if !domain.ReferenciaAdministracionPerfilesValida(dto.OperacionRef, "propuesta_admin:") ||
		dto.CatalogoRef == "" || dto.CatalogoVersion < 1 || dto.CatalogoHuellaSHA256 == "" ||
		dto.BaseRef == "" || dto.BaseHuellaSHA256 == "" || dto.ControlRevision == 0 ||
		dto.ControlHuellaSHA256 == "" || (len(dto.Selecciones) != 4 && len(dto.Selecciones) != 1 &&
		len(dto.Selecciones) != 3) ||
		len(dto.Asignaciones) < 1 || len(dto.Asignaciones) > 16 {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	catalogo, err := h.fuenteVersionBolsaB103.ObtenerCatalogoAccionesAdministracionV1(r.Context(),
		dto.CatalogoRef, dto.CatalogoVersion, dto.CatalogoHuellaSHA256)
	if err != nil {
		h.responderErrorVersionBolsa(w, r, sesion, ports.ConClaseVersionBolsa("catalogo_consulta", err),
			dto.OperacionRef, "propuesta_catalogo")
		return
	}
	intencion := domain.IntencionVersionarRolBolsaB103{CatalogoRef: dto.CatalogoRef,
		CatalogoVersion: dto.CatalogoVersion, CatalogoHuellaSHA256: dto.CatalogoHuellaSHA256,
		BaseRef: dto.BaseRef, BaseHuellaSHA256: dto.BaseHuellaSHA256,
		ControlRevision: dto.ControlRevision, ControlHuellaSHA256: dto.ControlHuellaSHA256,
		Selecciones: dto.Selecciones, Asignaciones: dto.Asignaciones, Motivo: dto.Motivo,
		ReferenciaActo: dto.ReferenciaActo}
	ahora := h.relojVersionBolsaB103.Ahora().UTC().Truncate(time.Microsecond)
	plan, err := domain.PrepararPlanVersionarRolBolsaB103(catalogo, intencion, ahora)
	if err != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.OperacionRef)
		return
	}
	huella, err := plan.HuellaSHA256()
	if err != nil {
		h.responderErrorVersionBolsa(w, r, sesion, err, dto.OperacionRef, "propuesta_huella_plan")
		return
	}
	solicitud := domain.SolicitudPropuestaVersionarRolBolsaB103{OperacionRef: dto.OperacionRef,
		Actor: sesion.Actor, Evidencia: sesion.Evidencia, InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Intencion: intencion, HuellaPlanEsperada: huella, CorrelacionRef: sesion.CorrelacionRef}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.OperacionRef)
		return
	}
	resultado, err := h.versionBolsaB103.ProponerVersionarRolBolsaB103(r.Context(), solicitud)
	if err != nil {
		h.responderErrorVersionBolsa(w, r, sesion, err, dto.OperacionRef, "propuesta_servicio")
		return
	}
	orden := domain.OrdenPropuestaVersionarRolBolsaB103{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionarRolBolsaB103{OperacionRef: dto.OperacionRef,
			ProponentePersonaRef: sesion.Actor.PersonaRef, PerfilActivoRef: sesion.Actor.PerfilActivoRef,
			AsignacionPerfilRef: sesion.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}}
	if resultado.ValidarPara(orden, ahora) != nil {
		h.registrarClaseVersionBolsa(r, "propuesta_respuesta_incoherente")
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	propuesta := resultado.Propuesta
	estado := http.StatusCreated
	if resultado.Replay {
		estado = http.StatusOK
	}
	jsonRespuesta(w, estado, struct {
		PropuestaRef       string    `json:"propuesta_ref"`
		HuellaSHA256       string    `json:"huella_sha256"`
		VersionRolObjetivo string    `json:"version_rol_objetivo_ref"`
		CaducaEn           time.Time `json:"caduca_en"`
		Replay             bool      `json:"replay"`
		AuditoriaAccesoRef string    `json:"auditoria_acceso_ref"`
	}{propuesta.Material.OperacionRef, propuesta.HuellaSHA256,
		propuesta.Material.Plan.VersionRolObjetivoRef, propuesta.CaducaEn,
		resultado.Replay, resultado.AuditoriaAccesoRef})
}

func (h *Handler) postVersionarRolBolsaB103Cerrar(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.versionBolsaB103) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto cierreVersionarRolBolsaDTO
	if err := decodificar(w, r, &dto); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, sesion, estado, "solicitud_invalida", accion, "")
		return
	}
	solicitud := domain.SolicitudCierreVersionarRolBolsaB103{OperacionRef: dto.OperacionRef,
		PropuestaRef: dto.PropuestaRef, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
		Aprobador: sesion.Actor, Evidencia: sesion.Evidencia, InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: dto.Motivo, CorrelacionRef: sesion.CorrelacionRef}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.PropuestaRef)
		return
	}
	cierre, err := h.versionBolsaB103.CerrarVersionarRolBolsaB103PorReferencia(r.Context(), solicitud)
	if err != nil {
		h.responderErrorVersionBolsa(w, r, sesion, err, dto.PropuestaRef, "cierre_servicio")
		return
	}
	if cierre.ValidarPara(solicitud) != nil {
		h.registrarClaseVersionBolsa(r, "cierre_respuesta_incoherente")
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	jsonRespuesta(w, http.StatusOK, struct {
		OperacionRef       string                             `json:"operacion_ref"`
		PropuestaRef       string                             `json:"propuesta_ref"`
		VersionRolObjetivo string                             `json:"version_rol_objetivo_ref"`
		ConfirmadoEn       time.Time                          `json:"confirmado_en"`
		AuditoriaAccesoRef string                             `json:"auditoria_acceso_ref"`
		Recibo             domain.ReciboVersionarRolBolsaB103 `json:"recibo"`
	}{cierre.OperacionRef, cierre.Material.OperacionRef,
		cierre.Material.Plan.VersionRolObjetivoRef, cierre.ConfirmadoEn,
		cierre.AuditoriaAccesoRef, *cierre.Recibo})
}
