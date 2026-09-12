package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrInscripcionVinculoUsuarioVECNoDisponible = errors.New("bolsa: inscripcion propia VEC no disponible")

const (
	AccionRegistrarInscripcionPropiaUsuarioVEC = "bolsa.inscripcion_propia.registrar"
	ModuloInscripcionPropiaUsuarioVEC          = "bolsa"
	TipoRecursoInscripcionPropiaUsuarioVEC     = "inscripcion_propia"
)

// SolicitudRegistrarInscripcionPropiaUsuarioVEC procede de un canal. No lleva
// sujeto, candidato, política ni recurso: todos nacen de autoridades VEC.
type SolicitudRegistrarInscripcionPropiaUsuarioVEC struct {
	AutenticacionRef, SesionRef, PerfilActivoRef string
	ConvocatoriaRef, IntencionRef                string
}

type PoliticaInscripcionPropiaUsuarioVEC struct {
	Referencia                   string
	Version                      uint64
	Huella                       string
	VigenteDesde, VigenteHasta   time.Time
	Accion, Finalidad, Audiencia string
	Motivo                       dominiovec.ReferenciaEntradaCatalogo
	RecursoBase                  dominiovec.RecursoAutorizable
}

// ResolverPoliticaInscripcionPropiaUsuarioVEC entrega sólo configuración
// publicada y admisible; no hay valor por defecto cuando la fuente no responde.
type ResolverPoliticaInscripcionPropiaUsuarioVEC interface {
	ResolverPoliticaInscripcionPropiaUsuarioVEC(context.Context, string, time.Time) (PoliticaInscripcionPropiaUsuarioVEC, error)
}

type ReservaInscripcionPropiaUsuarioVEC struct {
	InscripcionRef, SujetoRef                 string
	IntencionRef, PersonaRef, ConvocatoriaRef string
}

// Reservar es durable e idempotente globalmente por intención vinculada a la
// misma persona y convocatoria; una intención ya vinculada a otro actor falla
// cerrada y no crea otra pareja. Las referencias nuevas nunca reutilizan un
// sujeto histórico. La reserva no crea una inscripción ni una participación.
type ReservadorInscripcionPropiaUsuarioVEC interface {
	ReservarInscripcionPropiaUsuarioVEC(context.Context, string, string, string) (ReservaInscripcionPropiaUsuarioVEC, error)
}

type PreparadorAuditoriaInscripcionPropiaUsuarioVEC interface {
	PrepararAuditoriaInscripcionPropiaUsuarioVEC(context.Context, dominiovec.ResultadoContextoActorRegistradoV2, dominiovec.RecursoAutorizable, dominiovec.ReferenciaCorrelacionAutorizacionV2, string) (dominiovec.AuditEntry, error)
}

// AutorizadorInscripcionUsuarioVEC emite material nominal V3. La exportación
// posterior no es por sí misma una concesión.
type AutorizadorInscripcionUsuarioVEC interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenRegistroInscripcionPropiaUsuarioVEC struct {
	InscripcionRef, SujetoRef, PersonaRef, CandidatoRef string
	VinculoCandidatoRef                                 string
	VersionVinculoCandidato, VersionPersona             uint64
	ConvocatoriaRef, IntencionRef                       string
	PoliticaRef                                         string
	PoliticaVersion                                     uint64
	PoliticaHuella                                      string
	Audiencia, Finalidad                                string
	PayloadNegocio                                      []byte
	Recurso                                             dominiovec.RecursoAutorizable
	Solicitud                                           dominiovec.SolicitudAutorizacionLigadaV3
	Decision                                            dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion                                        puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material                                            puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ResultadoContexto                                   dominiovec.ResultadoContextoActorRegistradoV2
	Auditoria                                           dominiovec.AuditEntry
}

type ReciboInscripcionPropiaUsuarioVEC struct {
	InscripcionRef, SujetoRef, ReciboRef, EventoRef, AuditoriaRef string
	Version                                                       uint64
	RegistradaEn                                                  time.Time
}

// Registrador consume la capacidad en la misma transacción SERIALIZABLE que
// inscripción, auditoría y outbox. Debe usar AcreditadorUsoRegistroContextoActorV2
// con una OrdenAcreditacion construida desde ResultadoContexto: una vez antes
// de sus locks y otra tras tomarlos, usando este segundo instante para volver a
// comprobar revocación, política y ventana V3 antes del efecto. Fija en la
// auditoría final la DecisionRef consumida y devuelve referencias opacas de
// auditoría, evento y recibo. No admite material fabricado por un DTO del canal
// ni convierte un replay en un efecto adicional. Hasta que este flujo filtre
// campos y cumpla obligaciones, el consumidor también deniega toda decisión V3
// cuya representación canónica no tenga ambas listas explícitamente vacías.
type RegistradorInscripcionPropiaUsuarioVEC interface {
	RegistrarInscripcionPropiaUsuarioVEC(context.Context, OrdenRegistroInscripcionPropiaUsuarioVEC) (ReciboInscripcionPropiaUsuarioVEC, error)
}
