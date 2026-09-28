package ports

import (
	"context"
	"errors"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionPublicarCausasParticipacion               = "bolsa.causas_participacion.publicar"
	AccionProponerCausasParticipacion               = "bolsa.causas_participacion.proponer"
	AccionConsultarPropuestaCausasParticipacion     = "bolsa.causas_participacion.propuesta.consultar"
	AccionConsultarCausasParticipacion              = "bolsa.causas_participacion.consultar"
	FinalidadPublicarCausasParticipacion            = "gestionar_catalogo_causas_participacion"
	FinalidadProponerCausasParticipacion            = "preparar_catalogo_causas_participacion"
	FinalidadConsultarPropuestaCausasParticipacion  = "revisar_propuesta_catalogo_causas_participacion"
	FinalidadConsultarCausasParticipacion           = "seleccionar_causa_participacion"
	AudienciaPublicarCausasParticipacion            = "vec_bolsa_llamamientos.causas_participacion.publicar.v1"
	AudienciaProponerCausasParticipacion            = "vec_bolsa_llamamientos.causas_participacion.proponer.v1"
	AudienciaConsultarPropuestaCausasParticipacion  = "vec_bolsa_llamamientos.causas_participacion.propuesta.consultar.v1"
	AudienciaConsultarCausasParticipacion           = "vec_bolsa_llamamientos.causas_participacion.consultar.v1"
	TipoRecursoCatalogoCausasParticipacion          = "catalogo_causas_participacion"
	TipoRecursoPropuestaCatalogoCausasParticipacion = "propuesta_catalogo_causas_participacion"
	ReferenciaCatalogoCausasParticipacion           = "vec.bolsa.causas_participacion"
)

var (
	ErrCatalogoCausasParticipacionNoDisponible = errors.New("bolsa: catalogo de causas de participacion no disponible")
	ErrCatalogoCausasParticipacionEnConflicto  = errors.New("bolsa: catalogo de causas de participacion en conflicto")
	ErrPropuestaCausaParticipacionNoEncontrada = errors.New("bolsa: propuesta de causa de participacion no encontrada")
)

type CausaParticipacionCatalogada struct {
	Codigo          string `json:"codigo"`
	Version         int64  `json:"version"`
	HuellaSHA256    string `json:"huella_sha256"`
	Etiqueta        string `json:"etiqueta"`
	AplicaSituacion bool   `json:"aplica_situacion"`
	AplicaContacto  bool   `json:"aplica_contacto"`
	Publicable      bool   `json:"publicable"`
	Activa          bool   `json:"activa"`
}

type SolicitudPublicarCausaParticipacion struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Causa              CausaParticipacionCatalogada
	PropuestaRef       string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}
type SolicitudProponerCausaParticipacion SolicitudPublicarCausaParticipacion

func (s SolicitudProponerCausaParticipacion) Validar() error {
	x := SolicitudPublicarCausaParticipacion(s)
	x.PropuestaRef = "propuesta"
	return x.Validar()
}

type PropuestaCausaParticipacion struct {
	PropuestaRef string `json:"propuesta_ref"`
	CausaParticipacionCatalogada
}
type PropuestaCausaParticipacionLeida struct {
	Propuesta PropuestaCausaParticipacion `json:"propuesta"`
	Estado    string                      `json:"estado"`
	Recibo    string                      `json:"recibo"`
}
type SolicitudConsultarPropuestaCausaParticipacion struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	PropuestaRef       string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudPublicarCausaParticipacion) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil || s.PropuestaRef == "" || s.Causa.Codigo == "" || s.Causa.Version < 1 || s.Causa.Etiqueta == "" || (!s.Causa.AplicaSituacion && !s.Causa.AplicaContacto) || s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrCatalogoCausasParticipacionNoDisponible
	}
	return nil
}

type SolicitudConsultarCausasParticipacion struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudConsultarCausasParticipacion) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil || s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrCatalogoCausasParticipacionNoDisponible
	}
	return nil
}

type ComandoPublicarCausaParticipacion struct {
	Causa                 CausaParticipacionCatalogada
	PropuestaRef          string
	Actor, ReciboRef      string
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ComandoProponerCausaParticipacion struct {
	Causa                          CausaParticipacionCatalogada
	Actor, PropuestaRef, ReciboRef string
	SolicitudAutorizacion          dominiovec.SolicitudAutorizacionLigadaV3
	Decision                       dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion                   puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material                       puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ConsultaCausasParticipacionAutorizada struct {
	Actor                 string
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ConsultaPropuestaCausaParticipacionAutorizada struct {
	PropuestaRef, Actor   string
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type AutorizadorCatalogoCausasParticipacionV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
type RepositorioCatalogoCausasParticipacion interface {
	ProponerCausaParticipacion(context.Context, ComandoProponerCausaParticipacion) (PropuestaCausaParticipacion, string, error)
	PublicarCausaParticipacion(context.Context, ComandoPublicarCausaParticipacion) (CausaParticipacionCatalogada, string, error)
	ListarCausasParticipacion(context.Context, ConsultaCausasParticipacionAutorizada) ([]CausaParticipacionCatalogada, error)
	ConsultarPropuestaCausaParticipacion(context.Context, ConsultaPropuestaCausaParticipacionAutorizada) (PropuestaCausaParticipacionLeida, error)
}
