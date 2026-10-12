package domain

import (
	"reflect"
	"time"
)

type SolicitudPropuestaVersionarRolBolsaB103 struct {
	OperacionRef            string
	Actor                   ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Intencion               IntencionVersionarRolBolsaB103
	HuellaPlanEsperada      string
	CorrelacionRef          string
}

func (s SolicitudPropuestaVersionarRolBolsaB103) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "propuesta_admin:") ||
		s.Actor.Validar() != nil || s.Evidencia.ValidarPara(s.Actor) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!huellaSHA256AutorizacionV3NoNula(s.HuellaPlanEsperada) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Intencion.Motivo) ||
		!ReferenciaActoAdministracionValida(s.Intencion.ReferenciaActo) ||
		len(s.Intencion.Asignaciones) < 1 || len(s.Intencion.Asignaciones) > 16 ||
		(len(s.Intencion.Selecciones) != len(accionesVersionarRolBolsaB103) && len(s.Intencion.Selecciones) != 1) {
		return ErrVersionarRolBolsaB103Invalido
	}
	return nil
}

// Sin tags JSON para mantener el canon de material de AUT75.
type MaterialPropuestaVersionarRolBolsaB103 struct {
	OperacionRef         string
	ProponentePersonaRef string
	PerfilActivoRef      string
	AsignacionPerfilRef  string
	Plan                 PlanVersionarRolBolsaB103
}

func (m MaterialPropuestaVersionarRolBolsaB103) HuellaSHA256() (string, error) {
	if !ReferenciaAdministracionPerfilesValida(m.OperacionRef, "propuesta_admin:") ||
		!referenciaOpacaAdministracionPerfiles(m.ProponentePersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(m.PerfilActivoRef, "prf_") ||
		!textoAutorizacionSinComodinSeguro(m.AsignacionPerfilRef, 512, false) {
		return "", ErrVersionarRolBolsaB103Invalido
	}
	if _, err := m.Plan.HuellaSHA256(); err != nil {
		return "", err
	}
	return huellaAutorizacion(m)
}

func (m MaterialPropuestaVersionarRolBolsaB103) Copia() MaterialPropuestaVersionarRolBolsaB103 {
	m.Plan = m.Plan.Copia()
	return m
}

type OrdenPropuestaVersionarRolBolsaB103 struct {
	Solicitud SolicitudPropuestaVersionarRolBolsaB103
	Material  MaterialPropuestaVersionarRolBolsaB103
}

func (o OrdenPropuestaVersionarRolBolsaB103) Validar() error {
	h, err := o.Material.Plan.HuellaSHA256()
	if err != nil || o.Solicitud.Validar() != nil || h != o.Solicitud.HuellaPlanEsperada ||
		o.Material.OperacionRef != o.Solicitud.OperacionRef ||
		o.Material.ProponentePersonaRef != o.Solicitud.Actor.PersonaRef ||
		o.Material.PerfilActivoRef != o.Solicitud.Actor.PerfilActivoRef ||
		o.Material.AsignacionPerfilRef != o.Solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia() {
		return ErrVersionarRolBolsaB103Invalido
	}
	return nil
}

type PropuestaVersionarRolBolsaB103 struct {
	Material     MaterialPropuestaVersionarRolBolsaB103
	HuellaSHA256 string
	CaducaEn     time.Time
}

func (p PropuestaVersionarRolBolsaB103) ValidarPara(o OrdenPropuestaVersionarRolBolsaB103) error {
	h, err := o.Material.HuellaSHA256()
	hp, errp := p.Material.HuellaSHA256()
	if err != nil || errp != nil || o.Validar() != nil || h != hp || p.HuellaSHA256 != h ||
		!instanteAutorizacionCanonico(p.CaducaEn) {
		return ErrVersionarRolBolsaB103Invalido
	}
	return nil
}

// El cierre recibe sólo referencia y huella. SQL recupera el material
// inmutable bajo la decisión V3 y coteja a ambos ADMIN nominales.
type SolicitudCierreVersionarRolBolsaB103 struct {
	OperacionRef            string
	PropuestaRef            string
	PropuestaHuellaSHA256   string
	Aprobador               ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Decision                DecisionPropuestaAdministracionPerfiles
	Motivo                  ReferenciaEntradaCatalogo
	CorrelacionRef          string
}

func (s SolicitudCierreVersionarRolBolsaB103) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "cierre_admin:") ||
		!ReferenciaAdministracionPerfilesValida(s.PropuestaRef, "propuesta_admin:") ||
		!huellaSHA256AutorizacionV3NoNula(s.PropuestaHuellaSHA256) ||
		s.Aprobador.Validar() != nil || s.Evidencia.ValidarPara(s.Aprobador) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil || s.Decision != DecisionAprobarPropuestaPerfil ||
		s.Aprobador.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Aprobador.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrVersionarRolBolsaB103Invalido
	}
	return nil
}

