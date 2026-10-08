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
	RutaVersionarRolBolsaProponer = PrefijoV1 + "/gobierno-version-bolsa/propuestas"
	RutaVersionarRolBolsaCerrar   = PrefijoV1 + "/gobierno-version-bolsa/cierres"
)

// Las preimágenes del cuerpo son expectativas CAS, nunca una fuente de
// concesiones o identidad. AUT63 las coteja bajo bloqueo con la autoridad.
type propuestaVersionarRolBolsaDTO struct {
	OperacionRef         string                                        `json:"operacion_ref"`
	CatalogoRef          string                                        `json:"catalogo_ref"`
	CatalogoVersion      int                                           `json:"catalogo_version"`
	CatalogoHuellaSHA256 string                                        `json:"catalogo_huella_sha256"`
	BaseRef              string                                        `json:"base_ref"`
	BaseHuellaSHA256     string                                        `json:"base_huella_sha256"`
	ControlRevision      uint64                                        `json:"control_revision"`
	ControlHuellaSHA256  string                                        `json:"control_huella_sha256"`
	Seleccion            domain.SeleccionAccionAdministracionV1        `json:"seleccion"`
	Asignaciones         []domain.SeleccionAsignacionVersionarRolBolsa `json:"asignaciones"`
	Motivo               domain.ReferenciaEntradaCatalogo              `json:"motivo"`
	ReferenciaActo       string                                        `json:"referencia_acto,omitempty"`
}

type cierreVersionarRolBolsaDTO struct {
	OperacionRef          string                           `json:"operacion_ref"`
	PropuestaRef          string                           `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                           `json:"propuesta_huella_sha256"`
	Motivo                domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

type ServicioVersionarRolBolsaADMIN interface {
	ProponerVersionarRolBolsa(context.Context, domain.SolicitudPropuestaVersionarRolBolsa) (ports.ResultadoPropuestaVersionarRolBolsa, error)
	CerrarVersionarRolBolsaPorReferencia(context.Context, domain.SolicitudCierreVersionarRolBolsa) (domain.CierreVersionarRolBolsa, error)
}

func (h *Handler) ConVersionarRolBolsa(s ServicioVersionarRolBolsaADMIN,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, reloj ports.Reloj) error {
	if h == nil || dependenciaNula(s) || dependenciaNula(fuente) || dependenciaNula(reloj) ||
		!h.soloMetadatos || h.versionBolsa != nil || h.fuenteVersionBolsa != nil {
		return ErrConfiguracionIncompleta
	}
	h.versionBolsa, h.fuenteVersionBolsa, h.relojVersionBolsa, h.soloLectura = s, fuente, reloj, false
	return nil
}

func (h *Handler) postVersionarRolBolsaProponer(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.versionBolsa) || dependenciaNula(h.fuenteVersionBolsa) || dependenciaNula(h.relojVersionBolsa) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto propuestaVersionarRolBolsaDTO
	if err := decodificar(w, r, &dto); err != nil {
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
		dto.ControlHuellaSHA256 == "" || len(dto.Asignaciones) < 1 || len(dto.Asignaciones) > 16 {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	catalogo, err := h.fuenteVersionBolsa.ObtenerCatalogoAccionesAdministracionV1(r.Context(),
		dto.CatalogoRef, dto.CatalogoVersion, dto.CatalogoHuellaSHA256)
	if err != nil {
		h.responderErrorGobiernoRol(w, r, sesion, err, dto.OperacionRef)
		return
	}
	intencion := domain.IntencionVersionarRolBolsa{CatalogoRef: dto.CatalogoRef,
		CatalogoVersion: dto.CatalogoVersion, CatalogoHuellaSHA256: dto.CatalogoHuellaSHA256,
		BaseRef: dto.BaseRef, BaseHuellaSHA256: dto.BaseHuellaSHA256,
		ControlRevision: dto.ControlRevision, ControlHuellaSHA256: dto.ControlHuellaSHA256,
		Seleccion: dto.Seleccion, Asignaciones: dto.Asignaciones, Motivo: dto.Motivo,
		ReferenciaActo: dto.ReferenciaActo}
	plan, err := domain.PrepararPlanVersionarRolBolsa(catalogo, intencion,
		h.relojVersionBolsa.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.OperacionRef)
		return
	}
	huella, err := plan.HuellaSHA256()
	if err != nil {
		h.denegarActor(w, r, sesion, http.StatusServiceUnavailable, "servicio_no_disponible", accion, dto.OperacionRef)
		return
	}
	solicitud := domain.SolicitudPropuestaVersionarRolBolsa{OperacionRef: dto.OperacionRef,
		Actor: sesion.Actor, Evidencia: sesion.Evidencia, InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Intencion: intencion, HuellaPlanEsperada: huella, CorrelacionRef: sesion.CorrelacionRef}
	resultado, err := h.versionBolsa.ProponerVersionarRolBolsa(r.Context(), solicitud)
	if err != nil {
		h.responderErrorGobiernoRol(w, r, sesion, err, dto.OperacionRef)
		return
	}
	propuesta := resultado.Propuesta
	if propuesta.Material.OperacionRef != dto.OperacionRef || propuesta.Material.Plan.VersionRolObjetivoRef != plan.VersionRolObjetivoRef {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
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

func (h *Handler) postVersionarRolBolsaCerrar(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.versionBolsa) {
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
	solicitud := domain.SolicitudCierreVersionarRolBolsa{OperacionRef: dto.OperacionRef,
		PropuestaRef: dto.PropuestaRef, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
		Aprobador: sesion.Actor, Evidencia: sesion.Evidencia, InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: dto.Motivo, CorrelacionRef: sesion.CorrelacionRef}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.PropuestaRef)
		return
	}
	cierre, err := h.versionBolsa.CerrarVersionarRolBolsaPorReferencia(r.Context(), solicitud)
	if err != nil {
		h.responderErrorGobiernoRol(w, r, sesion, err, dto.PropuestaRef)
		return
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.Recibo == nil {
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
