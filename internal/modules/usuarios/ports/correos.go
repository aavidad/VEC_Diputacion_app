package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrCorreosNoAutenticado = errors.New("usuarios correos: no autenticado")
	ErrCorreosProhibido     = errors.New("usuarios correos: prohibido")
	ErrCorreosConflicto     = errors.New("usuarios correos: conflicto")
	ErrCorreosInvalidos     = errors.New("usuarios correos: peticion invalida")
	ErrCorreosNoDisponible  = errors.New("usuarios correos: no disponible")
	ErrCorreosLimite        = errors.New("usuarios correos: limite alcanzado")
)

const (
	FinalidadCorreosPropios          = "finalidad:usuarios:correos-propios:v1"
	MaxReenviosCorreoPorHora         = 3
	MaxIntentosCodigoCorreo          = 5
	AccionConsultarCorreos           = "vec.correos.consultar"
	AccionAnadirCorreo               = "vec.correos.anadir"
	AccionReenviarCorreo             = "vec.correos.reenviar"
	AccionVerificarCorreo            = "vec.correos.verificar"
	AccionActivarCorreo              = "vec.correos.activar"
	AccionRetirarCorreo              = "vec.correos.retirar"
	TipoRecursoCorreos               = "correos_persona"
	AudienciaConsultarCorreosInterna = "vec_usuarios.correos.consultar.interna_corporativa.v1"
	AudienciaAnadirCorreoInterna     = "vec_usuarios.correos.anadir.interna_corporativa.v1"
	AudienciaReenviarCorreoInterna   = "vec_usuarios.correos.reenviar.interna_corporativa.v1"
	AudienciaVerificarCorreoInterna  = "vec_usuarios.correos.verificar.interna_corporativa.v1"
	AudienciaActivarCorreoInterna    = "vec_usuarios.correos.activar.interna_corporativa.v1"
	AudienciaRetirarCorreoInterna    = "vec_usuarios.correos.retirar.interna_corporativa.v1"
	AudienciaConsultarCorreosExterna = "vec_usuarios.correos.consultar.externa_personal.v1"
	AudienciaAnadirCorreoExterna     = "vec_usuarios.correos.anadir.externa_personal.v1"
	AudienciaReenviarCorreoExterna   = "vec_usuarios.correos.reenviar.externa_personal.v1"
	AudienciaVerificarCorreoExterna  = "vec_usuarios.correos.verificar.externa_personal.v1"
	AudienciaActivarCorreoExterna    = "vec_usuarios.correos.activar.externa_personal.v1"
	AudienciaRetirarCorreoExterna    = "vec_usuarios.correos.retirar.externa_personal.v1"
)

