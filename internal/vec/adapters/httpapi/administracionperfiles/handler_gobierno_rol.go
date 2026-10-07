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
	RutaGobiernoRolProponer = PrefijoV1 + "/gobierno-roles/propuestas"
	RutaGobiernoRolCerrar   = PrefijoV1 + "/gobierno-roles/cierres"
)

// El body sólo elige un descriptor publicado; nunca transporta la concesión,
// el actor, los ámbitos, la evidencia V3 ni metadatos de publicación efectivos.
type solicitudGobiernoRolProponerDTO struct {
	OperacionRef         string                           `json:"operacion_ref"`
	CatalogoRef          string                           `json:"catalogo_ref"`
	CatalogoVersion      int                              `json:"catalogo_version"`
	CatalogoHuellaSHA256 string                           `json:"catalogo_huella_sha256"`
	EntradaRef           string                           `json:"entrada_ref"`
	EntradaVersion       int                              `json:"entrada_version"`
	EntradaHuellaSHA256  string                           `json:"entrada_huella_sha256"`
	RolID                string                           `json:"rol_id"`
	Nombre               string                           `json:"nombre"`
	Motivo               domain.ReferenciaEntradaCatalogo `json:"motivo"`
	ReferenciaActo       string                           `json:"referencia_acto,omitempty"`
}

type ServicioGobiernoRolNuevoADMIN interface {
	ProponerGobiernoPerfil(context.Context, domain.SolicitudPropuestaGobiernoPerfil) (domain.PropuestaGobiernoPerfil, error)
	CerrarGobiernoRolPorReferencia(context.Context, domain.SolicitudCierreGobiernoRolPorReferencia) (domain.CierreGobiernoPerfil, error)
}

