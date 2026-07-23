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
	AccionRegistrarAnalisis  = "contratacion_temporal.analisis.registrar"
	AccionRectificarAnalisis = "contratacion_temporal.analisis.rectificar"
	TipoRecursoAnalisis      = "analisis_contratacion_temporal"
)

var (
	ErrOperacionAnalisisInvalida = errors.New(
		"contratacion temporal: operacion de analisis invalida",
	)
	ErrPoliticaOperacionAnalisisNoDisponible = errors.New(
		"contratacion temporal: politica de operacion de analisis no disponible",
	)
	ErrSegregacionOperacionAnalisisIncumplida = errors.New(
		"contratacion temporal: segregacion de operacion de analisis incumplida",
	)
)

type TipoOperacionAnalisis string

const (
	OperacionRegistrarAnalisis  TipoOperacionAnalisis = "registrar"
	OperacionRectificarAnalisis TipoOperacionAnalisis = "rectificar"
)

func (t TipoOperacionAnalisis) Valida() bool {
	return t == OperacionRegistrarAnalisis ||
		t == OperacionRectificarAnalisis
}

type MotivoRectificacionGobernado struct {
	ReferenciaCatalogo dominiovec.ReferenciaEntradaCatalogo
	ClaveMensajeI18N   domain.ClaveCatalogo
}

func (m MotivoRectificacionGobernado) ValidarPara(
	clave domain.ClaveCatalogo,
) error {
	if !clave.Valida() || m.ReferenciaCatalogo.Validar() != nil ||
		uint64(m.ReferenciaCatalogo.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		m.ReferenciaCatalogo.EntradaClave != string(clave) ||
		!m.ClaveMensajeI18N.Valida() ||
		!strings.HasPrefix(
			string(m.ClaveMensajeI18N),
			"contratacion_temporal.analisis.rectificacion.",
		) {
		return ErrPoliticaOperacionAnalisisNoDisponible
	}
	return nil
}

type SolicitudResolverPoliticaOperacionAnalisis struct {
	Operacion                TipoOperacionAnalisis
	OrganizacionRef          string
	ExpedienteRef            string
	VersionExpediente        uint64
	Flujo                    domain.ReferenciaFlujo
	FasePrevia               domain.ClaveFase
	EstadoPrevio             domain.EstadoOperativo
	ActorRef                 string
	PerfilRef                string
	ActorAnalisisAnteriorRef string
	ArtefactoRef             string
	ArtefactoHuellaSHA256    string
	MotivoRectificacionClave domain.ClaveCatalogo
	Instante                 time.Time
}

func (s SolicitudResolverPoliticaOperacionAnalisis) Validar() error {
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		s.Flujo.Validar() != nil ||
		!VersionOperacionAnalisisValida(s.Flujo.Version) ||
		!s.FasePrevia.Valida() || !s.EstadoPrevio.Valido() ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.ReferenciaOpacaValida(s.ArtefactoRef) ||
		!huellaSHA256OperacionAnalisisValida(s.ArtefactoHuellaSHA256) ||
		!instanteSeguroOperacionAnalisis(s.Instante) {
		return ErrPoliticaOperacionAnalisisNoDisponible
	}
	if s.Operacion == OperacionRegistrarAnalisis {
		if s.ActorAnalisisAnteriorRef != "" ||
			s.MotivoRectificacionClave != "" {
			return ErrPoliticaOperacionAnalisisNoDisponible
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(s.ActorAnalisisAnteriorRef) ||
		!s.MotivoRectificacionClave.Valida() {
		return ErrPoliticaOperacionAnalisisNoDisponible
	}
	return nil
}

type PoliticaOperacionAnalisis struct {
	Operacion                TipoOperacionAnalisis
	OrganizacionRef          string
	ExpedienteRef            string
	VersionExpediente        uint64
	FasePrevia               domain.ClaveFase
	EstadoPrevio             domain.EstadoOperativo
	ActorRef                 string
	ActorAnalisisAnteriorRef string
	ArtefactoRef             string
	ArtefactoHuellaSHA256    string
	DefinicionRef            string
	Version                  uint64
	HuellaSHA256             string
	Accion                   domain.ClaveCatalogo
	Finalidad                domain.ClaveCatalogo
	UnidadRef                string
	MotivoAutorizacion       dominiovec.ReferenciaEntradaCatalogo
	ExigeActorDistinto       bool
	MotivoRectificacion      MotivoRectificacionGobernado
	EvaluadaEn               time.Time
}

func (p PoliticaOperacionAnalisis) ValidarPara(
	solicitud SolicitudResolverPoliticaOperacionAnalisis,
) error {
	if solicitud.Validar() != nil ||
		p.Operacion != solicitud.Operacion ||
		p.OrganizacionRef != solicitud.OrganizacionRef ||
		p.ExpedienteRef != solicitud.ExpedienteRef ||
		p.VersionExpediente != solicitud.VersionExpediente ||
		p.FasePrevia != solicitud.FasePrevia ||
		p.EstadoPrevio != solicitud.EstadoPrevio ||
		p.ActorRef != solicitud.ActorRef ||
		p.ActorAnalisisAnteriorRef !=
			solicitud.ActorAnalisisAnteriorRef ||
		p.ArtefactoRef != solicitud.ArtefactoRef ||
		p.ArtefactoHuellaSHA256 != solicitud.ArtefactoHuellaSHA256 ||
		!domain.ReferenciaOpacaValida(p.DefinicionRef) ||
		!VersionOperacionAnalisisValida(p.Version) ||
		!huellaSHA256OperacionAnalisisValida(p.HuellaSHA256) ||
		!p.Accion.Valida() || !p.Finalidad.Valida() ||
		!domain.ReferenciaOpacaValida(p.UnidadRef) ||
		p.MotivoAutorizacion.Validar() != nil ||
		uint64(p.MotivoAutorizacion.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		!p.EvaluadaEn.Equal(solicitud.Instante) {
		return ErrPoliticaOperacionAnalisisNoDisponible
	}
	if p.Operacion == OperacionRegistrarAnalisis {
		if p.Accion != domain.ClaveCatalogo(AccionRegistrarAnalisis) ||
			p.ExigeActorDistinto ||
			p.MotivoRectificacion != (MotivoRectificacionGobernado{}) {
			return ErrPoliticaOperacionAnalisisNoDisponible
		}
		return nil
	}
	if p.Accion != domain.ClaveCatalogo(AccionRectificarAnalisis) ||
		p.MotivoRectificacion.ValidarPara(
			solicitud.MotivoRectificacionClave,
		) != nil {
		return ErrPoliticaOperacionAnalisisNoDisponible
	}
	return nil
}

type ResolutorPoliticaOperacionAnalisis interface {
	ResolverPoliticaOperacionAnalisis(
		context.Context,
		SolicitudResolverPoliticaOperacionAnalisis,
	) (PoliticaOperacionAnalisis, error)
}
