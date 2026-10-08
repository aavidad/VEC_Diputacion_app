package domain

import (
	"reflect"
	"time"
)

type SolicitudPropuestaVersionarRolBolsa struct {
	OperacionRef            string
	Actor                   ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Intencion               IntencionVersionarRolBolsa
	HuellaPlanEsperada      string
	CorrelacionRef          string
}

func (s SolicitudPropuestaVersionarRolBolsa) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "propuesta_admin:") ||
		s.Actor.Validar() != nil || s.Evidencia.ValidarPara(s.Actor) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!huellaSHA256AutorizacionV3NoNula(s.HuellaPlanEsperada) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Intencion.Motivo) ||
		!ReferenciaActoAdministracionValida(s.Intencion.ReferenciaActo) ||
		len(s.Intencion.Asignaciones) < 1 || len(s.Intencion.Asignaciones) > 16 {
		return ErrVersionarRolBolsaInvalido
	}
	return nil
}

// Sin tags JSON para mantener el canon de material de AUT60.
type MaterialPropuestaVersionarRolBolsa struct {
	OperacionRef         string
	ProponentePersonaRef string
	PerfilActivoRef      string
	AsignacionPerfilRef  string
	Plan                 PlanVersionarRolBolsa
}

func (m MaterialPropuestaVersionarRolBolsa) HuellaSHA256() (string, error) {
	if !ReferenciaAdministracionPerfilesValida(m.OperacionRef, "propuesta_admin:") ||
		!referenciaOpacaAdministracionPerfiles(m.ProponentePersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(m.PerfilActivoRef, "prf_") ||
		!textoAutorizacionSinComodinSeguro(m.AsignacionPerfilRef, 512, false) {
		return "", ErrVersionarRolBolsaInvalido
	}
	if _, err := m.Plan.HuellaSHA256(); err != nil {
		return "", err
	}
	return huellaAutorizacion(m)
}

func (m MaterialPropuestaVersionarRolBolsa) Copia() MaterialPropuestaVersionarRolBolsa {
	m.Plan = m.Plan.Copia()
	return m
}

type OrdenPropuestaVersionarRolBolsa struct {
	Solicitud SolicitudPropuestaVersionarRolBolsa
	Material  MaterialPropuestaVersionarRolBolsa
}

func (o OrdenPropuestaVersionarRolBolsa) Validar() error {
	h, err := o.Material.Plan.HuellaSHA256()
	if err != nil || o.Solicitud.Validar() != nil || h != o.Solicitud.HuellaPlanEsperada ||
		o.Material.OperacionRef != o.Solicitud.OperacionRef ||
		o.Material.ProponentePersonaRef != o.Solicitud.Actor.PersonaRef ||
		o.Material.PerfilActivoRef != o.Solicitud.Actor.PerfilActivoRef ||
		o.Material.AsignacionPerfilRef != o.Solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia() {
		return ErrVersionarRolBolsaInvalido
	}
	return nil
}

type PropuestaVersionarRolBolsa struct {
	Material     MaterialPropuestaVersionarRolBolsa
	HuellaSHA256 string
	CaducaEn     time.Time
}

func (p PropuestaVersionarRolBolsa) ValidarPara(o OrdenPropuestaVersionarRolBolsa) error {
	h, err := o.Material.HuellaSHA256()
	hp, errp := p.Material.HuellaSHA256()
	if err != nil || errp != nil || o.Validar() != nil || h != hp || p.HuellaSHA256 != h ||
		!instanteAutorizacionCanonico(p.CaducaEn) {
		return ErrVersionarRolBolsaInvalido
	}
	return nil
}

// El cierre recibe sólo referencia y huella. SQL recupera el material
// inmutable bajo la decisión V3 y coteja a ambos ADMIN nominales.
type SolicitudCierreVersionarRolBolsa struct {
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

func (s SolicitudCierreVersionarRolBolsa) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "cierre_admin:") ||
		!ReferenciaAdministracionPerfilesValida(s.PropuestaRef, "propuesta_admin:") ||
		!huellaSHA256AutorizacionV3NoNula(s.PropuestaHuellaSHA256) ||
		s.Aprobador.Validar() != nil || s.Evidencia.ValidarPara(s.Aprobador) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil || s.Decision != DecisionAprobarPropuestaPerfil ||
		s.Aprobador.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Aprobador.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrVersionarRolBolsaInvalido
	}
	return nil
}