type AvanceAsignacionVersionarRolBolsaB103 struct {
	Anterior        AsignacionPerfil `json:"anterior"`
	Posterior       AsignacionPerfil `json:"posterior"`
	AnteriorSHA256  string           `json:"anterior_sha256"`
	PosteriorSHA256 string           `json:"posterior_sha256"`
}

type ReciboVersionarRolBolsaB103 struct {
	ActoRef             string                                  `json:"acto_ref"`
	ReciboRef           string                                  `json:"recibo_ref"`
	ActorPersonaRef     string                                  `json:"actor_persona_ref"`
	PerfilActivoRef     string                                  `json:"perfil_activo_ref"`
	AsignacionPerfilRef string                                  `json:"asignacion_perfil_ref"`
	CorrelacionRef      string                                  `json:"correlacion_ref"`
	Motivo              ReferenciaEntradaCatalogo               `json:"motivo"`
	AuditoriaRef        string                                  `json:"auditoria_ref"`
	VersionRol          VersionRol                              `json:"version_rol"`
	ControlPosterior    ControlVigenciaVersionRol               `json:"control_posterior"`
	Asignaciones        []AvanceAsignacionVersionarRolBolsaB103 `json:"asignaciones"`
}

type CierreVersionarRolBolsaB103 struct {
	OperacionRef          string
	Material              MaterialPropuestaVersionarRolBolsaB103
	PropuestaHuellaSHA256 string
	Decision              DecisionPropuestaAdministracionPerfiles
	ConfirmadoEn          time.Time
	AuditoriaAccesoRef    string
	Recibo                *ReciboVersionarRolBolsaB103
}

