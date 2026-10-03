package domain

import "time"

type SolicitudPropuestaGobiernoPerfil struct {
	OperacionRef            string
	Actor                   ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Intencion               SolicitudPlanGobiernoPerfil
	HuellaPlanEsperada      string
	CorrelacionRef          string
}

func (s SolicitudPropuestaGobiernoPerfil) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "propuesta_admin:") || s.Actor.Validar() != nil ||
		s.Evidencia.ValidarPara(s.Actor) != nil || s.InstantaneaAutorizacion.Validar() != nil ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!s.Intencion.Operacion.Valida() || !ReferenciaMotivoAutorizacionV2Valida(s.Intencion.Motivo) ||
		!ReferenciaActoAdministracionValida(s.Intencion.ReferenciaActo) || !huellaSHA256AutorizacionV3NoNula(s.HuellaPlanEsperada) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrPlanGobiernoPerfilInvalido
	}
	return nil
}

type MaterialPropuestaGobiernoPerfil struct {
	OperacionRef         string
	ProponentePersonaRef string
	PerfilActivoRef      string
	AsignacionPerfilRef  string
	Plan                 PlanGobiernoPerfil
}

func (m MaterialPropuestaGobiernoPerfil) HuellaSHA256() (string, error) {
	if !ReferenciaAdministracionPerfilesValida(m.OperacionRef, "propuesta_admin:") ||
		!referenciaOpacaAdministracionPerfiles(m.ProponentePersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(m.PerfilActivoRef, "prf_") ||
		!textoAutorizacionSinComodinSeguro(m.AsignacionPerfilRef, 512, false) {
		return "", ErrPlanGobiernoPerfilInvalido
	}
	if _, err := m.Plan.HuellaSHA256(); err != nil {
		return "", err
	}
	return huellaAutorizacion(m)
}

type OrdenPropuestaGobiernoPerfil struct {
	Solicitud SolicitudPropuestaGobiernoPerfil
	Material  MaterialPropuestaGobiernoPerfil
}

func (o OrdenPropuestaGobiernoPerfil) Validar() error {
	h, err := o.Material.Plan.HuellaSHA256()
	if err != nil {
		return err
	}
	if o.Solicitud.Validar() != nil || h != o.Solicitud.HuellaPlanEsperada ||
		o.Material.OperacionRef != o.Solicitud.OperacionRef || o.Material.ProponentePersonaRef != o.Solicitud.Actor.PersonaRef ||
		o.Material.PerfilActivoRef != o.Solicitud.Actor.PerfilActivoRef ||
		o.Material.AsignacionPerfilRef != o.Solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia() {
		return ErrPlanGobiernoPerfilInvalido
	}
	return nil
}

type PropuestaGobiernoPerfil struct {
	Material     MaterialPropuestaGobiernoPerfil
	HuellaSHA256 string
	CaducaEn     time.Time
}

func (p PropuestaGobiernoPerfil) ValidarPara(o OrdenPropuestaGobiernoPerfil) error {
	h, err := o.Material.HuellaSHA256()
	if err != nil {
		return err
	}
	hp, errp := p.Material.HuellaSHA256()
	if errp != nil {
		return errp
	}
	if o.Validar() != nil || h != hp || p.HuellaSHA256 != h || !instanteAutorizacionCanonico(p.CaducaEn) {
		return ErrPlanGobiernoPerfilInvalido
	}
	return nil
}

// El recurso del cierre es VersionRol, no una persona ficticia. Publicar una
// definición no otorga ni migra una asignación, tampoco al propio administrador.
type SolicitudCierreGobiernoPerfil struct {
	OperacionRef            string
	PropuestaRef            string
	PropuestaHuellaSHA256   string
	ProponentePersonaRef    string
	VersionRolObjetivoRef   string
	Aprobador               ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Decision                DecisionPropuestaAdministracionPerfiles
	Motivo                  ReferenciaEntradaCatalogo
	CorrelacionRef          string
}

func (s SolicitudCierreGobiernoPerfil) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "cierre_admin:") ||
		!ReferenciaAdministracionPerfilesValida(s.PropuestaRef, "propuesta_admin:") ||
		!huellaSHA256AutorizacionV3NoNula(s.PropuestaHuellaSHA256) ||
		!referenciaOpacaAdministracionPerfiles(s.ProponentePersonaRef, "per_") ||
		!RolVersionAdministracionPerfilesValido(s.VersionRolObjetivoRef) || s.Aprobador.Validar() != nil ||
		s.Evidencia.ValidarPara(s.Aprobador) != nil || s.InstantaneaAutorizacion.Validar() != nil || !s.Decision.Valida() ||
		s.Aprobador.PersonaRef == s.ProponentePersonaRef || s.Aprobador.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Aprobador.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) || !ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrPlanGobiernoPerfilInvalido
	}
	return nil
}

