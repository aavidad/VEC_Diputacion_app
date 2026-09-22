package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionRegistrarContactoParticipacion    = "bolsa.contacto_participacion.registrar"
	FinalidadRegistrarContactoParticipacion = "gestion_contactos_participacion"
	AudienciaRegistrarContactoParticipacion = "vec_bolsa_llamamientos.contacto_participacion.registrar.v1"
	AccionConsultarContactoParticipacion    = "bolsa.contacto_participacion.consultar"
	FinalidadConsultarContactoParticipacion = "consulta_contactos_participacion"
	AudienciaConsultarContactoParticipacion = "vec_bolsa_llamamientos.contacto_participacion.consultar.v1"
)

var ErrContactoParticipacionNoDisponible = errors.New("bolsa: contacto de participacion no disponible")
var ErrContactoParticipacionNoEncontrado = errors.New("bolsa: contacto de participacion no encontrado")

type SolicitudRegistrarContactoParticipacion struct {
	Vinculo                                                                                    dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto                                                                          dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef, ParticipacionRef, LlamamientoRef, Canal, Resultado, Anotacion, ClaveIdempotencia string
	Instante                                                                                   time.Time
	Correlacion                                                                                dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion                                                                         dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudRegistrarContactoParticipacion) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil || s.BolsaRef == "" || s.ParticipacionRef == "" || s.Canal == "" || s.Resultado == "" || s.Anotacion == "" || s.ClaveIdempotencia == "" || s.Instante.IsZero() || s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrContactoParticipacionNoDisponible
	}
	return nil
}

type RegistroContactoParticipacion struct {
	Contacto    dominiobolsa.ContactoParticipacion
	ReciboRef   string
	Reutilizado bool
}
type ComandoRegistrarContactoParticipacion struct {
	Contacto                     dominiobolsa.ContactoParticipacion
	ClaveIdempotencia, ReciboRef string
	SolicitudAutorizacion        dominiovec.SolicitudAutorizacionLigadaV3
	Decision                     dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion                 puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material                     puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ConsultaContactosParticipacion struct {
	Vinculo                            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto                  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef, ParticipacionRef, Cursor string
	Limite                             int
	Correlacion                        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion                 dominiovec.ReferenciaEntradaCatalogo
	SolicitudAutorizacion              dominiovec.SolicitudAutorizacionLigadaV3
	Decision                           dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion                       puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material                           puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ConsultaContactosBolsa struct {
	Vinculo               dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto     dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef, Cursor      string
	Limite                int
	Correlacion           dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion    dominiovec.ReferenciaEntradaCatalogo
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type PaginaContactosParticipacion struct {
	Contactos       []dominiobolsa.ContactoParticipacion
	CursorSiguiente string
}
type RepositorioContactoParticipacion interface {
	ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error)
	RegistrarContacto(context.Context, ComandoRegistrarContactoParticipacion) (RegistroContactoParticipacion, error)
	ListarContactosParticipacion(context.Context, ConsultaContactosParticipacion) (PaginaContactosParticipacion, error)
	ListarContactosBolsa(context.Context, ConsultaContactosBolsa) (PaginaContactosParticipacion, error)
}

type ResolutorContextoContactoParticipacion interface {
	ResolutorContextoSituacionParticipacion
	ResolverContextoContactosBolsa(context.Context, dominiovec.ContextoActor, string) (ContextoSituacionParticipacionResuelto, error)
}
