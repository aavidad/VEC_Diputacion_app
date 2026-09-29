package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Errores nominales de «Mis correos». La API los traduce a 401/403/409/422/
// 429/503 sin revelar si una dirección existe fuera del conjunto propio.
var (
	ErrCorreosNoAutenticado    = errors.New("usuarios correos: no autenticado")
	ErrCorreosProhibido        = errors.New("usuarios correos: prohibido")
	ErrCorreosConflicto        = errors.New("usuarios correos: conflicto")
	ErrCorreosInvalidos        = errors.New("usuarios correos: peticion invalida")
	ErrCorreosNoDisponible     = errors.New("usuarios correos: no disponible")
	ErrCorreosLimite           = errors.New("usuarios correos: limite de codigos alcanzado")
	ErrCorreosCodigoIncorrecto = errors.New("usuarios correos: codigo incorrecto")
	ErrCorreosCodigoCaducado   = errors.New("usuarios correos: codigo caducado o agotado")
	ErrCorreosYaRegistrado     = errors.New("usuarios correos: direccion ya registrada")
	ErrCorreosMaximo           = errors.New("usuarios correos: maximo de direcciones")
	ErrCorreosEnUso            = errors.New("usuarios correos: direccion en uso para avisos")
)

// CodigoIncorrecto conserva los intentos que quedan para el mismo código.
// errors.Is(err, ErrCorreosCodigoIncorrecto) sigue identificándolo.
type CodigoIncorrecto struct{ IntentosRestantes int }

func (CodigoIncorrecto) Error() string          { return ErrCorreosCodigoIncorrecto.Error() }
func (CodigoIncorrecto) Is(objetivo error) bool { return objetivo == ErrCorreosCodigoIncorrecto }

const (
	FinalidadCorreosPropios          = "finalidad:usuarios:correos-propios:v1"
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

// CamposPermitidosCorreos es el contrato de campos que la concesión V3 y
// AD3-107 exigen por acción. Un cambio aquí exige otra migración AD3.
func CamposPermitidosCorreos(accion string) []string {
	switch accion {
	case AccionConsultarCorreos:
		return []string{"activo", "correo_ref", "direccion", "estado", "version"}
	case AccionAnadirCorreo:
		return []string{"correo_ref", "direccion", "estado", "version"}
	case AccionReenviarCorreo, AccionVerificarCorreo:
		return []string{"correo_ref", "estado", "version"}
	case AccionActivarCorreo:
		return []string{"activo", "correo_ref", "version"}
	case AccionRetirarCorreo:
		return []string{"activo", "correo_ref", "estado", "version"}
	}
	return nil
}

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
// preimagen: la dirección y el código no tienen un SHA-256 público que
// permita un diccionario. Usa HMAC-SHA256 con clave semántica propia.
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
}

type VistaCorreos struct {
	PersonaRef string                `json:"-"`
	Version    uint64                `json:"version"`
	Correos    []domain.CorreoPropio `json:"correos"`
}

// EstadoEnvioCorreo resume la respuesta del relay SMTP a la salida que generó
// la operación. «aceptado» no acredita entrega ni lectura.
type EstadoEnvioCorreo string

const (
	EnvioSinCorreo  EstadoEnvioCorreo = ""
	EnvioAceptado   EstadoEnvioCorreo = "aceptado"
	EnvioNoAceptado EstadoEnvioCorreo = "no_enviado"
	TipoEnvioCodigo                   = "verificacion"
	TipoEnvioAviso                    = "aviso_cambio"
)

type ReciboCorreos struct {
	ReciboRef  string            `json:"recibo_ref"`
	PersonaRef string            `json:"-"`
	Accion     string            `json:"accion"`
	CorreoRef  string            `json:"correo_ref"`
	Version    uint64            `json:"version"`
	FechaUTC   time.Time         `json:"fecha_utc"`
	Replay     bool              `json:"replay"`
	Envio      EstadoEnvioCorreo `json:"envio,omitempty"`
}

// ReservaDesafio vive sólo en memoria hasta el envío. El registro conserva
// huella, clave_ref y vencimiento; Codigo nunca llega a PostgreSQL.
type ReservaDesafio struct {
	DesafioRef   string
	Codigo       string
	HuellaCodigo []byte
	ClaveRef     string
	VenceUTC     time.Time
}

// SobreDireccionCorreo es la dirección cifrada con AEAD ligado a persona,
// correo_ref y versión. HuellaIgualdad es HMAC con clave propia (no SHA-256
// de una dirección adivinable) y sólo sirve para la unicidad por persona.
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
	ConDireccionCorreoDescifrada(context.Context, string, SobreDireccionCorreo, func([]byte) error) error
}

type PreparadorDesafioCorreo interface {
	// Genera con CSPRNG un código de 8 dígitos y su HMAC con clave dedicada y
	// versionada, ligado a persona, correo_ref, desafío y vencimiento.
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

// El registro invoca Comprobar bajo bloqueo de la fila pendiente y consume
// atómicamente el intento o el éxito. Una clave revocada produce error.
type ComprobadorCodigoCorreo interface {
	Comprobar(context.Context, MetadatosDesafioCorreo) (bool, error)
}

// ValidadorCodigoCorreo compara en tiempo constante.
type ValidadorCodigoCorreo interface {
	ComprobarCodigoCorreo(context.Context, MetadatosDesafioCorreo, string) (bool, error)
}

// EnvioPendiente es la salida ya reservada en la misma transacción que el
// efecto. ReservaRef es la única llave para anotar su resultado.
type EnvioPendiente struct {
	EnvioRef   string
	ReservaRef string
	Tipo       string
	CorreoRef  string
	DesafioRef string
	Sobre      SobreDireccionCorreo
}

type ResultadoCorreos struct {
	Recibo ReciboCorreos
	Envios []EnvioPendiente
}

// Aplicar es la única puerta durable de las mutaciones. Verifica/consume V3 y
// CAS en la misma transacción que límites, intentos, estado, historia,
// auditoría, recibo y salida reservada. La misma clave con otra huella es
// ErrCorreosConflicto; el replay devuelve el recibo ORIGINAL sin envíos.
// ConfirmarEnvio anota una sola vez la respuesta del relay con la reserva.
type RegistroCorreos interface {
	ConsultarPropios(context.Context, OrdenCorreos, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (VistaCorreos, error)
	RecuperarOperacion(context.Context, OrdenCorreos, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboCorreos, bool, error)
	Aplicar(context.Context, OrdenCorreos, PeticionCorreo, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, SobreDireccionCorreo, ReservaDesafio, ComprobadorCodigoCorreo) (ResultadoCorreos, error)
	ConfirmarEnvio(context.Context, OrdenCorreos, EnvioPendiente, bool) error
}

// MensajeCorreoPropio es efímero: destino y código existen sólo en memoria
// durante el envío. El transportador redacta asunto y cuerpo desde el
// catálogo i18n y usa EnvioRef para un Message-ID estable.
type MensajeCorreoPropio struct {
	EnvioRef string
	Tipo     string
	Destino  string
	Codigo   string
	VenceUTC time.Time
}

func (MensajeCorreoPropio) String() string   { return "usuarios.MensajeCorreoPropio{redactado}" }
func (MensajeCorreoPropio) GoString() string { return "usuarios.MensajeCorreoPropio{redactado}" }

// TransportadorCorreosPropios devuelve true sólo si el relay aceptó el
// mensaje. Un error o una respuesta incierta se trata como no aceptado.
type TransportadorCorreosPropios interface {
	EnviarCorreoPropio(context.Context, MensajeCorreoPropio) bool
}
