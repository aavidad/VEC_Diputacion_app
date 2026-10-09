package domain

import "time"

// SolicitudPropuestaVersionInscripcion recibe el plan desde ADMIN, pero la
// identidad, evidencia de sesión y asignación proceden de la frontera segura.
// La fuente publicada se vuelve a resolver en la autoridad PostgreSQL.
type SolicitudPropuestaVersionInscripcion struct {
	OperacionRef            string
	Actor                   ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	Plan                    PlanVersionInscripcion
	CorrelacionRef          string
}

func (s SolicitudPropuestaVersionInscripcion) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "propuesta_admin:") ||
		s.Actor.Validar() != nil || s.Evidencia.ValidarPara(s.Actor) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		s.Plan.ValidarEstructura() != nil || !ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrPlanVersionInscripcionInvalido
	}
	return nil
}

// Sin etiquetas JSON: AUT66 conserva exactamente estas cinco claves PascalCase
// y su orden de json.Marshal para la huella del material.
type MaterialPropuestaVersionInscripcion struct {
	OperacionRef         string
	ProponentePersonaRef string
	PerfilActivoRef      string
	AsignacionPerfilRef  string
	Plan                 PlanVersionInscripcion
}

func (m MaterialPropuestaVersionInscripcion) HuellaSHA256() (string, error) {
	if !ReferenciaAdministracionPerfilesValida(m.OperacionRef, "propuesta_admin:") ||
		!referenciaOpacaAdministracionPerfiles(m.ProponentePersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(m.PerfilActivoRef, "prf_") ||
		!textoAutorizacionSinComodinSeguro(m.AsignacionPerfilRef, 512, false) ||
		m.Plan.ValidarEstructura() != nil {
		return "", ErrPlanVersionInscripcionInvalido
	}
	return huellaAutorizacion(m)
}

type OrdenPropuestaVersionInscripcion struct {
	Solicitud SolicitudPropuestaVersionInscripcion
	Material  MaterialPropuestaVersionInscripcion
}

func (o OrdenPropuestaVersionInscripcion) Validar() error {
	planSHA, err := o.Solicitud.Plan.HuellaSHA256()
	materialSHA, errMaterial := o.Material.Plan.HuellaSHA256()
	if err != nil || errMaterial != nil || planSHA != materialSHA ||
		o.Solicitud.Validar() != nil || o.Material.OperacionRef != o.Solicitud.OperacionRef ||
		o.Material.ProponentePersonaRef != o.Solicitud.Actor.PersonaRef ||
		o.Material.PerfilActivoRef != o.Solicitud.Actor.PerfilActivoRef ||
		o.Material.AsignacionPerfilRef != o.Solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia() {
		return ErrPlanVersionInscripcionInvalido
	}
	return nil
}

type PropuestaVersionInscripcion struct {
	Material           MaterialPropuestaVersionInscripcion
	HuellaSHA256       string
	CaducaEn           time.Time
	AuditoriaAccesoRef string
}

func (p PropuestaVersionInscripcion) ValidarPara(o OrdenPropuestaVersionInscripcion) error {
	h, err := o.Material.HuellaSHA256()
	hp, ep := p.Material.HuellaSHA256()
	if o.Validar() != nil || err != nil || ep != nil || h != hp || p.HuellaSHA256 != h ||
		!instanteAutorizacionCanonico(p.CaducaEn) ||
		!textoAutorizacionSinComodinSeguro(p.AuditoriaAccesoRef, 256, false) {
		return ErrPlanVersionInscripcionInvalido
	}
	return nil
}

// El cierre transporta sólo referencia y huella de la propuesta. AUT68 carga
// el material original inmutable y comprueba separación de dos ADMIN por
// Persona antes de publicar la versión y mover cada asignación seleccionada.
type SolicitudCierreVersionInscripcion struct {
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

func (s SolicitudCierreVersionInscripcion) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "cierre_admin:") ||
		!ReferenciaAdministracionPerfilesValida(s.PropuestaRef, "propuesta_admin:") ||
		!huellaSHA256AutorizacionV3NoNula(s.PropuestaHuellaSHA256) ||
		s.Aprobador.Validar() != nil || s.Evidencia.ValidarPara(s.Aprobador) != nil ||
		s.InstantaneaAutorizacion.Validar() != nil || s.Decision != DecisionAprobarPropuestaPerfil ||
		s.Aprobador.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Aprobador.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrPlanVersionInscripcionInvalido
	}
	return nil
}
