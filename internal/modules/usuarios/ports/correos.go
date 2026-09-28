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
	FinalidadCorreosPropios  = "finalidad:usuarios:correos-propios:v1"
	MaxReenviosCorreoPorHora = 3
	MaxIntentosCodigoCorreo  = 5
	AccionConsultarCorreos   = "vec.correos.consultar"
	AccionAnadirCorreo       = "vec.correos.anadir"
	AccionReenviarCorreo     = "vec.correos.reenviar"
	AccionVerificarCorreo    = "vec.correos.verificar"
	AccionActivarCorreo      = "vec.correos.activar"
	AccionRetirarCorreo      = "vec.correos.retirar"
)

type ProveedorMaterialCorreos interface {
	ProveerMaterialCorreos(context.Context, MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenCorreos sólo se obtiene a partir de la persona canónica del actor.
// PersonaRef nunca forma parte de una petición HTTP.
type OrdenCorreos struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialCorreos
}

func NuevaOrdenCorreos(actor vecdomain.ContextoActor, proveedor ProveedorMaterialCorreos) (OrdenCorreos, error) {
	if actor.Validar() != nil || (actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate && actor.Principal.AuthMethod != vecdomain.AuthMethodDNIe) || proveedor == nil {
		return OrdenCorreos{}, ErrCorreosNoAutenticado
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenCorreos{}, ErrCorreosNoAutenticado
	}
	return OrdenCorreos{actor: copia, proveedor: proveedor}, nil
}

func (o OrdenCorreos) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil || (o.actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate && o.actor.Principal.AuthMethod != vecdomain.AuthMethodDNIe) {
		return vecdomain.ContextoActor{}, ErrCorreosNoAutenticado
	}
	return o.actor.Clonar()
}
func (o OrdenCorreos) Proveedor() ProveedorMaterialCorreos { return o.proveedor }

type PeticionCorreo struct {
	VersionEsperada uint64 `json:"version_esperada"`
	ClaveOperacion  string `json:"clave_operacion"`
	CorreoRef       string `json:"correo_ref,omitempty"`
	Direccion       string `json:"direccion,omitempty"`
	Codigo          string `json:"codigo,omitempty"`
	SustitutoRef    string `json:"sustituto_ref,omitempty"`
}

// Sólo lleva el digest semántico de la entrada. Nunca el código de verificación.
type MaterialCorreos struct {
	PersonaRef      string
	PerfilRef       string
	Accion          string
	FinalidadRef    string
	VersionEsperada uint64
	ClaveOperacion  string
	HuellaPeticion  string
	CorreoRef       string
	SustitutoRef    string
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
// nunca SHA-256 simple de una dirección adivinable.
type SobreDireccionCorreo struct {
	CorreoRef      string
	Version        uint64
	ClaveRef       string
	Nonce          []byte
	Cifrado        []byte
	HuellaIgualdad []byte
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
