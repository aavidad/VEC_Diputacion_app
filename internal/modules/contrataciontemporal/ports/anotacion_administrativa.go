package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"time"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	DominioAmbitoIdempotenciaAnotacionAdministrativa = "vec.contratacion-temporal.anotacion-administrativa.ambito"
	DominioHuellaPeticionAnotacionAdministrativa     = "vec.contratacion-temporal.anotacion-administrativa.peticion"
	OperacionRegistrarAnotacionAdministrativa        = "registrar_anotacion_administrativa"
	TipoRecursoAnotacionAdministrativa               = "anotacion_administrativa_incorporacion_v1"
	FinalidadRegistrarAnotacionAdministrativa        = "registrar_anotacion_administrativa_incorporacion"
)

var (
	ErrPreparacionAnotacionAdministrativaInvalida      = errors.New("contratacion temporal: preparacion de anotacion administrativa invalida")
	ErrResultadoAnotacionAdministrativaNoConfiable     = errors.New("contratacion temporal: resultado de anotacion administrativa no confiable")
	ErrPersistenciaAnotacionAdministrativaNoDisponible = errors.New("contratacion temporal: persistencia de anotacion administrativa no disponible")
)

// MaterialAnotacionAdministrativa solo contiene referencias ya obtenidas por
// servidor. SolicitudPersonalRef nunca procede del navegador.
type MaterialAnotacionAdministrativa struct {
	OrganizacionRef      string
	ExpedienteRef        string
	SolicitudPersonalRef string
	VersionEsperada      uint64
	ClaveIdempotencia    string
	Observaciones        string
	ActorRef             string
	PerfilRef            string
}

func (m MaterialAnotacionAdministrativa) Validar() error {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) || !domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(m.SolicitudPersonalRef) || m.VersionEsperada == 0 || m.VersionEsperada > 9007199254740990 ||
		!ClaveIdempotenciaValida(m.ClaveIdempotencia) || !domain.ReferenciaOpacaValida(m.ActorRef) ||
		!domain.ReferenciaOpacaValida(m.PerfilRef) || !textoFiscalizacionValido(m.Observaciones, false) {
		return ErrPreparacionAnotacionAdministrativaInvalida
	}
	return nil
}

type SolicitudPrepararAnotacionAdministrativa struct {
	Material            MaterialAnotacionAdministrativa
	AmbitosHMAC         ColeccionSellosHMAC
	HuellasPeticionHMAC ColeccionSellosHMAC
}

func (s SolicitudPrepararAnotacionAdministrativa) Validar() error {
	if s.Material.Validar() != nil || s.AmbitosHMAC.ValidarDominio(DominioAmbitoIdempotenciaAnotacionAdministrativa) != nil || s.HuellasPeticionHMAC.ValidarDominio(DominioHuellaPeticionAnotacionAdministrativa) != nil {
		return ErrPreparacionAnotacionAdministrativaInvalida
	}
	return nil
}

type EstadoPreparacionAnotacionAdministrativa string

const (
	PreparacionAnotacionAdministrativaPreparada  EstadoPreparacionAnotacionAdministrativa = "preparada"
	PreparacionAnotacionAdministrativaConfirmada EstadoPreparacionAnotacionAdministrativa = "confirmada"
)

type ReferenciasEfectoAnotacionAdministrativa struct{ ReciboRef, EventoRef string }

type PreparacionAnotacionAdministrativa struct {
	Material                       MaterialAnotacionAdministrativa
	AmbitoIdempotenciaHMAC         string
	HuellaPeticionHMAC             string
	Expediente                     domain.Expediente
	SeguimientoOriginal            domain.VinculoSeguimientoOriginal
	Referencias                    ReferenciasEfectoAnotacionAdministrativa
	Estado                         EstadoPreparacionAnotacionAdministrativa
	ReciboConfirmado               *ReciboAnotacionAdministrativa
	HuellaEstadoSeguimientoSHA256  string
	ReciboIncorporacionOriginalRef string
}

type ResolutorSolicitudPersonalAnotacionAdministrativa interface {
	ResolverSolicitudPersonalAnotacionAdministrativa(context.Context, string, string) (string, error)
}

type PreparadorAnotacionAdministrativaIdempotente interface {
	PrepararAnotacionAdministrativa(context.Context, SolicitudPrepararAnotacionAdministrativa) (PreparacionAnotacionAdministrativa, error)
}