type ProveedorMaterialCorreos interface {
	ProveerMaterialCorreos(context.Context, vecdomain.VinculoAutenticacionActorV2, MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenCorreos transporta la identidad opaca validada en dominio y el puerto
// V3. El servicio vuelve a validar Identidad en cada uso.
type OrdenCorreos struct {
	Identidad domain.IdentidadCorreos
	Proveedor ProveedorMaterialCorreos
}

type PeticionCorreo struct {
	VersionEsperada uint64 `json:"version_esperada"`
	ClaveOperacion  string `json:"clave_operacion"`
	CorreoRef       string `json:"correo_ref,omitempty"`
	Direccion       string `json:"direccion,omitempty"`
	Codigo          string `json:"codigo,omitempty"`
	SustitutoRef    string `json:"sustituto_ref,omitempty"`
}

// La colección contiene HMAC dedicados y versionados de la preimagen completa.
// El activo se persiste; retenidos sirven exclusivamente para comparar replay
// tras rotación, sin reescribir el recibo ni ampliar el ámbito de autorización.
type HuellaSemanticaCorreo struct {
	ClaveRef string `json:"clave_ref"`
	Valor    string `json:"valor"`
}

type HuellasSemanticasCorreo struct {
	Activa    HuellaSemanticaCorreo   `json:"activa"`
	Retenidas []HuellaSemanticaCorreo `json:"retenidas"`
}

// Sellar acepta bytes sólo en memoria y nunca registra ni devuelve la
// preimagen. Usa HMAC-SHA256 con clave de huella semántica propia del módulo;
// no la clave de desafío ni de igualdad de direcciones. Retener generaciones
// válidas permite comparar la misma petición con su clave histórica.
type SelladorHuellaCorreos interface {
	SellarHuellaCorreo(context.Context, []byte) (HuellasSemanticasCorreo, error)
}

// Sólo lleva HMAC semánticos de la entrada. Nunca dirección ni código.
type MaterialCorreos struct {
	Superficie      vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
	PersonaRef      string                                   `json:"persona_ref"`
	PerfilRef       string                                   `json:"perfil_ref"`
	Accion          string                                   `json:"accion"`
	FinalidadRef    string                                   `json:"finalidad_ref"`
	VersionEsperada uint64                                   `json:"version_esperada"`
	ClaveOperacion  string                                   `json:"clave_operacion"`
	HuellasPeticion HuellasSemanticasCorreo                  `json:"huellas_peticion"`
	CorreoRef       string                                   `json:"correo_ref"`
	SustitutoRef    string                                   `json:"sustituto_ref"`
}

type VistaCorreos struct {
	PersonaRef string                `json:"persona_ref"`
	Version    uint64                `json:"version"`
	Correos    []domain.CorreoPropio `json:"correos"`
}

type ReciboCorreos struct {
	ReciboRef  string    `json:"recibo_ref"`
	PersonaRef string    `json:"persona_ref"`
	Accion     string    `json:"accion"`
	CorreoRef  string    `json:"correo_ref"`
	Version    uint64    `json:"version"`
	FechaUTC   time.Time `json:"fecha_utc"`
	Replay     bool      `json:"replay"`
}

// ReservaDesafio sólo circula en memoria hasta la transacción. El registro
// conserva huella/estado/clave_ref y escribe Desafio exclusivamente en outbox.
// La dirección cifrada debe ligar AAD a CorreoRef y VersionNueva; el sobre
// ContactoUsuario existente no cumple esa ligadura por sí solo.
type ReservaDesafio struct {
	DesafioRef   string
	Desafio      []byte
	HuellaCodigo []byte
	ClaveRef     string
	VenceUTC     time.Time
}

// El protector cifra con AAD que incorpora persona, correo_ref y versión del
// conjunto. La huella de igualdad debe ser HMAC separada para la unicidad;
// nunca SHA-256 simple de una dirección adivinable. ClaveIgualdadRef identifica
// esa generación HMAC aparte de ClaveRef AEAD; el registro rechaza el alta si
// la generación de igualdad de la persona cambió sin reindexado gobernado.
type SobreDireccionCorreo struct {
	CorreoRef        string
	Version          uint64
	ClaveRef         string
	ClaveIgualdadRef string
	Nonce            []byte
	Cifrado          []byte
	HuellaIgualdad   []byte
}

type ProtectorDireccionCorreo interface {
	CifrarDireccionCorreo(context.Context, string, string, uint64, []byte) (SobreDireccionCorreo, error)
}

type PreparadorDesafioCorreo interface {
	// Debe usar CSPRNG >=128 bits y clave HMAC dedicada/versionada. El código
	// se deriva del dominio, persona, correo_ref, desafío y vencimiento.
	PrepararDesafioCorreo(context.Context, string, string, time.Time) (ReservaDesafio, error)
}

type MetadatosDesafioCorreo struct {
	PersonaRef   string
	CorreoRef    string
	DesafioRef   string
	HuellaCodigo []byte
	ClaveRef     string
	VenceUTC     time.Time
}

// El registro invoca Comprobar bajo bloqueo de la fila pendiente; consume
// atómicamente intento o éxito. Una clave revocada debe producir false/error.
type ComprobadorCodigoCorreo interface {
	Comprobar(context.Context, MetadatosDesafioCorreo) (bool, error)
}

type ValidadorCodigoCorreo interface {
	ComprobarCodigoCorreo(context.Context, MetadatosDesafioCorreo, string) (bool, error)
}

// Aplicar es la única puerta durable. Verifica/consume V3 y CAS en la misma
// transacción que cuota persistente (3 reenvíos/hora, sin contar replay),
// intentos, estado, historia, auditoría, recibo y outbox. El primer alta
// pendiente no se convierte en activo. Activar exige verificado y reserva
// aviso al anterior; retirar activo exige SustitutoRef verificado distinto y
// realiza ambos cambios de forma atómica. No elige sustituto por defecto.
// Si una llamada concurrente ya confirmó misma persona+clave+huella de una
// generación válida, devuelve el recibo ORIGINAL con Replay=true, incluso
// cuando el correo_ref aleatorio de la segunda alta sea distinto. Descarta su
// sobre/desafío nuevos y no consume cuota, historia, auditoría ni outbox extra.
// La misma clave con huella distinta devuelve ErrCorreosConflicto.
type RegistroCorreos interface {
	ConsultarPropios(context.Context, OrdenCorreos, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (VistaCorreos, error)
	RecuperarOperacion(context.Context, OrdenCorreos, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboCorreos, bool, error)
	Aplicar(context.Context, OrdenCorreos, PeticionCorreo, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, SobreDireccionCorreo, ReservaDesafio, ComprobadorCodigoCorreo) (ReciboCorreos, error)
}

// Lectura para CT/Bolsa. El consumidor aporta finalidad y concesión exacta;
// el adaptador audita la lectura y sólo expone el activo verificado. No se
// permite consultar tablas de usuarios desde el módulo consumidor.
type SolicitudCorreoActivo struct {
	PersonaRef   string
	FinalidadRef string
	Material     vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type CorreoActivoVerificado struct {
	PersonaRef string
	CorreoRef  string
	Direccion  string
	Version    uint64
}

type LectorCorreoActivoVerificado interface {
	ConsultarActivoVerificado(context.Context, SolicitudCorreoActivo) (CorreoActivoVerificado, bool, error)
}

// El dispatcher consume una reserva ya confirmada por el registro. Sólo
// Desafio sale por outbox; el código se deriva en memoria y el transporte no
// informa entrega jurídica, únicamente aceptación SMTP.
type DespachoVerificacionCorreo struct {
	OutboxRef  string
	PersonaRef string
	CorreoRef  string
	DesafioRef string
	Desafio    []byte
	ClaveRef   string
	VenceUTC   time.Time
}

// Resuelve y descifra la dirección en memoria después de autorizar el intento
// de despacho. La dirección en claro no figura en el outbox.
type LectorDireccionDespachoCorreo interface {
	ConDireccionDespachoCorreo(context.Context, string, string, func(string) error) error // persona_ref, correo_ref
}

type DerivadorCodigoDespachoCorreo interface {
	DerivarCodigoCorreo(context.Context, DespachoVerificacionCorreo) (string, error)
}

type TransportadorVerificacionCorreo interface {
	AceptarVerificacionCorreo(context.Context, string, string, string) error // outbox_ref, direccion, codigo
}
