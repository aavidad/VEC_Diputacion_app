package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrImagenProhibida    = errors.New("documentos: imagen prohibida")
	ErrImagenConflicto    = errors.New("documentos: reserva de imagen en conflicto")
	ErrImagenNoDisponible = errors.New("documentos: imagen no disponible")
	ErrImagenInvalida     = errors.New("documentos: imagen invalida")
)

const (
	AccionImagenReservar    = "documentos.imagen.reservar"
	AccionImagenRecuperar   = "documentos.imagen.recuperar"
	AccionImagenConfirmar   = "documentos.imagen.confirmar"
	AccionImagenDisponible  = "documentos.imagen.disponible"
	AccionImagenAbrirPropia = "documentos.imagen.abrir_propia"
	AccionImagenAbrirAjena  = "documentos.imagen.abrir_ajena"
	AudienciaImagenPersonal = "portal_personal_autenticado"
	AudienciaImagenInterna  = "portal_interno_autenticado"
	FinalidadImagenPropia   = "finalidad:usuarios:imagen-propia:v1"
	FinalidadImagenInterna  = "finalidad:usuarios:imagen-directorio-interno:v1"
)

type OperacionImagen struct {
	Actor             vecdomain.ContextoActor
	TitularPersonaRef string
	Audiencia         string
	Finalidad         string
	Accion            string
	DocumentoRef      string
	ClaveOperacion    string
	HuellaPeticion    string
}

// Esta autoridad es propia de Documentos. Debe emitir material nominal V3
// para actor, perfil, titular, audiencia, finalidad, acción y referencia exactos.
// El registro consume el material en la misma transacción que estado y auditoría.
type AutoridadImagen interface {
	AutorizarImagen(context.Context, OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// RegistroImagen es propiedad exclusiva de Documentos: unicidad persona+clave,
// historia y recibo durables. Nunca consulta tablas de Usuarios ni del almacén.
// Los métodos de transición aceptan solo el objeto exacto verificado por el
// caso de uso; el adaptador debe aplicar CAS y guardar versión/SHA/tamaño.
type RegistroImagen interface {
	RecuperarImagen(context.Context, OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReservaImagen, bool, error)
	ReservarImagen(context.Context, OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, domain.IdentidadCustodiaImagen) (ReservaImagen, error)
	RegistrarObjetoImagen(context.Context, OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ReservaImagen, vecports.ObjetoAlmacenado) (ReservaImagen, error)
	AdmitirImagen(context.Context, OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ReservaImagen, vecports.ObjetoAlmacenado) (ReservaImagen, error)
	ConfirmarImagen(context.Context, OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ReservaImagen) (ReservaImagen, error)
	LeerImagen(context.Context, OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReservaImagen, bool, error)
}

type ReservaImagen struct {
	Identidad        domain.IdentidadCustodiaImagen
	Estado           domain.EstadoCustodiaImagen
	ObjetoCuarentena vecports.ObjetoAlmacenado
	ObjetoAdmitido   vecports.ObjetoAlmacenado
}

// La política de análisis debe comprobar la versión exacta en cuarentena.
// Solo evidencia limpia habilita promoción; ausencia o fallo deniegan.
type AdmisorImagen interface {
	EvidenciaLimpia(context.Context, vecports.ObjetoAlmacenado) (string, error)
}

// ContextosAlmacenImagen obtiene capacidades opacas de la autoridad de almacén.
// Cada contexto está ligado a la operación nominal y al objeto exacto. No es
// fachada de bytes: la aplicación usa directamente AlmacenObjetos.
type ContextosAlmacenImagen interface {
	EscribirCuarentena(context.Context, OperacionImagen, domain.IdentidadCustodiaImagen, int64) (vecports.ContextoOperacionAlmacen, error)
	PromoverAdmitida(context.Context, OperacionImagen, ReservaImagen) (vecports.ContextoOperacionAlmacen, error)
	AbrirAdmitida(context.Context, OperacionImagen, ReservaImagen) (vecports.ContextoOperacionAlmacen, error)
}

// Consulta a Usuarios en cada apertura, inmediatamente antes de leer del
// almacén. Un recibo anterior o la mera autenticación no mantienen acceso.
type ReferenciaActivaUsuarios interface {
	ReferenciaActiva(context.Context, OperacionImagen) (bool, error)
}
