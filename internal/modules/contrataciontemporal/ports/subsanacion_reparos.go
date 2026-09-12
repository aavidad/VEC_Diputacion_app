package ports

import (
	"context"
	"time"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const OperacionRegistrarSubsanacionReparo = "registrar_subsanacion"
const TipoRecursoSubsanacionReparo = "subsanacion_reparo_contratacion_temporal"
const FinalidadRegistrarSubsanacionReparo = "gestionar_contratacion_temporal"
const DominioAmbitoIdempotenciaSubsanacionReparo = "vec.contratacion-temporal.subsanacion-reparos.ambito"
const DominioHuellaPeticionSubsanacionReparo = "vec.contratacion-temporal.subsanacion-reparos.peticion"

// OrdenConfirmarSubsanacionReparo es el write-set que la composición durable
// debe confirmar en una transacción: expediente, actuación, auditoría y outbox.
// La adaptación durable debe revalidar la autorización vigente en esa misma
// transacción antes de hacer visible el recibo.
type OrdenConfirmarSubsanacionReparo struct {
	OrganizacionRef   string
	Expediente        domain.Expediente
	VersionAnterior   uint64
	ClaveIdempotencia string
	ActorRef          string
	UnidadRef         string
	ReciboRef         string
	EventoRef         string
	RegistradaEn      time.Time
	Evidencia         EvidenciaAutorizacionSubsanacionReparo
	Material          MaterialSubsanacionReparo
	Politica          PoliticaSubsanacionReparo
	Preparacion       PreparacionSubsanacionReparo
}

type SolicitudResolverPoliticaSubsanacionReparo struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef, RetornoRef string
	VersionEsperada                                                 uint64
	Instante                                                        time.Time
}
type PoliticaSubsanacionReparo struct {
	MotivoAutorizacion      vd.ReferenciaEntradaCatalogo
	DefinicionRef           string
	DefinicionVersion       uint64
	DefinicionHuellaSHA256  string
	EvaluadaEn, ValidaHasta time.Time
	Accion, Finalidad       domain.ClaveCatalogo
}

func (p PoliticaSubsanacionReparo) ValidaPara(s SolicitudResolverPoliticaSubsanacionReparo, instante time.Time) bool {
	return domain.ReferenciaOpacaValida(s.OrganizacionRef) && domain.ReferenciaOpacaValida(s.ExpedienteRef) && domain.ReferenciaOpacaValida(s.ActorRef) && domain.ReferenciaOpacaValida(s.PerfilRef) && domain.ReferenciaOpacaValida(s.RetornoRef) && s.VersionEsperada > 0 && domain.ReferenciaOpacaValida(p.DefinicionRef) && VersionOperacionAnalisisValida(p.DefinicionVersion) && huellaSHA256OperacionAnalisisValida(p.DefinicionHuellaSHA256) && p.MotivoAutorizacion.Validar() == nil && p.Accion == domain.AccionRegistrarSubsanacionReparo && p.Finalidad == domain.ClaveCatalogo(FinalidadRegistrarSubsanacionReparo) && p.EvaluadaEn.Equal(s.Instante) && p.ValidaHasta.After(p.EvaluadaEn) && !instante.Before(p.EvaluadaEn) && instante.Before(p.ValidaHasta)
}

