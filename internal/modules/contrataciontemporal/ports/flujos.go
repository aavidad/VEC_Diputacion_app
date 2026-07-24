package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrFlujoNoDisponible = errors.New(
		"contratacion temporal: flujo no disponible",
	)
	ErrDestinoAsignacionNoDisponible = errors.New(
		"contratacion temporal: destino de asignacion no disponible",
	)
)

type SolicitudResolverFlujo struct {
	OrganizacionRef string
	CentroRef       string
	CategoriaRef    string
	MotivoClave     domain.ClaveCatalogo
	Instante        time.Time
}

func (s SolicitudResolverFlujo) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.CentroRef) ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) ||
		!s.MotivoClave.Valida() || !domain.InstanteUTCCanonico(s.Instante) {
		return ErrFlujoNoDisponible
	}
	return nil
}

type ConfiguracionAltaFlujo struct {
	Flujo            domain.ReferenciaFlujo
	FaseInicial      domain.ClaveFase
	UnidadInicialRef string
	AccionInicial    domain.ClaveCatalogo
}

func (c ConfiguracionAltaFlujo) Validar() error {
	if c.Flujo.Validar() != nil || !c.FaseInicial.Valida() ||
		!domain.ReferenciaOpacaValida(c.UnidadInicialRef) ||
		!c.AccionInicial.Valida() {
		return ErrFlujoNoDisponible
	}
	return nil
}

type ResolutorFlujoAlta interface {
	ResolverFlujoAlta(context.Context, SolicitudResolverFlujo) (ConfiguracionAltaFlujo, error)
}

// SolicitudResolverDestinoAsignacion identifica las coordenadas mínimas que
// debe comprobar una fuente organizativa autoritativa antes de asignar un
// expediente. ActorRef procede del contexto de autenticación resuelto por el
// servidor; nunca se acepta directamente del cliente.
type SolicitudResolverDestinoAsignacion struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	ActorRef          string
	UnidadRef         string
	ResponsableRef    string
	Instante          time.Time
}

func (s SolicitudResolverDestinoAsignacion) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.UnidadRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableRef) ||
		!domain.InstanteUTCCanonico(s.Instante) {
		return ErrDestinoAsignacionNoDisponible
	}
	return nil
}

// DestinoAsignacionResuelto acredita una comprobación afirmativa de unidad y
// responsable. No contiene nombres, DNI, correo ni otros datos personales.
// Las referencias y la evidencia quedan ligadas a la solicitud exacta para
// impedir reutilizarlas en otro expediente o versión.
type DestinoAsignacionResuelto struct {
	OrganizacionRef        string
	ExpedienteRef          string
	VersionExpediente      uint64
	ActorRef               string
	UnidadRef              string
	ResponsableRef         string
	DefinicionRef          string
	DefinicionVersion      uint64
	DefinicionHuellaSHA256 string
	EvidenciaRef           string
	EvidenciaHuellaSHA256  string
	EvaluadoEn             time.Time
	ValidoHasta            time.Time
}

func (d DestinoAsignacionResuelto) ValidarPara(
	solicitud SolicitudResolverDestinoAsignacion,
	instanteUso time.Time,
) error {
	if solicitud.Validar() != nil ||
		d.OrganizacionRef != solicitud.OrganizacionRef ||
		d.ExpedienteRef != solicitud.ExpedienteRef ||
		d.VersionExpediente != solicitud.VersionExpediente ||
		d.ActorRef != solicitud.ActorRef ||
		d.UnidadRef != solicitud.UnidadRef ||
		d.ResponsableRef != solicitud.ResponsableRef ||
		!domain.ReferenciaOpacaValida(d.DefinicionRef) ||
		!VersionOperacionAnalisisValida(d.DefinicionVersion) ||
		!huellaSHA256OperacionAnalisisValida(d.DefinicionHuellaSHA256) ||
		!domain.ReferenciaOpacaValida(d.EvidenciaRef) ||
		!huellaSHA256OperacionAnalisisValida(d.EvidenciaHuellaSHA256) ||
		!domain.InstanteUTCCanonico(d.EvaluadoEn) ||
		!domain.InstanteUTCCanonico(d.ValidoHasta) ||
		!domain.InstanteUTCCanonico(instanteUso) ||
		!d.EvaluadoEn.Equal(solicitud.Instante) ||
		!d.ValidoHasta.After(d.EvaluadoEn) ||
		instanteUso.Before(d.EvaluadoEn) ||
		instanteUso.After(d.ValidoHasta) {
		return ErrDestinoAsignacionNoDisponible
	}
	return nil
}

// ResolutorDestinoAsignacion es un puerto de consulta. La implementación debe
// cruzar una fuente organizativa publicada con el directorio autoritativo y
// devolver error si la unidad no está activa o si la persona responsable no
// está activa, adscrita y habilitada para la tarea.
type ResolutorDestinoAsignacion interface {
	ResolverDestinoAsignacion(
		context.Context,
		SolicitudResolverDestinoAsignacion,
	) (DestinoAsignacionResuelto, error)
}