type AvanceAsignacionVersionarRolBolsa struct {
	Anterior        AsignacionPerfil `json:"anterior"`
	Posterior       AsignacionPerfil `json:"posterior"`
	AnteriorSHA256  string           `json:"anterior_sha256"`
	PosteriorSHA256 string           `json:"posterior_sha256"`
}

type ReciboVersionarRolBolsa struct {
	ActoRef             string                              `json:"acto_ref"`
	ReciboRef           string                              `json:"recibo_ref"`
	ActorPersonaRef     string                              `json:"actor_persona_ref"`
	PerfilActivoRef     string                              `json:"perfil_activo_ref"`
	AsignacionPerfilRef string                              `json:"asignacion_perfil_ref"`
	CorrelacionRef      string                              `json:"correlacion_ref"`
	Motivo              ReferenciaEntradaCatalogo           `json:"motivo"`
	AuditoriaRef        string                              `json:"auditoria_ref"`
	VersionRol          VersionRol                          `json:"version_rol"`
	ControlPosterior    ControlVigenciaVersionRol           `json:"control_posterior"`
	Asignaciones        []AvanceAsignacionVersionarRolBolsa `json:"asignaciones"`
}

type CierreVersionarRolBolsa struct {
	OperacionRef          string
	Material              MaterialPropuestaVersionarRolBolsa
	PropuestaHuellaSHA256 string
	Decision              DecisionPropuestaAdministracionPerfiles
	ConfirmadoEn          time.Time
	AuditoriaAccesoRef    string
	Recibo                *ReciboVersionarRolBolsa
}

func (c CierreVersionarRolBolsa) ValidarPara(s SolicitudCierreVersionarRolBolsa) error {
	h, err := c.Material.HuellaSHA256()
	if err != nil || s.Validar() != nil || h != s.PropuestaHuellaSHA256 ||
		c.PropuestaHuellaSHA256 != h || c.Material.OperacionRef != s.PropuestaRef ||
		c.Material.ProponentePersonaRef == s.Aprobador.PersonaRef ||
		c.Material.Plan.Motivo != s.Motivo || c.OperacionRef != s.OperacionRef ||
		c.Decision != s.Decision || !instanteAutorizacionCanonico(c.ConfirmadoEn) ||
		!textoAutorizacionSinComodinSeguro(c.AuditoriaAccesoRef, 256, false) {
		return ErrVersionarRolBolsaInvalido
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
		return ErrVersionarRolBolsaInvalido
	}
	d := c.Material.Plan.DefinicionNueva
	if r.VersionRol.RolID != d.RolID || r.VersionRol.Version != d.Version || r.VersionRol.Nombre != d.Nombre ||
		len(r.VersionRol.Concesiones) != len(d.Concesiones) ||
		c.ConfirmadoEn.Before(c.Material.Plan.Base.ControlVigencia.ActualizadoEn) {
		return ErrVersionarRolBolsaInvalido
	}
	for i := range d.Concesiones {
		if !concesionesPerfilAdministracionIguales(d.Concesiones[i], r.VersionRol.Concesiones[i]) {
			return ErrVersionarRolBolsaInvalido
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
			return ErrVersionarRolBolsaInvalido
		}
	}
	return nil
}

func (c CierreVersionarRolBolsa) Copia() CierreVersionarRolBolsa {
	c.Material = c.Material.Copia()
	if c.Recibo != nil {
		r := *c.Recibo
		r.VersionRol.Concesiones = copiarConcesionesVersionarBolsa(r.VersionRol.Concesiones)
		r.Asignaciones = append([]AvanceAsignacionVersionarRolBolsa(nil), r.Asignaciones...)
		for i := range r.Asignaciones {
			r.Asignaciones[i].Anterior = copiarAsignacionVersionarBolsa(r.Asignaciones[i].Anterior)
			r.Asignaciones[i].Posterior = copiarAsignacionVersionarBolsa(r.Asignaciones[i].Posterior)
		}
		c.Recibo = &r
	}
	return c
}