type ResolutorPoliticaSubsanacionReparo interface {
	ResolverPoliticaSubsanacionReparo(context.Context, SolicitudResolverPoliticaSubsanacionReparo) (PoliticaSubsanacionReparo, error)
}
type EvidenciaAutorizacionSubsanacionReparo struct {
	Contexto       ContextoAutorizacionAltaV3
	SolicitudV3    vd.SolicitudAutorizacionLigadaV3
	DecisionV3     vd.DecisionAutorizacionLigadaV3
	ConfirmacionV3 vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

func (o OrdenConfirmarSubsanacionReparo) Validar() bool {
	if !domain.ReferenciaOpacaValida(o.OrganizacionRef) ||
		o.Expediente.Validar() != nil || o.VersionAnterior == 0 ||
		o.Expediente.Version != o.VersionAnterior+1 ||
		!o.Material.Valido() || o.Material.OrganizacionRef != o.OrganizacionRef || o.Material.ExpedienteRef != o.Expediente.Referencia || o.Material.ActorRef != o.ActorRef ||
		!ClaveIdempotenciaValida(o.ClaveIdempotencia) ||
		o.Material.ClaveIdempotencia != o.ClaveIdempotencia || o.Material.VersionEsperada != o.VersionAnterior ||
		!domain.ReferenciaOpacaValida(o.ActorRef) || !domain.ReferenciaOpacaValida(o.UnidadRef) ||
		!domain.ReferenciaOpacaValida(o.ReciboRef) || !domain.ReferenciaOpacaValida(o.EventoRef) ||
		!domain.InstanteUTCCanonico(o.RegistradaEn) {
		return false
	}
	if len(o.Expediente.Actuaciones) == 0 || o.Expediente.Fiscalizacion == nil || o.Expediente.Fiscalizacion.Retorno == nil {
		return false
	}
	ultima := o.Expediente.Actuaciones[len(o.Expediente.Actuaciones)-1]
	return ultima.AccionClave == domain.AccionRegistrarSubsanacionReparo && ultima.ActorRef == o.ActorRef && ultima.UnidadRef == o.UnidadRef && ultima.ReciboRef == o.ReciboRef && ultima.RealizadaEn.Equal(o.RegistradaEn) && ultima.RetornoRef == o.Expediente.Fiscalizacion.Retorno.RetornoRef && ultima.RetornoRef != "" && ultima.Observaciones == o.Material.Observaciones
}

type ReciboSubsanacionReparo struct {
	Operacion         string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionAnterior   uint64
	VersionResultante uint64
	FaseResultante    domain.ClaveFase
	EstadoResultante  domain.EstadoOperativo
	ReciboRef         string
	AuditoriaRef      string
	EventoRef         string
	ActorRef          string
	RegistradaEn      time.Time
}

func (r ReciboSubsanacionReparo) ValidarPara(o OrdenConfirmarSubsanacionReparo) bool {
	return o.Validar() && r.Operacion == OperacionRegistrarSubsanacionReparo &&
		r.OrganizacionRef == o.OrganizacionRef && r.ExpedienteRef == o.Expediente.Referencia &&
		r.VersionAnterior == o.VersionAnterior && r.VersionResultante == o.Expediente.Version &&
		r.FaseResultante == domain.FaseSubsanacionUnidad && r.EstadoResultante == domain.EstadoIncidencia &&
		r.ReciboRef == o.ReciboRef && r.EventoRef == o.EventoRef && r.ActorRef == o.ActorRef &&
		domain.ReferenciaOpacaValida(r.AuditoriaRef) && domain.InstanteUTCCanonico(r.RegistradaEn) &&
		r.RegistradaEn.Equal(o.RegistradaEn)
}

type PreparacionSubsanacionReparo struct {
	Material               MaterialSubsanacionReparo
	Expediente             domain.Expediente
	RetornoRef             string
	ReciboRef              string
	EventoRef              string
	Confirmada             bool
	ReciboConfirmado       *ReciboSubsanacionReparo
	ReservaRef             string
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
}

type MaterialSubsanacionReparo struct {
	OrganizacionRef, ExpedienteRef, ClaveIdempotencia, Observaciones, ActorRef, PerfilRef string
	VersionEsperada                                                                       uint64
}

type ReferenciasEfectoSubsanacionReparo struct{ ReservaRef, ReciboRef, EventoRef string }

func (r ReferenciasEfectoSubsanacionReparo) Validas() bool {
	return domain.ReferenciaOpacaValida(r.ReservaRef) && domain.ReferenciaOpacaValida(r.ReciboRef) && domain.ReferenciaOpacaValida(r.EventoRef)
}

type GeneradorReferenciasSubsanacionReparo interface {
	GenerarReferenciasSubsanacionReparo(context.Context) (ReferenciasEfectoSubsanacionReparo, error)
}
type SelladorAmbitoSubsanacionReparo interface {
	SellarAmbitoSubsanacionReparo(context.Context, SolicitudSellarAmbitoIdempotencia) (ColeccionSellosHMAC, error)
}
type DerivadorHuellaSubsanacionReparo interface {
	DerivarHuellaSubsanacionReparo(context.Context, MaterialSubsanacionReparo) (ColeccionSellosHMAC, error)
}

func (m MaterialSubsanacionReparo) Valido() bool {
	return domain.ReferenciaOpacaValida(m.OrganizacionRef) && domain.ReferenciaOpacaValida(m.ExpedienteRef) && ClaveIdempotenciaValida(m.ClaveIdempotencia) && m.VersionEsperada > 0 && domain.ReferenciaOpacaValida(m.ActorRef) && domain.ReferenciaOpacaValida(m.PerfilRef) && textoFiscalizacionValido(m.Observaciones, false)
}

type SolicitudPrepararSubsanacionReparo struct {
	Material                         MaterialSubsanacionReparo
	AmbitosHMAC, HuellasPeticionHMAC ColeccionSellosHMAC
}
type PreparadorSubsanacionReparo interface {
	PrepararSubsanacionReparo(context.Context, SolicitudPrepararSubsanacionReparo) (PreparacionSubsanacionReparo, error)
}

type ConfirmadorSubsanacionReparo interface {
	ConfirmarSubsanacionReparo(context.Context, OrdenConfirmarSubsanacionReparo) (ReciboSubsanacionReparo, error)
}
