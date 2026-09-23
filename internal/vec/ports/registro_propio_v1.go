package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrRegistroPropioNoDisponible = errors.New("vec: registro propio no disponible")

const (
	AccionRegistroPropioV1    = "vec.registro_propio.crear"
	FinalidadRegistroPropioV1 = "alta_vec_propia"
	AudienciaRegistroPropioV1 = "vec.registro_propio.v1"
)

// La referencia de credencial procede de la frontera autenticada, nunca de
// datos libres enviados por el navegador. Sin autoridad institucional
// inyectada este puerto debe denegar.
type AcreditadorInstitucionalRegistroPropioV1 interface {
	AcreditarRegistroPropio(context.Context, string) (domain.AcreditacionInstitucionalRegistroPropioV1, error)
	RevalidarRegistroPropio(context.Context, domain.AcreditacionInstitucionalRegistroPropioV1) error
}

// EquivalenciaPersonaRegistroPropioV1 resuelve una persona existente o
// acredita que el sujeto institucional es nuevo. La operación permite
// recuperar el MISMO dictamen en un replay tras reinicio, aunque la primera
// inscripción ya haya creado el puntero. No deriva persona de certificado,
// correo, documento ni referencia de cuenta.
type EquivalenciaPersonaRegistroPropioV1 interface {
	ResolverEquivalenciaPersona(context.Context, string, domain.AcreditacionInstitucionalRegistroPropioV1) (ResultadoEquivalenciaPersonaRegistroPropioV1, error)
}

type ResultadoEquivalenciaPersonaRegistroPropioV1 struct {
	SujetoRef     string
	PersonaRef    string // Vacio solo si Nueva=true y la autoridad acredita ausencia.
	Nueva         bool
	PruebaRef     string
	PruebaVersion uint64
	PruebaSHA256  string
}

type SolicitudRegistroPropioV1 struct {
	CredencialRef  string
	OperacionRef   string
	VinculoActor   domain.VinculoAutenticacionActorV2
	ResultadoActor domain.ResultadoContextoActorRegistradoV2
	Correlacion    domain.ReferenciaCorrelacionAutorizacionV2
	Motivo         domain.ReferenciaEntradaCatalogo
}

type OrdenRegistroPropioV1 struct {
	OperacionRef    string
	Acreditacion    domain.AcreditacionInstitucionalRegistroPropioV1
	Equivalencia    ResultadoEquivalenciaPersonaRegistroPropioV1
	ActorRef        string
	EntradaCanonica []byte
	RecursoCanonico []byte
	Solicitud       domain.SolicitudAutorizacionLigadaV3
	Decision        domain.DecisionAutorizacionLigadaV3
	Confirmacion    ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material        ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type AutorizadorRegistroPropioV1 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ConfirmacionRegistroConcesionAutorizacionLigadaV3, ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type RegistroPropioV1 interface {
	RegistrarPropio(context.Context, OrdenRegistroPropioV1) (domain.ReciboRegistroPropioV1, error)
}
