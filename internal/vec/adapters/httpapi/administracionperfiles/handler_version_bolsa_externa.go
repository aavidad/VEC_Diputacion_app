package administracionperfiles

import (
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaVersionarRolBolsaExternaProponer = PrefijoV1 + "/gobierno-version-bolsa-externa/propuestas"
	RutaVersionarRolBolsaExternaCerrar   = PrefijoV1 + "/gobierno-version-bolsa-externa/cierres"
)

// Sólo expectativas CAS y la huella del plan esperado. El catálogo, la base,
// el plan efectivo y el actor se resuelven en la autoridad, no en el cliente.
type propuestaVersionarRolBolsaExternaDTO struct {
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
	HuellaPlanEsperada   string                                        `json:"huella_plan_esperada"`
}

type cierreVersionarRolBolsaExternaDTO struct {
	OperacionRef          string                           `json:"operacion_ref"`
	PropuestaRef          string                           `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                           `json:"propuesta_huella_sha256"`
	Motivo                domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

func (h *Handler) ConVersionarRolBolsaExterna(s ports.ServicioVersionarRolBolsaExterna, reloj ports.Reloj) error {
	if h == nil || dependenciaNula(s) || dependenciaNula(reloj) ||
		!h.soloMetadatos || h.versionBolsaExterna != nil {
		return ErrConfiguracionIncompleta
	}
	h.versionBolsaExterna, h.relojVersionBolsaExterna, h.soloLectura = s, reloj, false
	return nil
}

func (h *Handler) postVersionarRolBolsaExternaProponer(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.versionBolsaExterna) || dependenciaNula(h.relojVersionBolsaExterna) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto propuestaVersionarRolBolsaExternaDTO
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
		dto.ControlHuellaSHA256 == "" || len(dto.Selecciones) != 2 ||
		len(dto.Asignaciones) < 1 || len(dto.Asignaciones) > 16 || dto.HuellaPlanEsperada == "" {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	intencion := domain.IntencionVersionarRolBolsaExterna{
		CatalogoRef: dto.CatalogoRef, CatalogoVersion: dto.CatalogoVersion,
		CatalogoHuellaSHA256: dto.CatalogoHuellaSHA256,
		BaseRef:              dto.BaseRef, BaseHuellaSHA256: dto.BaseHuellaSHA256,
		ControlRevision: dto.ControlRevision, ControlHuellaSHA256: dto.ControlHuellaSHA256,
		Selecciones: dto.Selecciones, Asignaciones: dto.Asignaciones,
		Motivo: dto.Motivo, ReferenciaActo: dto.ReferenciaActo,
	}
	solicitud := domain.SolicitudPropuestaVersionarRolBolsaExterna{
		OperacionRef: dto.OperacionRef, Actor: sesion.Actor, Evidencia: sesion.Evidencia,
		InstantaneaAutorizacion: sesion.InstantaneaAutorizacion, Intencion: intencion,
		HuellaPlanEsperada: dto.HuellaPlanEsperada, CorrelacionRef: sesion.CorrelacionRef,
	}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.OperacionRef)
		return
	}
	resultado, err := h.versionBolsaExterna.ProponerVersionarRolBolsaExterna(r.Context(), solicitud)
	if err != nil {
		h.responderErrorVersionBolsa(w, r, sesion, err, dto.OperacionRef, "propuesta_servicio")
		return
	}
	orden := domain.OrdenPropuestaVersionarRolBolsaExterna{Solicitud: solicitud, Material: resultado.Propuesta.Material}
	if resultado.ValidarPara(orden, h.relojVersionBolsaExterna.Ahora()) != nil {
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

func (h *Handler) postVersionarRolBolsaExternaCerrar(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.versionBolsaExterna) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto cierreVersionarRolBolsaExternaDTO
	if err := decodificarLimitado(w, r, &dto, 16*1024); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, sesion, estado, "solicitud_invalida", accion, "")
		return
	}
	solicitud := domain.SolicitudCierreVersionarRolBolsaExterna{
		OperacionRef: dto.OperacionRef, PropuestaRef: dto.PropuestaRef,
		PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
		Aprobador:             sesion.Actor, Evidencia: sesion.Evidencia,
		InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil, Motivo: dto.Motivo,
		CorrelacionRef: sesion.CorrelacionRef,
	}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.PropuestaRef)
		return
	}
	cierre, err := h.versionBolsaExterna.CerrarVersionarRolBolsaExternaPorReferencia(r.Context(), solicitud)
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
		OperacionRef       string    `json:"operacion_ref"`
		PropuestaRef       string    `json:"propuesta_ref"`
		VersionRolObjetivo string    `json:"version_rol_objetivo_ref"`
		ReciboRef          string    `json:"recibo_ref"`
		AuditoriaRef       string    `json:"auditoria_ref"`
		AuditoriaAccesoRef string    `json:"auditoria_acceso_ref"`
		ConfirmadoEn       time.Time `json:"confirmado_en"`
		Asignaciones       int       `json:"asignaciones"`
	}{cierre.OperacionRef, cierre.Material.OperacionRef,
		cierre.Material.Plan.VersionRolObjetivoRef, cierre.Recibo.ReciboRef,
		cierre.Recibo.AuditoriaRef, cierre.AuditoriaAccesoRef, cierre.ConfirmadoEn,
		len(cierre.Recibo.Asignaciones)})
}