type ReciboGobiernoPerfil struct {
	ActoRef             string
	ReciboRef           string
	ActorPersonaRef     string
	PerfilActivoRef     string
	AsignacionPerfilRef string
	CorrelacionRef      string
	Motivo              ReferenciaEntradaCatalogo
	AuditoriaRef        string
	VersionRol          VersionRol
	ControlPosterior    ControlVigenciaVersionRol
}

type CierreGobiernoPerfil struct {
	OperacionRef          string
	Material              MaterialPropuestaGobiernoPerfil
	PropuestaHuellaSHA256 string
	Decision              DecisionPropuestaAdministracionPerfiles
	ConfirmadoEn          time.Time
	AuditoriaAccesoRef    string
	Recibo                *ReciboGobiernoPerfil
}

func (c CierreGobiernoPerfil) ValidarPara(s SolicitudCierreGobiernoPerfil) error {
	h, err := c.Material.HuellaSHA256()
	if err != nil {
		return err
	}
	if s.Validar() != nil || h != s.PropuestaHuellaSHA256 || c.PropuestaHuellaSHA256 != h ||
		c.Material.OperacionRef != s.PropuestaRef || c.OperacionRef != s.OperacionRef ||
		c.Material.ProponentePersonaRef != s.ProponentePersonaRef || c.Material.Plan.VersionRolObjetivoRef != s.VersionRolObjetivoRef ||
		c.Decision != s.Decision || !instanteAutorizacionCanonico(c.ConfirmadoEn) ||
		!textoAutorizacionSinComodinSeguro(c.AuditoriaAccesoRef, 256, false) {
		return ErrPlanGobiernoPerfilInvalido
	}
	if c.Decision == DecisionRechazarPropuestaPerfil {
		if c.Recibo != nil {
			return ErrPlanGobiernoPerfilInvalido
		}
		return nil
	}
	r := c.Recibo
	if r == nil || !ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") || r.ActorPersonaRef != s.Aprobador.PersonaRef ||
		r.PerfilActivoRef != s.Aprobador.PerfilActivoRef || r.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		!ReferenciaCorrelacionAutorizacionV2Valida(r.CorrelacionRef) || r.Motivo != s.Motivo ||
		!textoAutorizacionSinComodinSeguro(r.AuditoriaRef, 256, false) || r.VersionRol.Validar() != nil ||
		r.VersionRol.Referencia() != s.VersionRolObjetivoRef || r.ControlPosterior.Validar() != nil ||
		r.ControlPosterior.VersionRolRef != s.VersionRolObjetivoRef || r.ControlPosterior.ActualizadoPor != s.Aprobador.PersonaRef ||
		!r.ControlPosterior.ActualizadoEn.Equal(c.ConfirmadoEn) {
		return ErrPlanGobiernoPerfilInvalido
	}
	p := c.Material.Plan
	if p.Base != nil && c.ConfirmadoEn.Before(p.Base.ControlVigencia.ActualizadoEn) {
		return ErrPlanGobiernoPerfilInvalido
	}
	if p.Operacion == OperacionDeshabilitarVersionPerfil {
		hr, er := r.VersionRol.HuellaSHA256()
		if er != nil {
			return er
		}
		hb, eb := p.Base.Rol.HuellaSHA256()
		if eb != nil {
			return eb
		}
		if hr != hb || r.ControlPosterior.Revision != p.Base.ControlVigencia.Revision+1 ||
			r.ControlPosterior.Estado != EstadoControlVigenciaVersionRolRetirada || r.ControlPosterior.ActoRef != r.ActoRef ||
			r.ControlPosterior.MotivoCodigo != s.Motivo.EntradaClave {
			return ErrPlanGobiernoPerfilInvalido
		}
	} else {
		d := p.DefinicionNueva
		if r.VersionRol.RolID != d.RolID || r.VersionRol.Version != d.Version || r.VersionRol.Nombre != d.Nombre ||
			r.VersionRol.Estado != EstadoVersionRolPublicada || r.VersionRol.PublicadaPor != s.Aprobador.PersonaRef ||
			!r.VersionRol.PublicadaEn.Equal(c.ConfirmadoEn) || len(r.VersionRol.Concesiones) != len(d.Concesiones) ||
			r.ControlPosterior.Revision != 1 || r.ControlPosterior.Estado != EstadoControlVigenciaVersionRolHabilitada {
			return ErrPlanGobiernoPerfilInvalido
		}
		for i := range d.Concesiones {
			if !concesionesPerfilAdministracionIguales(r.VersionRol.Concesiones[i], d.Concesiones[i]) {
				return ErrPlanGobiernoPerfilInvalido
			}
		}
	}
	return nil
}

func (r ReciboGobiernoPerfil) Copia() ReciboGobiernoPerfil {
	r.VersionRol.Concesiones = copiarConcesionesGobiernoPerfil(r.VersionRol.Concesiones)
	return r
}
