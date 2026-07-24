package ports

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionRegistrarAsignacion   = "contratacion_temporal.unidad.asignar"
	AccionRegistrarReasignacion = "contratacion_temporal.unidad.reasignar"
	TipoRecursoAsignacion       = "asignacion_contratacion_temporal"
)

var (
	ErrFlujoNoDisponible = errors.New(
		"contratacion temporal: flujo no disponible",
	)
	ErrDestinoAsignacionNoDisponible = errors.New(
		"contratacion temporal: destino de asignacion no disponible",
	)
	ErrPoliticaAsignacionNoDisponible = errors.New(
		"contratacion temporal: politica de asignacion no disponible",
	)
	ErrSegregacionAsignacionIncumplida = errors.New(
		"contratacion temporal: segregacion de asignacion incumplida",
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

type TipoOperacionAsignacion string

const (
	OperacionRegistrarAsignacion   TipoOperacionAsignacion = "asignar"
	OperacionRegistrarReasignacion TipoOperacionAsignacion = "reasignar"
)

func (t TipoOperacionAsignacion) Valida() bool {
	return t == OperacionRegistrarAsignacion ||
		t == OperacionRegistrarReasignacion
}

type MotivoReasignacionGobernado struct {
	ReferenciaCatalogo dominiovec.ReferenciaEntradaCatalogo
	ClaveMensajeI18N   domain.ClaveCatalogo
}

func (m MotivoReasignacionGobernado) ValidarPara(
	clave domain.ClaveCatalogo,
) error {
	if !clave.Valida() || m.ReferenciaCatalogo.Validar() != nil ||
		uint64(m.ReferenciaCatalogo.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		m.ReferenciaCatalogo.EntradaClave != string(clave) ||
		!m.ClaveMensajeI18N.Valida() ||
		!strings.HasPrefix(
			string(m.ClaveMensajeI18N),
			"contratacion_temporal.asignacion.motivo.",
		) {
		return ErrPoliticaAsignacionNoDisponible
	}
	return nil
}

type SolicitudResolverPoliticaAsignacion struct {
	Operacion               TipoOperacionAsignacion
	OrganizacionRef         string
	ExpedienteRef           string
	VersionExpediente       uint64
	Flujo                   domain.ReferenciaFlujo
	FasePrevia              domain.ClaveFase
	EstadoPrevio            domain.EstadoOperativo
	ActorRef                string
	PerfilRef               string
	UnidadAnteriorRef       string
	ResponsableAnteriorRef  string
	Destino                 DestinoAsignacionResuelto
	MotivoReasignacionClave domain.ClaveCatalogo
	Instante                time.Time
}

func (s SolicitudResolverPoliticaAsignacion) Validar() error {
	destinoSolicitado := SolicitudResolverDestinoAsignacion{
		OrganizacionRef:   s.OrganizacionRef,
		ExpedienteRef:     s.ExpedienteRef,
		VersionExpediente: s.VersionExpediente,
		ActorRef:          s.ActorRef,
		UnidadRef:         s.Destino.UnidadRef,
		ResponsableRef:    s.Destino.ResponsableRef,
		Instante:          s.Instante,
	}
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		s.Flujo.Validar() != nil || !s.FasePrevia.Valida() ||
		!s.EstadoPrevio.Valido() ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.InstanteUTCCanonico(s.Instante) ||
		s.Destino.ValidarPara(destinoSolicitado, s.Instante) != nil {
		return ErrPoliticaAsignacionNoDisponible
	}
	if s.Operacion == OperacionRegistrarAsignacion {
		if s.UnidadAnteriorRef != "" ||
			s.ResponsableAnteriorRef != "" ||
			s.MotivoReasignacionClave != "" {
			return ErrPoliticaAsignacionNoDisponible
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(s.UnidadAnteriorRef) ||
		!domain.ReferenciaOpacaValida(s.ResponsableAnteriorRef) ||
		!s.MotivoReasignacionClave.Valida() ||
		(s.UnidadAnteriorRef == s.Destino.UnidadRef &&
			s.ResponsableAnteriorRef == s.Destino.ResponsableRef) {
		return ErrPoliticaAsignacionNoDisponible
	}
	return nil
}

type PoliticaAsignacion struct {
	Operacion                     TipoOperacionAsignacion
	OrganizacionRef               string
	ExpedienteRef                 string
	VersionExpediente             uint64
	ActorRef                      string
	PerfilRef                     string
	DestinoEvidenciaRef           string
	DestinoEvidenciaHuellaSHA256  string
	DefinicionRef                 string
	DefinicionVersion             uint64
	DefinicionHuellaSHA256        string
	Accion                        domain.ClaveCatalogo
	Finalidad                     domain.ClaveCatalogo
	UnidadEjecutoraRef            string
	MotivoAutorizacion            dominiovec.ReferenciaEntradaCatalogo
	ExigeActorDistintoResponsable bool
	MotivoReasignacion            MotivoReasignacionGobernado
	EvaluadaEn                    time.Time
	ValidaHasta                   time.Time
}

func (p PoliticaAsignacion) ValidarPara(
	solicitud SolicitudResolverPoliticaAsignacion,
	instanteUso time.Time,
) error {
	if solicitud.Validar() != nil ||
		p.Operacion != solicitud.Operacion ||
		p.OrganizacionRef != solicitud.OrganizacionRef ||
		p.ExpedienteRef != solicitud.ExpedienteRef ||
		p.VersionExpediente != solicitud.VersionExpediente ||
		p.ActorRef != solicitud.ActorRef ||
		p.PerfilRef != solicitud.PerfilRef ||
		p.DestinoEvidenciaRef != solicitud.Destino.EvidenciaRef ||
		p.DestinoEvidenciaHuellaSHA256 !=
			solicitud.Destino.EvidenciaHuellaSHA256 ||
		!domain.ReferenciaOpacaValida(p.DefinicionRef) ||
		!VersionOperacionAnalisisValida(p.DefinicionVersion) ||
		!huellaSHA256OperacionAnalisisValida(p.DefinicionHuellaSHA256) ||
		!p.Accion.Valida() || !p.Finalidad.Valida() ||
		!domain.ReferenciaOpacaValida(p.UnidadEjecutoraRef) ||
		p.MotivoAutorizacion.Validar() != nil ||
		uint64(p.MotivoAutorizacion.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		!domain.InstanteUTCCanonico(p.EvaluadaEn) ||
		!domain.InstanteUTCCanonico(p.ValidaHasta) ||
		!domain.InstanteUTCCanonico(instanteUso) ||
		!p.EvaluadaEn.Equal(solicitud.Instante) ||
		!p.ValidaHasta.After(p.EvaluadaEn) ||
		instanteUso.Before(p.EvaluadaEn) ||
		instanteUso.After(p.ValidaHasta) ||
		(p.ExigeActorDistintoResponsable &&
			p.ActorRef == solicitud.Destino.ResponsableRef) {
		return ErrPoliticaAsignacionNoDisponible
	}
	if p.Operacion == OperacionRegistrarAsignacion {
		if p.Accion != domain.ClaveCatalogo(AccionRegistrarAsignacion) ||
			p.MotivoReasignacion != (MotivoReasignacionGobernado{}) {
			return ErrPoliticaAsignacionNoDisponible
		}
		return nil
	}
	if p.Accion != domain.ClaveCatalogo(AccionRegistrarReasignacion) ||
		p.MotivoReasignacion.ValidarPara(
			solicitud.MotivoReasignacionClave,
		) != nil {
		return ErrPoliticaAsignacionNoDisponible
	}
	return nil
}

type ResolutorPoliticaAsignacion interface {
	ResolverPoliticaAsignacion(
		context.Context,
		SolicitudResolverPoliticaAsignacion,
	) (PoliticaAsignacion, error)
}
