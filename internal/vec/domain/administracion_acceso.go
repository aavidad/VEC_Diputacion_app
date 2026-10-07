package domain

import (
	"errors"
	"math"
	"time"
)

var ErrAdministracionAccesoInvalida = errors.New("admin_acceso_invalido")

// EstadoCuentaAdministracionAcceso conserva los estados de la autoridad
// identidad_sesiones_v1.estado_cuenta. No es otro estado de persona o perfil.
type EstadoCuentaAdministracionAcceso string

const (
	EstadoCuentaAccesoActiva   EstadoCuentaAdministracionAcceso = "activa"
	EstadoCuentaAccesoInactiva EstadoCuentaAdministracionAcceso = "inactiva"
)

func (e EstadoCuentaAdministracionAcceso) Valido() bool {
	return e == EstadoCuentaAccesoActiva || e == EstadoCuentaAccesoInactiva
}

type PreimagenCuentaAdministracionAcceso struct {
	CuentaRef string
	Revision  uint64
	Estado    EstadoCuentaAdministracionAcceso
}

// PreimagenConjuntoCuentasAdministracionAcceso procede exclusivamente de una
// lectura central autorizada. Incluye TODAS las cuentas de la persona, de
// cualquier superficie, también las inactivas y privilegiadas, sin truncar.
// PersonaVersion no se presenta como revisión de agrupación: las altas de
// cuenta IS2 no demuestran que la incrementen. La autoridad durable debe
// proteger la composición del conjunto por su barrera central hasta COMMIT.
type PreimagenConjuntoCuentasAdministracionAcceso struct {
	PersonaRef          string
	PersonaVersion      uint64
	RevisionContinuidad uint64
	Cuentas             []PreimagenCuentaAdministracionAcceso
}

func (p PreimagenConjuntoCuentasAdministracionAcceso) Validar() error {
	if !referenciaOpacaAdministracionPerfiles(p.PersonaRef, "per_") || p.PersonaVersion == 0 ||
		p.RevisionContinuidad == 0 || len(p.Cuentas) == 0 || len(p.Cuentas) > maximoElementosAutorizacion {
		return ErrAdministracionAccesoInvalida
	}
	for i, c := range p.Cuentas {
		if !referenciaOpacaAdministracionPerfiles(c.CuentaRef, "cta_") || c.Revision == 0 || !c.Estado.Valido() ||
			(i > 0 && p.Cuentas[i-1].CuentaRef >= c.CuentaRef) {
			return ErrAdministracionAccesoInvalida
		}
	}
	return nil
}

func (p PreimagenConjuntoCuentasAdministracionAcceso) HuellaSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrAdministracionAccesoInvalida
	}
	return huellaAutorizacion(p)
}

// SolicitudPropuestaAdministracionAcceso selecciona una persona, nunca una
// lista de cuentas ni una clase de permiso. Contexto, evidencia y asignación
// son internos; la frontera no los reconstruye desde JSON del cliente.
type SolicitudPropuestaAdministracionAcceso struct {
	OperacionRef            string
	Actor                   ContextoActor                         `json:"-"`
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion               `json:"-"`
	ObjetivoPersonaRef      string
	EstadoNuevo             EstadoCuentaAdministracionAcceso
	Motivo                  ReferenciaEntradaCatalogo
	ReferenciaActo          string
	CorrelacionRef          string
}

func (s SolicitudPropuestaAdministracionAcceso) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "propuesta_admin:") ||
		s.Actor.Validar() != nil || s.Evidencia.ValidarPara(s.Actor) != nil || s.InstantaneaAutorizacion.Validar() != nil ||
		!referenciaOpacaAdministracionPerfiles(s.ObjetivoPersonaRef, "per_") || !s.EstadoNuevo.Valido() ||
		s.Actor.PersonaRef == s.ObjetivoPersonaRef ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) || !ReferenciaActoAdministracionValida(s.ReferenciaActo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		return ErrAdministracionAccesoInvalida
	}
	return nil
}

