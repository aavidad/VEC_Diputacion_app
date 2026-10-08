package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrVinculoOfertaBolsaInvalido     = errors.New("contratacion temporal: vinculo de oferta Bolsa invalido")
	ErrVinculoOfertaBolsaDenegado     = errors.New("contratacion temporal: vinculo de oferta Bolsa denegado")
	ErrVinculoOfertaBolsaNoDisponible = errors.New("contratacion temporal: vinculo de oferta Bolsa no disponible")
	ErrVinculoOfertaBolsaEnConflicto  = errors.New("contratacion temporal: oferta o expediente ya vinculado")
	ErrVinculoOfertaBolsaNoConfiable  = errors.New("contratacion temporal: recibo de oferta Bolsa no confiable")
)

const (
	AccionVincularOfertaBolsaCT    = string(domain.AccionVincularOfertaBolsa)
	AudienciaVincularOfertaBolsaCT = "vec_contratacion_temporal.oferta_bolsa.vincular.v1"
	FinalidadVincularOfertaBolsaCT = "gestionar_contratacion_temporal"
	TipoRecursoVincularOfertaBolsa = "expediente_contratacion_temporal"
	AccionAsociarOfertaBolsa       = "bolsa.oferta.ct.asociar"
	AudienciaAsociarOfertaBolsa    = "vec_bolsa_llamamientos.oferta_ct.asociar.v1"
)

// La frontera autenticada fija identidad, perfil y organización. Oferta y
// plaza son la intención del operador, no evidencia de procedencia.
type SolicitudVincularOfertaBolsa struct {
	AutenticacionRef, SesionRef, PerfilRef string
	OrganizacionRef, ExpedienteRef         string
	BolsaRef, OfertaRef                    string
	NumeroPlaza                            int
	VersionEsperada                        uint64
	ClaveIdempotencia                      string
}

func (s SolicitudVincularOfertaBolsa) Validar() error {
	if (SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: s.AutenticacionRef, SesionRef: s.SesionRef,
		PerfilRef: s.PerfilRef,
	}).Validar() != nil || !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.BolsaRef) ||
		!domain.ReferenciaOpacaValida(s.OfertaRef) ||
		s.NumeroPlaza < 1 || s.NumeroPlaza > 100 ||
		s.VersionEsperada < 4 || s.VersionEsperada >= MaximoEnteroSeguroIntegracionBolsa ||
		!ClaveIdempotenciaValida(s.ClaveIdempotencia) {
		return ErrVinculoOfertaBolsaInvalido
	}
	return nil
}

// PreparacionVinculoOfertaBolsa nace tras resolver ambas autorizaciones
// nominales. CT y Bolsa consumen cada una su capacidad dentro de la misma
// transacción PostgreSQL; ningún recibo de la otra otorga el permiso local.
type PreparacionVinculoOfertaBolsa struct {
	Solicitud                                 SolicitudVincularOfertaBolsa
	ActorRef, PerfilRef, UnidadRef, AmbitoRef string
	CorrelacionRef                            string
	Definicion                                domain.DefinicionCircuitoRRHH
	AutorizacionCT                            vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	AutorizacionBolsa                         vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (p PreparacionVinculoOfertaBolsa) ValidarPara(s SolicitudVincularOfertaBolsa) error {
	if s.Validar() != nil || p.Solicitud != s ||
		!domain.ReferenciaOpacaValida(p.ActorRef) || p.PerfilRef != s.PerfilRef ||
		!domain.ReferenciaOpacaValida(p.UnidadRef) ||
		!domain.ReferenciaOpacaValida(p.AmbitoRef) ||
		!domain.ReferenciaOpacaValida(p.CorrelacionRef) ||
		p.Definicion.Validar() != nil ||
		p.AutorizacionCT.ValidarEstructura() != nil ||
		p.AutorizacionBolsa.ValidarEstructura() != nil {
		return ErrVinculoOfertaBolsaDenegado
	}
	ct, bolsa := p.AutorizacionCT.ResumenCapacidad(), p.AutorizacionBolsa.ResumenCapacidad()
	if ct.Operacion() != AccionVincularOfertaBolsaCT ||
		ct.AudienciaConsumo() != AudienciaVincularOfertaBolsaCT ||
		ct.EfectoRef() != s.ExpedienteRef ||
		bolsa.Operacion() != AccionAsociarOfertaBolsa ||
		bolsa.AudienciaConsumo() != AudienciaAsociarOfertaBolsa ||
		bolsa.EfectoRef() != s.OfertaRef ||
		ct.DecisionRef() == bolsa.DecisionRef() ||
		ct.ContextoRef() != bolsa.ContextoRef() ||
		ct.ContextoHuellaSHA256() != bolsa.ContextoHuellaSHA256() {
		return ErrVinculoOfertaBolsaDenegado
	}
	return nil
}

type ResultadoVinculoOfertaBolsa struct {
	Anterior                 domain.Expediente `json:"anterior"`
	Siguiente                domain.Expediente `json:"siguiente"`
	OfertaRef                string            `json:"oferta_ref"`
	BolsaRef                 string            `json:"bolsa_ref"`
	NumeroPlaza              int               `json:"numero_plaza"`
	ReciboPublicacionRef     string            `json:"recibo_publicacion_ref"`
	PublicacionSHA256        string            `json:"publicacion_sha256"`
	ReciboAsociacionBolsaRef string            `json:"recibo_asociacion_bolsa_ref"`
	ReciboCTRef              string            `json:"recibo_ct_ref"`
	AuditoriaCTRef           string            `json:"auditoria_ct_ref"`
	EventoCTRef              string            `json:"evento_ct_ref"`
	PublicadaEn              time.Time         `json:"publicada_en"`
	AsociadaEn               time.Time         `json:"asociada_en"`
	RegistradaEn             time.Time         `json:"registrada_en"`
	Reutilizada              bool              `json:"reutilizada"`
}

type PreparadorVinculoOfertaBolsa interface {
	PrepararVinculoOfertaBolsa(context.Context, SolicitudVincularOfertaBolsa) (PreparacionVinculoOfertaBolsa, error)
}

type TransaccionVinculoOfertaBolsa interface {
	VincularOfertaBolsa(context.Context, PreparacionVinculoOfertaBolsa) (ResultadoVinculoOfertaBolsa, error)
}
