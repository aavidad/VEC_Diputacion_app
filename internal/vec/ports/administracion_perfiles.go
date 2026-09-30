package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrAutoridadAdministracionPerfilesNoDisponible = errors.New("vec: autoridad de administracion de perfiles no disponible")

// RolAdministrable es una lectura del catalogo publicado, no una plantilla
// editable de concesiones. La autoridad durable coteja otra vez referencia,
// clase, huella y vigencia dentro de la transaccion que consume la decision.
type RolAdministrable struct {
	VersionRef   string
	Clase        domain.ClaseControlAdministracionPerfiles
	HuellaSHA256 string
	VigenteDesde time.Time
	VigenteHasta time.Time
}

func (r RolAdministrable) ValidarEn(instante time.Time) error {
	if !domain.RolVersionAdministracionPerfilesValido(r.VersionRef) || !r.Clase.Valida() ||
		!domain.HuellaAdministracionPerfilesValida(r.HuellaSHA256) ||
		r.VigenteDesde.IsZero() || r.VigenteHasta.IsZero() ||
		!r.VigenteHasta.After(r.VigenteDesde) || instante.IsZero() ||
		instante.Before(r.VigenteDesde) || !instante.Before(r.VigenteHasta) {
		return domain.ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

type CatalogoRolesAdministrables interface {
	// ResolverRolAdministrable consulta el catalogo autorizado. Nunca fabrica
	// un rol a partir de una peticion ni publica permisos nuevos.
	ResolverRolAdministrable(context.Context, string) (RolAdministrable, error)
}

type PropuestaAdministracionPerfiles struct {
	OperacionRef         string
	PropuestaRef         string
	HuellaSHA256         string
	ProponentePersonaRef string
	ObjetivoPersonaRef   string
	CaducaEn             time.Time
}

// CierrePropuestaAdministracionPerfiles distingue una aprobacion aplicada de
// un rechazo durable. Un rechazo conserva historia pero no tiene recibo de
// cambio de perfil.
type CierrePropuestaAdministracionPerfiles struct {
	OperacionRef       string
	PropuestaRef       string
	Decision           domain.DecisionPropuestaAdministracionPerfiles
	HuellaCierreSHA256 string
	ConfirmadoEn       time.Time
	Recibo             *domain.ReciboAdministracionPerfiles
}

func (c CierrePropuestaAdministracionPerfiles) ValidarPara(s domain.SolicitudCierrePropuestaAdministracionPerfiles) error {
	if s.Validar() != nil || c.OperacionRef != s.OperacionRef || c.PropuestaRef != s.PropuestaRef ||
		c.Decision != s.Decision || !domain.HuellaAdministracionPerfilesValida(c.HuellaCierreSHA256) ||
		c.ConfirmadoEn.IsZero() {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	if c.Decision == domain.DecisionRechazarPropuestaPerfil {
		if c.Recibo != nil {
			return domain.ErrControlAdministracionPerfilesInvalido
		}
		return nil
	}
	if c.Recibo == nil || c.Recibo.Validar() != nil || c.Recibo.OperacionRef != s.OperacionRef ||
		c.Recibo.PropuestaRef != s.PropuestaRef || !c.Recibo.ConfirmadoEn.Equal(c.ConfirmadoEn) {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

func (p PropuestaAdministracionPerfiles) ValidarPara(s domain.SolicitudActoAdministracionPerfiles) error {
	if s.Validar() != nil || !s.Clase.RequiereDobleControl() ||
		p.OperacionRef != s.OperacionRef || p.PropuestaRef != s.OperacionRef ||
		p.ProponentePersonaRef != s.Actor.PersonaRef ||
		p.ObjetivoPersonaRef != s.Objetivo.PersonaRef ||
		!domain.ReferenciaAdministracionPerfilesValida(p.PropuestaRef, "propuesta_admin:") ||
		!domain.HuellaAdministracionPerfilesValida(p.HuellaSHA256) || p.CaducaEn.IsZero() {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// AutoridadActosAdministracionPerfiles es un puerto indivisible. Ningun
// metodo admite una decision de permiso construida por HTTP. Su adaptador
// debe resolver el ContextoActor acreditado, revalidar la instantanea de
// autorizacion mediante el PDP V3 y consumir la autorizacion en la misma
// transaccion SERIALIZABLE que el CAS CA20/AUT24, la auditoria, historia y
// recibo. Debe comprobar que el aprobador sigue siendo administrador activo,
// distinto por persona del proponente y del afectado, y que revocar deja al
// menos dos personas administradoras efectivas. Intervencion usa el mismo
// doble control; su rol exacto sigue cerrado hasta publicarlo en catalogo.
// Una revocacion no puede reactivar el mismo perfil/vinculo historico.
//
// Las referencias de operacion son idempotentes: mismo contenido recupera el
// mismo recibo, contenido distinto falla; incluso al recuperar se revalida
// autorizacion vigente. Las propuestas sensibles no mutan el perfil.
type AutoridadActosAdministracionPerfiles interface {
	AplicarActoOrdinario(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error)
	ProponerActoSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (PropuestaAdministracionPerfiles, error)
	CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (CierrePropuestaAdministracionPerfiles, error)
}

// PreimagenBootstrapAdministracionPerfiles se obtiene de dos personas y
// cuentas ya acreditadas por la autoridad maestra. No contiene nombres,
// documento de identidad ni facultad de crear personas o cuentas.
type PreimagenBootstrapAdministracionPerfiles struct {
	Primera          domain.PreimagenAdministracionPerfiles
	Segunda          domain.PreimagenAdministracionPerfiles
	HuellaPlanSHA256 string
}

func (p PreimagenBootstrapAdministracionPerfiles) Validar() error {
	if p.Primera.ValidarBootstrap() != nil || p.Segunda.ValidarBootstrap() != nil ||
		p.Primera.PersonaRef == p.Segunda.PersonaRef || p.Primera.CuentaRef == p.Segunda.CuentaRef ||
		!domain.HuellaAdministracionPerfilesValida(p.HuellaPlanSHA256) {
		return domain.ErrControlAdministracionPerfilesInvalido
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
		!domain.ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!domain.ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		r.HuellaPlanSHA256 != p.HuellaPlanSHA256 ||
		r.PrimeraPersonaRef != p.Primera.PersonaRef ||
		r.SegundaPersonaRef != p.Segunda.PersonaRef || r.ConfirmadoEn.IsZero() {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// ProvisionadorBootstrapAdministracionPerfiles pertenece exclusivamente al
// canal de operador privado. La implementacion debe verificar aprobacion del
// operador contra la huella exacta, CAS de las dos personas acreditadas y
// contador inicial vacio, todo en una transaccion con auditoria y recibo.
// El origen de la aprobacion es configuracion privada, no un campo HTTP.
// No hay adaptador de HTTP ni implementacion en este corte.
type ProvisionadorBootstrapAdministracionPerfiles interface {
	ProvisionarDosAdministradoresIniciales(context.Context, PreimagenBootstrapAdministracionPerfiles) (ReciboBootstrapAdministracionPerfiles, error)
}