type SelladorAmbitoAnotacionAdministrativa interface {
	SellarAmbitoAnotacionAdministrativa(context.Context, SolicitudSellarAmbitoIdempotencia) (ColeccionSellosHMAC, error)
}
type DerivadorHuellaAnotacionAdministrativa interface {
	DerivarHuellaAnotacionAdministrativa(context.Context, MaterialAnotacionAdministrativa) (ColeccionSellosHMAC, error)
}
type SolicitudResolverPoliticaAnotacionAdministrativa struct {
	Material    MaterialAnotacionAdministrativa
	Preparacion PreparacionAnotacionAdministrativa
	Instante    time.Time
}
type ResolutorPoliticaAnotacionAdministrativa interface {
	ResolverPoliticaAnotacionAdministrativa(context.Context, SolicitudResolverPoliticaAnotacionAdministrativa) (PoliticaAnotacionAdministrativa, error)
}
type PoliticaAnotacionAdministrativa struct {
	MotivoAutorizacion     vd.ReferenciaEntradaCatalogo
	DefinicionRef          string
	DefinicionVersion      uint64
	DefinicionHuellaSHA256 string
	EvaluadaEn             time.Time
	ValidaHasta            time.Time
}

type EvidenciaAutorizacionAnotacionAdministrativa struct {
	Contexto       ContextoAutorizacionAltaV3
	SolicitudV3    vd.SolicitudAutorizacionLigadaV3
	DecisionV3     vd.DecisionAutorizacionLigadaV3
	ConfirmacionV3 vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

type OrdenConfirmarAnotacionAdministrativa struct {
	Material            MaterialAnotacionAdministrativa
	ExpedienteSiguiente domain.Expediente
	SeguimientoOriginal domain.VinculoSeguimientoOriginal
	Referencias         ReferenciasEfectoAnotacionAdministrativa
	InstanteEfecto      time.Time
	Evidencia           EvidenciaAutorizacionAnotacionAdministrativa
	Preparacion         PreparacionAnotacionAdministrativa
	Politica            PoliticaAnotacionAdministrativa
}

type TransaccionAnotacionesAdministrativas interface {
	ConfirmarAnotacionAdministrativa(context.Context, OrdenConfirmarAnotacionAdministrativa) (ReciboAnotacionAdministrativa, error)
}

type ReciboAnotacionAdministrativa struct {
	Operacion           string                            `json:"operacion"`
	OrganizacionRef     string                            `json:"organizacion_ref"`
	ExpedienteRef       string                            `json:"expediente_ref"`
	VersionAnterior     uint64                            `json:"version_anterior"`
	VersionResultante   uint64                            `json:"version_resultante"`
	FaseResultante      domain.ClaveFase                  `json:"fase_resultante"`
	EstadoResultante    domain.EstadoOperativo            `json:"estado_resultante"`
	SeguimientoOriginal domain.VinculoSeguimientoOriginal `json:"seguimiento_original"`
	ReciboRef           string                            `json:"recibo_ref"`
	AuditoriaRef        string                            `json:"auditoria_ref"`
	EventoRef           string                            `json:"evento_ref"`
	ActorRef            string                            `json:"actor_ref"`
	RegistradaEn        time.Time                         `json:"registrada_en"`
}

func (p PreparacionAnotacionAdministrativa) ValidarPara(s SolicitudPrepararAnotacionAdministrativa) error {
	if s.Validar() != nil || p.Material != s.Material || p.Expediente.Validar() != nil || p.SeguimientoOriginal.Validar() != nil || p.Expediente.Referencia != p.Material.ExpedienteRef || p.Expediente.OrganizacionRef != p.Material.OrganizacionRef || p.Expediente.Version != p.Material.VersionEsperada || p.Expediente.Asignacion == nil ||
		!domain.ReferenciaOpacaValida(p.Referencias.ReciboRef) || !domain.ReferenciaOpacaValida(p.Referencias.EventoRef) || !domain.ReferenciaOpacaValida(p.ReciboIncorporacionOriginalRef) || !huellaSHA256OperacionAnalisisValida(p.HuellaEstadoSeguimientoSHA256) ||
		!ColeccionesHMACContienenPar(s.AmbitosHMAC, DominioAmbitoIdempotenciaAnotacionAdministrativa, s.HuellasPeticionHMAC, DominioHuellaPeticionAnotacionAdministrativa, p.AmbitoIdempotenciaHMAC, p.HuellaPeticionHMAC) {
		return ErrPreparacionAnotacionAdministrativaInvalida
	}
	switch p.Estado {
	case PreparacionAnotacionAdministrativaPreparada:
		if p.ReciboConfirmado != nil {
			return ErrPreparacionAnotacionAdministrativaInvalida
		}
	case PreparacionAnotacionAdministrativaConfirmada:
		if p.ReciboConfirmado == nil || p.ReciboConfirmado.ValidarParaPreparacion(p) != nil {
			return ErrPreparacionAnotacionAdministrativaInvalida
		}
	default:
		return ErrPreparacionAnotacionAdministrativaInvalida
	}
	return nil
}
func (p PoliticaAnotacionAdministrativa) ValidarPara(s SolicitudResolverPoliticaAnotacionAdministrativa, t time.Time) error {
	if s.Material.Validar() != nil || s.Material != s.Preparacion.Material || s.Preparacion.Expediente.Validar() != nil || s.Preparacion.SeguimientoOriginal.Validar() != nil ||
		!domain.ReferenciaOpacaValida(p.DefinicionRef) || p.DefinicionVersion == 0 || !huellaSHA256OperacionAnalisisValida(p.DefinicionHuellaSHA256) || p.MotivoAutorizacion.Validar() != nil ||
		!domain.InstanteUTCCanonico(t) || !domain.InstanteUTCCanonico(p.EvaluadaEn) || !domain.InstanteUTCCanonico(p.ValidaHasta) || !p.EvaluadaEn.Equal(s.Instante) || t.Before(p.EvaluadaEn) || !t.Before(p.ValidaHasta) {
		return ErrPreparacionAnotacionAdministrativaInvalida
	}
	return nil
}
func (r ReciboAnotacionAdministrativa) ValidarParaPreparacion(p PreparacionAnotacionAdministrativa) error {
	m := p.Material
	if r.Operacion != OperacionRegistrarAnotacionAdministrativa || r.OrganizacionRef != m.OrganizacionRef || r.ExpedienteRef != m.ExpedienteRef || r.VersionAnterior != m.VersionEsperada || r.VersionResultante != r.VersionAnterior+1 || r.FaseResultante != p.Expediente.FaseActual || r.EstadoResultante != p.Expediente.EstadoActual || r.SeguimientoOriginal != p.SeguimientoOriginal || r.ReciboRef != p.Referencias.ReciboRef || r.EventoRef != p.Referencias.EventoRef || r.ActorRef != m.ActorRef || !domain.ReferenciaOpacaValida(r.AuditoriaRef) || !domain.InstanteUTCCanonico(r.RegistradaEn) || r.RegistradaEn.Before(p.Expediente.ActualizadoEn) {
		return ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if p.ReciboConfirmado != nil && !reflect.DeepEqual(*p.ReciboConfirmado, r) {
		return ErrResultadoAnotacionAdministrativaNoConfiable
	}
	return nil
}

// RecursoAutorizacionAnotacionAdministrativa comparte el contrato exacto entre
// aplicación y adaptadores. Sólo recibe la preparación resuelta por servidor.
func RecursoAutorizacionAnotacionAdministrativa(p PreparacionAnotacionAdministrativa, politica PoliticaAnotacionAdministrativa) vd.RecursoAutorizable {
	m := p.Material
	h := sha256.Sum256([]byte(m.Observaciones))
	return vd.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ModuloContratacion, Tipo: TipoRecursoAnotacionAdministrativa,
		Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef, "expediente_ref": m.ExpedienteRef, "fase_previa": string(p.Expediente.FaseActual), "estado_previo": string(p.Expediente.EstadoActual)},
		Atributos: map[string]string{"operacion": OperacionRegistrarAnotacionAdministrativa, "version_expediente": strconv.FormatUint(m.VersionEsperada, 10), "solicitud_personal_ref": m.SolicitudPersonalRef,
			"seguimiento_ref": p.SeguimientoOriginal.SeguimientoRef, "version_seguimiento": strconv.FormatUint(p.SeguimientoOriginal.VersionSeguimiento, 10), "huella_raiz_seguimiento_sha256": p.SeguimientoOriginal.HuellaRaizSeguimientoSHA256,
			"estado_seguimiento_sha256": p.HuellaEstadoSeguimientoSHA256, "recibo_incorporacion_ref": p.ReciboIncorporacionOriginalRef, "observaciones_huella_sha256": hex.EncodeToString(h[:]),
			"ambito_idempotencia_hmac": p.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": p.HuellaPeticionHMAC, "politica_ref": politica.DefinicionRef, "politica_version": strconv.FormatUint(politica.DefinicionVersion, 10), "politica_huella_sha256": politica.DefinicionHuellaSHA256}}
}

// RecuperadorMaterialAnotacionAdministrativaAutorizado debe validar lectura
// nominal actual antes de obtener texto privado. El HMAC sólo localiza; no autoriza.
type SolicitudRecuperarMaterialAnotacionAdministrativa struct {
	Contexto          ContextoAutorizacionAltaV3
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	ActorRef          string
	PerfilRef         string
	AmbitosHMAC       ColeccionSellosHMAC
}
type RecuperadorMaterialAnotacionAdministrativaAutorizado interface {
	RecuperarMaterialAnotacionAdministrativaAutorizada(context.Context, SolicitudRecuperarMaterialAnotacionAdministrativa) (MaterialAnotacionAdministrativa, error)
}
