package domain

import "time"

// RolAdministrable es una lectura del catalogo publicado, no una plantilla
// editable de concesiones. La autoridad durable coteja otra vez referencia,
// clase, huella y vigencia dentro de la transaccion que consume la decision.
type RolAdministrable struct {
	// UnidadRequerida procede del catálogo administrable publicado.
	UnidadRequerida bool
	// CategoriaAdmin procede del gobierno publicado (aplicacion/sistemas).
	// No se infiere de Clase, del nombre ni de roles declarados.
	CategoriaAdmin string
	VersionRef     string
	Clase          ClaseControlAdministracionPerfiles
	HuellaSHA256   string
	VigenteDesde   time.Time
	VigenteHasta   time.Time
}

func (r RolAdministrable) ValidarEn(instante time.Time) error {
	if !RolVersionAdministracionPerfilesValido(r.VersionRef) || !r.Clase.Valida() ||
		(r.Clase == ClaseControlPerfilAdministrador && r.CategoriaAdmin != "aplicacion" && r.CategoriaAdmin != "sistemas") ||
		(r.Clase != ClaseControlPerfilAdministrador && r.CategoriaAdmin != "") ||
		!HuellaAdministracionPerfilesValida(r.HuellaSHA256) ||
		r.VigenteDesde.IsZero() || r.VigenteHasta.IsZero() ||
		!r.VigenteHasta.After(r.VigenteDesde) || instante.IsZero() ||
		instante.Before(r.VigenteDesde) || !instante.Before(r.VigenteHasta) {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

type PropuestaAdministracionPerfiles struct {
	OperacionRef         string
	PropuestaRef         string
	HuellaSHA256         string
	ProponentePersonaRef string
	ObjetivoPersonaRef   string
	CaducaEn             time.Time
}

func (p PropuestaAdministracionPerfiles) ValidarPara(s SolicitudActoAdministracionPerfiles) error {
	if s.Validar() != nil || !s.Clase.RequiereDobleControl() ||
		p.OperacionRef != s.OperacionRef || p.PropuestaRef != s.OperacionRef ||
		p.ProponentePersonaRef != s.Actor.PersonaRef ||
		p.ObjetivoPersonaRef != s.Objetivo.PersonaRef ||
		!ReferenciaAdministracionPerfilesValida(p.PropuestaRef, "propuesta_admin:") ||
		!HuellaAdministracionPerfilesValida(p.HuellaSHA256) || p.CaducaEn.IsZero() {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// CierrePropuestaAdministracionPerfiles distingue una aprobacion aplicada de
// un rechazo durable. Un rechazo conserva historia pero no tiene recibo de
// cambio de perfil.
type CierrePropuestaAdministracionPerfiles struct {
	OperacionRef          string
	PropuestaHuellaSHA256 string
	PropuestaRef          string
	Decision              DecisionPropuestaAdministracionPerfiles
	HuellaCierreSHA256    string
	ConfirmadoEn          time.Time
	Recibo                *ReciboAdministracionPerfiles
}

func (c CierrePropuestaAdministracionPerfiles) ValidarPara(s SolicitudCierrePropuestaAdministracionPerfiles) error {
	if s.Validar() != nil || c.OperacionRef != s.OperacionRef || c.PropuestaRef != s.PropuestaRef ||
		c.PropuestaHuellaSHA256 != s.PropuestaHuellaSHA256 ||
		c.Decision != s.Decision || !HuellaAdministracionPerfilesValida(c.HuellaCierreSHA256) ||
		c.ConfirmadoEn.IsZero() {
		return ErrControlAdministracionPerfilesInvalido
	}
	if c.Decision == DecisionRechazarPropuestaPerfil {
		if c.Recibo != nil {
			return ErrControlAdministracionPerfilesInvalido
		}
		return nil
	}
	if c.Recibo == nil || c.Recibo.Validar() != nil || c.Recibo.OperacionRef != s.OperacionRef ||
		c.Recibo.PropuestaRef != s.PropuestaRef ||
		c.Recibo.ActorPersonaRef != s.Aprobador.PersonaRef ||
		c.Recibo.PerfilActivoRef != s.Aprobador.PerfilActivoRef ||
		c.Recibo.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		!ReferenciaCorrelacionAutorizacionV2Valida(c.Recibo.CorrelacionRef) || c.Recibo.Motivo != s.Motivo ||
		c.Recibo.ObjetivoPersonaRef != s.ObjetivoPersonaRef ||
		!c.Recibo.ConfirmadoEn.Equal(c.ConfirmadoEn) {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// PreimagenBootstrapAdministracionPerfiles se obtiene de dos personas y
// cuentas ya acreditadas por la autoridad maestra. No contiene nombres,
// documento de identidad ni facultad de crear personas o cuentas.
type PreimagenBootstrapAdministracionPerfiles struct {
	PlanV2           *PlanBootstrapAdministracionV2
	Primera          PreimagenAdministracionPerfiles
	Segunda          PreimagenAdministracionPerfiles
	HuellaPlanSHA256 string
}

func (p PreimagenBootstrapAdministracionPerfiles) Validar() error {
	if p.Primera.ValidarBootstrap() != nil || p.Segunda.ValidarBootstrap() != nil ||
		p.Primera.PersonaRef == p.Segunda.PersonaRef || p.Primera.CuentaRef == p.Segunda.CuentaRef ||
		!HuellaAdministracionPerfilesValida(p.HuellaPlanSHA256) {
		return ErrControlAdministracionPerfilesInvalido
	}
	if p.PlanV2 != nil {
		_, huella, err := p.PlanV2.CanonicoYHuella()
		if err != nil || huella != p.HuellaPlanSHA256 || p.PlanV2.Personas[0].Preimagen() != p.Primera || p.PlanV2.Personas[1].Preimagen() != p.Segunda {
			return ErrControlAdministracionPerfilesInvalido
		}
	}
	return nil
}

// ReciboBootstrapAdministracionPerfiles corresponde a una sola transaccion
// de arranque para dos personas. Los perfiles concretos se verifican contra
// el documento canonico del recibo por el adaptador que lo emite.
type ReciboBootstrapAdministracionPerfiles struct {
	ActoRef           string
	ReciboRef         string
	HuellaPlanSHA256  string
	PrimeraPersonaRef string
	SegundaPersonaRef string
	ConfirmadoEn      time.Time
}

func (r ReciboBootstrapAdministracionPerfiles) ValidarPara(p PreimagenBootstrapAdministracionPerfiles) error {
	if p.Validar() != nil ||
		!ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		r.HuellaPlanSHA256 != p.HuellaPlanSHA256 ||
		r.PrimeraPersonaRef != p.Primera.PersonaRef ||
		r.SegundaPersonaRef != p.Segunda.PersonaRef || r.ConfirmadoEn.IsZero() {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}