type solicitudGobiernoRolCerrarDTO struct {
	OperacionRef          string                           `json:"operacion_ref"`
	PropuestaRef          string                           `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                           `json:"propuesta_huella_sha256"`
	Motivo                domain.ReferenciaEntradaCatalogo `json:"motivo"`
}

// ConGobiernoRolNuevo sólo se admite en la superficie ADMIN nominal de
// usuarios. La fuente es la publicación AUT58 exacta y el servicio vuelve a
// resolverla antes de proponer; SQL coteja cabeza y huella al cerrar.
func (h *Handler) ConGobiernoRolNuevo(s ServicioGobiernoRolNuevoADMIN,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, reloj ports.Reloj) error {
	if h == nil || dependenciaNula(s) || dependenciaNula(fuente) || dependenciaNula(reloj) ||
		!h.soloMetadatos || h.gobiernoRol != nil || h.fuenteGobiernoRol != nil {
		return ErrConfiguracionIncompleta
	}
	h.gobiernoRol, h.fuenteGobiernoRol, h.relojGobiernoRol, h.soloLectura = s, fuente, reloj, false
	return nil
}

func (h *Handler) postGobiernoRolProponer(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.gobiernoRol) || dependenciaNula(h.fuenteGobiernoRol) || dependenciaNula(h.relojGobiernoRol) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto solicitudGobiernoRolProponerDTO
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
		dto.EntradaRef == "" || dto.EntradaVersion < 1 || dto.EntradaHuellaSHA256 == "" {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	catalogo, err := h.fuenteGobiernoRol.ObtenerCatalogoAccionesAdministracionV1(r.Context(),
		dto.CatalogoRef, dto.CatalogoVersion, dto.CatalogoHuellaSHA256)
	if err != nil {
		falloError(w, err)
		return
	}
	var seleccion *domain.EntradaAccionAdministracionV1
	for i := range catalogo.Entradas {
		e := &catalogo.Entradas[i]
		huella, errorHuella := e.HuellaSHA256()
		if errorHuella == nil && e.Referencia == dto.EntradaRef && e.Version == dto.EntradaVersion &&
			huella == dto.EntradaHuellaSHA256 {
			seleccion = e
			break
		}
	}
	if seleccion == nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	ahora := h.relojGobiernoRol.Ahora().UTC().Truncate(time.Microsecond)
	propuestaRol := domain.VersionRol{RolID: dto.RolID, Version: 1, Nombre: dto.Nombre,
		Estado: domain.EstadoVersionRolPublicada, Concesiones: []domain.ConcesionRol{seleccion.Concesion},
		PublicadaPor: sesion.Actor.PersonaRef, PublicadaEn: ahora}
	publicacion := domain.PropuestaPerfilAdministracionV1{CatalogoRef: dto.CatalogoRef,
		CatalogoVersion: dto.CatalogoVersion, CatalogoHuellaSHA256: dto.CatalogoHuellaSHA256,
		RolPropuesto: propuestaRol, Selecciones: []domain.SeleccionAccionAdministracionV1{{
			EntradaRef: dto.EntradaRef, EntradaVersion: dto.EntradaVersion,
			EntradaHuellaSHA256: dto.EntradaHuellaSHA256}}}
	intencion := domain.SolicitudPlanGobiernoPerfil{Operacion: domain.OperacionCrearPerfilGobernado,
		Publicacion: &publicacion, Motivo: dto.Motivo, ReferenciaActo: dto.ReferenciaActo}
	plan, err := domain.PrepararPlanGobiernoPerfil(catalogo, intencion, ahora)
	if err != nil || plan.Base != nil || plan.DefinicionNueva == nil || len(plan.DefinicionNueva.Concesiones) != 1 {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	huellaPlan, err := plan.HuellaSHA256()
	if err != nil {
		h.denegarActor(w, r, sesion, http.StatusServiceUnavailable, "servicio_no_disponible", accion, "")
		return
	}
	solicitud := domain.SolicitudPropuestaGobiernoPerfil{OperacionRef: dto.OperacionRef,
		Actor: sesion.Actor, Evidencia: sesion.Evidencia, InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Intencion: intencion, HuellaPlanEsperada: huellaPlan, CorrelacionRef: sesion.CorrelacionRef}
	propuesta, err := h.gobiernoRol.ProponerGobiernoPerfil(r.Context(), solicitud)
	if err != nil {
		falloError(w, err)
		return
	}
	if propuesta.Material.OperacionRef != dto.OperacionRef || propuesta.Material.Plan.VersionRolObjetivoRef != propuestaRol.Referencia() {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	jsonRespuesta(w, http.StatusCreated, struct {
		PropuestaRef       string    `json:"propuesta_ref"`
		HuellaSHA256       string    `json:"huella_sha256"`
		VersionRolObjetivo string    `json:"version_rol_objetivo_ref"`
		CaducaEn           time.Time `json:"caduca_en"`
	}{propuesta.Material.OperacionRef, propuesta.HuellaSHA256,
		propuesta.Material.Plan.VersionRolObjetivoRef, propuesta.CaducaEn})
}

func (h *Handler) postGobiernoRolCerrar(w http.ResponseWriter, r *http.Request, sesion SesionConfiable) {
	const accion = "escribir"
	if dependenciaNula(h.gobiernoRol) {
		h.denegarActor(w, r, sesion, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto solicitudGobiernoRolCerrarDTO
	if err := decodificar(w, r, &dto); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, sesion, estado, "solicitud_invalida", accion, "")
		return
	}
	solicitud := domain.SolicitudCierreGobiernoRolPorReferencia{OperacionRef: dto.OperacionRef,
		PropuestaRef: dto.PropuestaRef, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
		Aprobador: sesion.Actor, Evidencia: sesion.Evidencia, InstantaneaAutorizacion: sesion.InstantaneaAutorizacion,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: dto.Motivo, CorrelacionRef: sesion.CorrelacionRef}
	if solicitud.Validar() != nil {
		h.denegarActor(w, r, sesion, http.StatusBadRequest, "solicitud_invalida", accion, dto.PropuestaRef)
		return
	}
	cierre, err := h.gobiernoRol.CerrarGobiernoRolPorReferencia(r.Context(), solicitud)
	if err != nil {
		falloError(w, err)
		return
	}
	completa, err := solicitud.CompletarCierreGobiernoRolConMaterial(cierre.Material)
	if err != nil || cierre.ValidarPara(completa) != nil || cierre.Recibo == nil {
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
	}{cierre.OperacionRef, cierre.Material.OperacionRef,
		cierre.Material.Plan.VersionRolObjetivoRef, cierre.Recibo.ReciboRef,
		cierre.Recibo.AuditoriaRef, cierre.AuditoriaAccesoRef, cierre.ConfirmadoEn})
}