// MaterialPropuestaAdministracionAcceso es el material inmutable del acto.
// Su huella excluye evidencia, autorización y correlación del acceso actual:
// un replay exige revalidarlas, sin reescribir la propuesta o el recibo.
type MaterialPropuestaAdministracionAcceso struct {
	OperacionRef         string
	ProponentePersonaRef string
	PerfilActivoRef      string
	AsignacionPerfilRef  string
	Conjunto             PreimagenConjuntoCuentasAdministracionAcceso
	EstadoNuevo          EstadoCuentaAdministracionAcceso
	Motivo               ReferenciaEntradaCatalogo
	ReferenciaActo       string
}

func (m MaterialPropuestaAdministracionAcceso) Validar() error {
	if !ReferenciaAdministracionPerfilesValida(m.OperacionRef, "propuesta_admin:") ||
		!referenciaOpacaAdministracionPerfiles(m.ProponentePersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(m.PerfilActivoRef, "prf_") || m.Conjunto.Validar() != nil ||
		!textoAutorizacionSinComodinSeguro(m.AsignacionPerfilRef, 512, false) ||
		m.ProponentePersonaRef == m.Conjunto.PersonaRef || !m.EstadoNuevo.Valido() ||
		!ReferenciaMotivoAutorizacionV2Valida(m.Motivo) || !ReferenciaActoAdministracionValida(m.ReferenciaActo) {
		return ErrAdministracionAccesoInvalida
	}
	cambios := 0
	for _, c := range m.Conjunto.Cuentas {
		if c.Estado != m.EstadoNuevo {
			if c.Revision == math.MaxUint64 {
				return ErrAdministracionAccesoInvalida
			}
			cambios++
		}
	}
	if cambios == 0 {
		return ErrAdministracionAccesoInvalida
	}
	return nil
}

func (m MaterialPropuestaAdministracionAcceso) HuellaSHA256() (string, error) {
	if m.Validar() != nil {
		return "", ErrAdministracionAccesoInvalida
	}
	return huellaAutorizacion(m)
}

type OrdenPropuestaAdministracionAcceso struct {
	Solicitud SolicitudPropuestaAdministracionAcceso
	Material  MaterialPropuestaAdministracionAcceso
}

func (o OrdenPropuestaAdministracionAcceso) Validar() error {
	exacta, err := PrepararPropuestaAdministracionAcceso(o.Solicitud, o.Material.Conjunto)
	if err != nil {
		return ErrAdministracionAccesoInvalida
	}
	h, err := o.Material.HuellaSHA256()
	he, erre := exacta.Material.HuellaSHA256()
	if err != nil || erre != nil || h != he {
		return ErrAdministracionAccesoInvalida
	}
	return nil
}

func PrepararPropuestaAdministracionAcceso(s SolicitudPropuestaAdministracionAcceso, p PreimagenConjuntoCuentasAdministracionAcceso) (OrdenPropuestaAdministracionAcceso, error) {
	p.Cuentas = append([]PreimagenCuentaAdministracionAcceso(nil), p.Cuentas...)
	m := MaterialPropuestaAdministracionAcceso{OperacionRef: s.OperacionRef, ProponentePersonaRef: s.Actor.PersonaRef,
		PerfilActivoRef: s.Actor.PerfilActivoRef, AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Conjunto: p, EstadoNuevo: s.EstadoNuevo,
		Motivo: s.Motivo, ReferenciaActo: s.ReferenciaActo}
	if s.Validar() != nil || m.Validar() != nil || p.PersonaRef != s.ObjetivoPersonaRef {
		return OrdenPropuestaAdministracionAcceso{}, ErrAdministracionAccesoInvalida
	}
	return OrdenPropuestaAdministracionAcceso{Solicitud: s, Material: m}, nil
}

type PropuestaAdministracionAcceso struct {
	Material     MaterialPropuestaAdministracionAcceso
	HuellaSHA256 string
	CaducaEn     time.Time
}

func (p PropuestaAdministracionAcceso) ValidarPara(o OrdenPropuestaAdministracionAcceso) error {
	h, err := o.Material.HuellaSHA256()
	hp, errp := p.Material.HuellaSHA256()
	if o.Validar() != nil || err != nil || errp != nil || h != hp || p.HuellaSHA256 != h ||
		!instanteContextoActorCanonico(p.CaducaEn) {
		return ErrAdministracionAccesoInvalida
	}
	return nil
}

type ResultadoCuentaAdministracionAcceso struct {
	CuentaRef         string
	RevisionPosterior uint64
	EstadoPosterior   EstadoCuentaAdministracionAcceso
	OperacionISRef    string
}

type ReciboAdministracionAcceso struct {
	ReciboRef           string
	ActorPersonaRef     string
	PerfilActivoRef     string
	AsignacionPerfilRef string
	CorrelacionRef      string
	Motivo              ReferenciaEntradaCatalogo
	AuditoriaRef        string
	Cuentas             []ResultadoCuentaAdministracionAcceso
}

// CierreAdministracionAcceso recupera el material almacenado por la fuente.
// Ninguna cuenta o decisión de clasificación procede del solicitante del cierre.
type CierreAdministracionAcceso struct {
	OperacionRef          string
	Material              MaterialPropuestaAdministracionAcceso
	PropuestaHuellaSHA256 string
	Decision              DecisionPropuestaAdministracionPerfiles
	ConfirmadoEn          time.Time
	AuditoriaAccesoRef    string
	Recibo                *ReciboAdministracionAcceso
}

func (c CierreAdministracionAcceso) ValidarPara(s SolicitudCierrePropuestaAdministracionPerfiles) error {
	h, err := c.Material.HuellaSHA256()
	if s.Validar() != nil || !ReferenciaMotivoAutorizacionV2Valida(s.Motivo) || err != nil ||
		c.OperacionRef != s.OperacionRef || c.Material.OperacionRef != s.PropuestaRef ||
		c.PropuestaHuellaSHA256 != s.PropuestaHuellaSHA256 || h != s.PropuestaHuellaSHA256 ||
		c.Material.ProponentePersonaRef != s.ProponentePersonaRef || c.Material.Conjunto.PersonaRef != s.ObjetivoPersonaRef ||
		c.Decision != s.Decision || !instanteContextoActorCanonico(c.ConfirmadoEn) ||
		!textoAutorizacionSinComodinSeguro(c.AuditoriaAccesoRef, 256, false) {
		return ErrAdministracionAccesoInvalida
	}
	if c.Decision == DecisionRechazarPropuestaPerfil {
		if c.Recibo != nil {
			return ErrAdministracionAccesoInvalida
		}
		return nil
	}
	r := c.Recibo
	if r == nil || !ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		r.ActorPersonaRef != s.Aprobador.PersonaRef || r.PerfilActivoRef != s.Aprobador.PerfilActivoRef ||
		r.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		!ReferenciaCorrelacionAutorizacionV2Valida(r.CorrelacionRef) || r.Motivo != s.Motivo ||
		!textoAutorizacionSinComodinSeguro(r.AuditoriaRef, 256, false) || len(r.Cuentas) != len(c.Material.Conjunto.Cuentas) {
		return ErrAdministracionAccesoInvalida
	}
	operacionesIS := make(map[string]bool, len(r.Cuentas))
	for i, resultado := range r.Cuentas {
		antes := c.Material.Conjunto.Cuentas[i]
		if resultado.CuentaRef != antes.CuentaRef || resultado.EstadoPosterior != c.Material.EstadoNuevo {
			return ErrAdministracionAccesoInvalida
		}
		if antes.Estado == c.Material.EstadoNuevo {
			if resultado.RevisionPosterior != antes.Revision || resultado.OperacionISRef != "" {
				return ErrAdministracionAccesoInvalida
			}
		} else if resultado.RevisionPosterior != antes.Revision+1 || !referenciaOpacaAdministracionPerfiles(resultado.OperacionISRef, "opr_") {
			return ErrAdministracionAccesoInvalida
		} else {
			if operacionesIS[resultado.OperacionISRef] {
				return ErrAdministracionAccesoInvalida
			}
			operacionesIS[resultado.OperacionISRef] = true
		}
	}
	return nil
}
