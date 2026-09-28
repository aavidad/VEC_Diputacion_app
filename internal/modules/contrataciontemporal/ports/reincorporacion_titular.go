package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	OperacionRegistrarReincorporacionTitular = "registrar_reincorporacion_titular"
	TipoRecursoReincorporacionTitular        = "reincorporacion_titular_contratacion_temporal"
	FinalidadRegistrarReincorporacionTitular = "registrar_reincorporacion_titular"
	AudienciaConsumoReincorporacionTitularV1 = "vec_contratacion_temporal.reincorporacion_titular.v1"
	DominioAmbitoReincorporacionTitular      = "vec.contratacion-temporal.reincorporacion-titular.ambito"
	DominioHuellaReincorporacionTitular      = "vec.contratacion-temporal.reincorporacion-titular.peticion"
)

var (
	ErrReincorporacionSinCese        = errors.New("contratacion temporal: reincorporacion sin cese")
	ErrReincorporacionCeseNoCoincide = errors.New("contratacion temporal: cese y reincorporacion no coinciden")
	ErrReincorporacionYaRegistrada   = errors.New("contratacion temporal: reincorporacion ya registrada")
)

type MaterialReincorporacionTitular struct {
	OrganizacionRef, ExpedienteRef, RelacionRef, ActorRef, PerfilRef string
	FechaEfectiva                                                    time.Time
	DocumentoRef, DocumentoSHA256, ClaveIdempotencia                 string
	VersionEsperada                                                  uint64
}

func (m MaterialReincorporacionTitular) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		(domain.DatosReincorporacionTitular{RelacionRef: m.RelacionRef, FechaEfectiva: m.FechaEfectiva, DocumentoRef: m.DocumentoRef, DocumentoSHA256: m.DocumentoSHA256}).Validar() == nil
}

type ReciboReincorporacionTitular struct {
	Operacion, OrganizacionRef, ExpedienteRef, RelacionRef, FechaEfectiva string
	CeseEventoRef, CeseReciboRef                                          string
	VersionAnterior, VersionResultante                                    uint64
	ReciboRef, AuditoriaRef, EventoRef, ActorRef                          string
	RegistradaEn                                                          time.Time
}

func (r ReciboReincorporacionTitular) ValidoPara(m MaterialReincorporacionTitular) bool {
	return r.Operacion == OperacionRegistrarReincorporacionTitular && r.OrganizacionRef == m.OrganizacionRef &&
		r.ExpedienteRef == m.ExpedienteRef && r.RelacionRef == m.RelacionRef &&
		r.FechaEfectiva == m.FechaEfectiva.Format(time.DateOnly) && r.ActorRef == m.ActorRef &&
		r.VersionAnterior == m.VersionEsperada && r.VersionResultante == m.VersionEsperada+1 &&
		domain.ReferenciaOpacaValida(r.CeseEventoRef) && domain.ReferenciaOpacaValida(r.CeseReciboRef) &&
		domain.ReferenciaOpacaValida(r.ReciboRef) && domain.ReferenciaOpacaValida(r.AuditoriaRef) &&
		domain.ReferenciaOpacaValida(r.EventoRef) && domain.InstanteUTCCanonico(r.RegistradaEn)
}

type PreparacionReincorporacionTitular struct {
	Expediente                                 domain.Expediente
	Referencias                                ReferenciasEfectoSeguimiento
	AmbitoIdempotenciaHMAC, HuellaPeticionHMAC string
	CeseEventoRef, CeseReciboRef               string
	Confirmada                                 bool
	Recibo                                     *ReciboReincorporacionTitular
}

type OrdenConfirmarReincorporacionTitular struct {
	Material       MaterialReincorporacionTitular
	Lectura        AntecedenteReincorporacionTitular
	Preparacion    PreparacionReincorporacionTitular
	Siguiente      domain.Expediente
	Politica       PoliticaOperacionSeguimiento
	Contexto       ContextoAutorizadoSeguimiento
	Autorizacion   ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	InstanteEfecto time.Time
}

type RepositorioReincorporacionTitular interface {
	PrepararReincorporacionTitular(context.Context, MaterialReincorporacionTitular, AntecedenteReincorporacionTitular, SellosOperacionSeguimiento, ReferenciasEfectoSeguimiento) (PreparacionReincorporacionTitular, error)
	ConfirmarReincorporacionTitular(context.Context, OrdenConfirmarReincorporacionTitular) (ReciboReincorporacionTitular, error)
}

// AutorizadorLecturaReincorporacionTitular exige la lectura V3 del expediente
// exacto antes de que la preparación durable revele su estado o antecedentes.
// La autorización de lectura no sustituye la decisión V3 de escritura.
type AutorizadorLecturaReincorporacionTitular interface {
	AutorizarLecturaSeguimiento(context.Context, string, string) error
}

type SelladorReincorporacionTitular interface {
	SellarAmbitoReincorporacionTitular(context.Context, SolicitudSellarAmbitoIdempotencia) (ColeccionSellosHMAC, error)
	DerivarHuellaReincorporacionTitular(context.Context, []byte) (ColeccionSellosHMAC, error)
}
