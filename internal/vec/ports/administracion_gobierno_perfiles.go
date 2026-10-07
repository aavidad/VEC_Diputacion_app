package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// AutoridadGobiernoPerfiles amplía la autoridad administrativa central existente.
// No publica mediante el adaptador de memoria ni mediante el publicador técnico
// CT/DBA. El catálogo completo procede de módulos y perfiles publicados de la
// fuente central, nunca del archivo o petición del cliente; omitir una semilla
// fija no puede convertirla en administrable. Los fijos sólo admiten lectura.
//
// Proponer almacena el material y audita, sin publicar ni cambiar asignaciones.
// Cerrar recupera la propuesta inmutable; revalida DOS personas Administrador
// Aplicación distintas, categoría fija, persona/perfil/asignación, evidencia
// y permiso PDP V3 exactos. Sistemas y permisos declarados no tienen autoridad.
// No acepta identidad, autor de publicación ni fechas como prueba desde archivos.
//
// Una aprobación consume V3 en la MISMA transacción que CAS del catálogo,
// ausencia del RolID al crear o base exacta al versionar, control de vigencia,
// historia, auditoría común y recibo. Las publicaciones para un mismo RolID
// comparten la barrera central y rechazan versiones paralelas o saltos.
// La nueva VersionRol recibe autor/fecha reales de la autoridad y control inicial
// habilitado revision1. Versiones anteriores permanecen inmutables y ninguna
// asignación se migra, reactiva ni se otorga por publicar una definición.
//
// Deshabilitar retira sólo la VERSION EXACTA seleccionada: nuevo control
// retirada/revisión+1 y puntero CAS, sin reescribir VersionRol o asignaciones.
// Es irreversible, no interpreta otra versión retirada como retirada de ésta
// y no asegura que otras versiones del mismo RolID hayan quedado deshabilitadas.
// Debe comprobar también continuidad/guardas centrales si afecta roles sensibles.
//
// Mismo material/operación recupera recibos originales; otra huella deniega.
// Cada acceso/replay se revalida y audita con correlación actual; el recibo
// conserva la del efecto. El resolver devuelve el catálogo/base originales al
// recuperar una propuesta existente, sin sustituirlos por postimágenes posteriores.
// Si falta fuente, catálogo gobernado, capacidad transaccional o autorización,
// deniega sin fallback. No hay adaptador nominal ni SQL instalado en este corte.
type AutoridadGobiernoPerfiles interface {
	ResolverCatalogoGobiernoPerfil(context.Context, domain.SolicitudPropuestaGobiernoPerfil) (domain.CatalogoAccionesAdministracionV1, error)
	ProponerGobiernoPerfil(context.Context, domain.OrdenPropuestaGobiernoPerfil) (domain.PropuestaGobiernoPerfil, error)
	CerrarGobiernoPerfil(context.Context, domain.SolicitudCierreGobiernoPerfil) (domain.CierreGobiernoPerfil, error)
}