func (c CierreVersionarRolBolsaB103) ValidarPara(s SolicitudCierreVersionarRolBolsaB103) error {
	h, err := c.Material.HuellaSHA256()
	if err != nil || s.Validar() != nil || h != s.PropuestaHuellaSHA256 ||
		c.PropuestaHuellaSHA256 != h || c.Material.OperacionRef != s.PropuestaRef ||
		c.Material.ProponentePersonaRef == s.Aprobador.PersonaRef ||
		c.Material.PerfilActivoRef == s.Aprobador.PerfilActivoRef ||
		c.Material.Plan.Motivo != s.Motivo || c.OperacionRef != s.OperacionRef ||
		c.Decision != s.Decision || !instanteAutorizacionCanonico(c.ConfirmadoEn) ||
		!textoAutorizacionSinComodinSeguro(c.AuditoriaAccesoRef, 256, false) {
		return ErrVersionarRolBolsaB103Invalido
	}
	r := c.Recibo
	if r == nil || !ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		r.ActorPersonaRef != s.Aprobador.PersonaRef || r.PerfilActivoRef != s.Aprobador.PerfilActivoRef ||
		r.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		!ReferenciaCorrelacionAutorizacionV2Valida(r.CorrelacionRef) || r.Motivo != s.Motivo ||
		!textoAutorizacionSinComodinSeguro(r.AuditoriaRef, 256, false) ||
		r.VersionRol.Validar() != nil || r.ControlPosterior.Validar() != nil ||
		r.VersionRol.Referencia() != c.Material.Plan.VersionRolObjetivoRef ||
		r.VersionRol.Estado != EstadoVersionRolPublicada ||
		r.VersionRol.PublicadaPor != s.Aprobador.PersonaRef || !r.VersionRol.PublicadaEn.Equal(c.ConfirmadoEn) ||
		r.ControlPosterior.VersionRolRef != r.VersionRol.Referencia() || r.ControlPosterior.Revision != 1 ||
		r.ControlPosterior.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		r.ControlPosterior.ActualizadoPor != s.Aprobador.PersonaRef ||
		!r.ControlPosterior.ActualizadoEn.Equal(c.ConfirmadoEn) ||
		len(r.Asignaciones) != len(c.Material.Plan.Asignaciones) {
		return ErrVersionarRolBolsaB103Invalido
	}
	d := c.Material.Plan.DefinicionNueva
	if r.VersionRol.RolID != d.RolID || r.VersionRol.Version != d.Version || r.VersionRol.Nombre != d.Nombre ||
		len(r.VersionRol.Concesiones) != len(d.Concesiones) ||
		c.ConfirmadoEn.Before(c.Material.Plan.Base.ControlVigencia.ActualizadoEn) {
		return ErrVersionarRolBolsaB103Invalido
	}
	for i := range d.Concesiones {
		if !reflect.DeepEqual(d.Concesiones[i], r.VersionRol.Concesiones[i]) {
			return ErrVersionarRolBolsaB103Invalido
		}
	}
	for i, avance := range r.Asignaciones {
		esperada := c.Material.Plan.Asignaciones[i]
		ha, ea := avance.Anterior.HuellaSHA256()
		hp, ep := avance.Posterior.HuellaSHA256()
		if ea != nil || ep != nil || ha != esperada.HuellaSHA256 || ha != avance.AnteriorSHA256 ||
			avance.Anterior.Referencia() != esperada.AsignacionRef ||
			hp != avance.PosteriorSHA256 || avance.Posterior.AsignacionID != avance.Anterior.AsignacionID ||
			avance.Posterior.Version != avance.Anterior.Version+1 ||
			avance.Posterior.PerfilActivoRef != avance.Anterior.PerfilActivoRef ||
			avance.Posterior.PrincipalID != avance.Anterior.PrincipalID ||
			!reflect.DeepEqual(avance.Posterior.Ambitos, avance.Anterior.Ambitos) ||
			avance.Posterior.VersionRolRef != r.VersionRol.Referencia() ||
			avance.Posterior.Estado != EstadoAsignacionPerfilActiva ||
			!avance.Posterior.VigenteDesde.Equal(c.ConfirmadoEn) ||
			!avance.Posterior.VigenteHasta.Equal(avance.Anterior.VigenteHasta) ||
			avance.Posterior.EmitidaPor != s.Aprobador.PersonaRef ||
			!avance.Posterior.EmitidaEn.Equal(c.ConfirmadoEn) ||
			!avance.Anterior.VigenteEn(c.ConfirmadoEn) ||
			!avance.Posterior.VigenteEn(c.ConfirmadoEn) {
			return ErrVersionarRolBolsaB103Invalido
		}
	}
	return nil
}

func (c CierreVersionarRolBolsaB103) Copia() CierreVersionarRolBolsaB103 {
	c.Material = c.Material.Copia()
	if c.Recibo != nil {
		r := *c.Recibo
		r.VersionRol.Concesiones = copiarConcesionesVersionarBolsa(r.VersionRol.Concesiones)
		r.Asignaciones = append([]AvanceAsignacionVersionarRolBolsaB103(nil), r.Asignaciones...)
		for i := range r.Asignaciones {
			r.Asignaciones[i].Anterior = copiarAsignacionVersionarBolsa(r.Asignaciones[i].Anterior)
			r.Asignaciones[i].Posterior = copiarAsignacionVersionarBolsa(r.Asignaciones[i].Posterior)
		}
		c.Recibo = &r
	}
	return c
}
