package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionPublicarDenominacionPersona = "vec.persona.denominacion.publicar"
	AccionLeerDenominacionPersona     = "vec.persona.denominacion.leer"
	AccionBuscarDenominacionPersona   = "vec.persona.denominacion.buscar"
)

// NormaDenominacionPersona procede de configuración/catálogo versionado.
// Case: exact o fold; FormaUnicode: NFC. No selecciona un idioma.
type NormaDenominacionPersona struct {
	Ref, Case, FormaUnicode, Separadores string
	MaxBytes, MaxTokens                  int
}

type IndiceDenominacionPersona struct {
	AmbitoRef, NormaRef, NormaSHA256, ClaveRef string
	Tokens                                     [][]byte
}

type SobreDenominacionPersona struct {
	Esquema, PersonaRef, ClaveRef string
	Version                       uint64
	Nonce, Cifrado                []byte
	Indice                        IndiceDenominacionPersona
}

// La preimagen durable contiene solamente el sobre y metadatos opacos. Su
// SHA256 nunca se calcula sobre el nombre o los términos de búsqueda claros.
type PreparacionDenominacionPersona struct {
	PersonaRef, ProcedenciaRef, SobreSHA256 string
	VersionEsperada                         uint64
	Sobre                                   SobreDenominacionPersona
}

type AccesoDenominacionPersona struct {
	PersonaRef, Audiencia, FinalidadRef string
	Version                             uint64
	Contexto                            domain.ContextoActor
	ResultadoContexto                   domain.ResultadoContextoActorRegistradoV2
	Vinculo                             domain.VinculoAutenticacionActorV2
	Recurso                             domain.RecursoAutorizable
	Auditoria                           domain.AuditEntry
	Material                            ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type OrdenDenominacionPersona struct {
	Preparacion PreparacionDenominacionPersona
	Acceso      AccesoDenominacionPersona
}

type ReciboDenominacionPersona struct {
	PersonaRef, ProcedenciaRef, SobreSHA256, AuditoriaRef string
	Version                                               uint64
}

type ProtectorDenominacionPersona interface {
	PrepararDenominacionPersona(context.Context, string, uint64, string, string, string, []byte) (PreparacionDenominacionPersona, error)
	ConDenominacionDescifrada(context.Context, SobreDenominacionPersona, func(domain.DenominacionPersona) error) error
	RevalidarProteccionDenominacionPersona(context.Context, SobreDenominacionPersona) error
	PrepararBusquedaDenominacionPersona(context.Context, string, string, []byte) (IndiceDenominacionPersona, error)
}

// RegistroDenominacionPersona es la futura autoridad durable. Debe revalidar
// existencia/vigencia de Persona, consumir V3 nominal fresco y verificar el
// sobre y CAS en la MISMA transacción que versión, puntero actual, auditoría y
// outbox. Conserva historia por adición; no modifica identidad_versiones.
// El recibo sólo se devuelve tras COMMIT confirmado o reconciliado.
type RegistroDenominacionPersona interface {
	PublicarDenominacionPersona(context.Context, OrdenDenominacionPersona) (ReciboDenominacionPersona, error)
}

// FuenteDenominacionPersonaAutorizada es una frontera confiable, nunca un
// permiso derivado del DTO. Verifica firmas V3, contexto, finalidad, campos,
// ámbito y revocación con las autoridades centrales. Leer y Buscar devuelven
// sólo sobres/referencias DESPUÉS de confirmar consumo y auditoría de acceso.
// Buscar compara TODOS los tokens completos dentro del ámbito y la versión
// de norma/clave; no descifra páginas para filtrarlas. Limite máximo: 100.
type FuenteDenominacionPersonaAutorizada interface {
	LeerDenominacionPersonaAutorizada(context.Context, AccesoDenominacionPersona) (LecturaDenominacionPersonaConfirmada, error)
	// Valida el acuse contra Material y el consumo COMMIT común original.
	ValidarAcuseDenominacionPersona(context.Context, AccesoDenominacionPersona, LecturaDenominacionPersonaConfirmada) error
	BuscarDenominacionPersonaAutorizada(context.Context, AccesoDenominacionPersona, IndiceDenominacionPersona, int) ([]ReferenciaDenominacionPersona, error)
	RevalidarAccesoDenominacionPersona(context.Context, AccesoDenominacionPersona, SobreDenominacionPersona) error
}

type LecturaDenominacionPersonaConfirmada struct {
	Sobre SobreDenominacionPersona
	Acuse domain.AcuseConsumoDenominacionPersona
}

type ReferenciaDenominacionPersona struct {
	PersonaRef string
	Version    uint64
}

// Este puerto consulta Persona canónica; comprobar un prefijo no acredita su
// existencia. La autoridad durable debe volver a comprobarlo en su transacción.
type ExistenciaPersonaDenominacion interface {
	RevalidarPersonaDenominacion(context.Context, string) error
}
