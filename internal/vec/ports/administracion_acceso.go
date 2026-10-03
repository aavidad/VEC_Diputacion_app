package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// AutoridadAdministracionAcceso amplía la autoridad administrativa existente;
// no crea otro registro de cuentas, bloqueos o sesiones. La solicitud selecciona
// una persona. SOLO la fuente central autorizada enumera su conjunto COMPLETO
// de cuentas, incluidas todas las superficies, cuentas privilegiadas e inactivas.
// No se admite filtrar por lo visible, truncar ni tomar una lista del cliente.
//
// La implementación nominal exige Administrador Aplicación vigente del catálogo
// fijo, actor/contexto/evidencia/asignación centrales y permiso PDP V3 exacto.
// Sistemas, nombres de rol y permisos enviados por petición no conceden acceso.
// Cada lectura, propuesta, rechazo, cierre y replay usa la auditoría común.
// Proponer no altera accesos. Cerrar resuelve su material de la propuesta
// almacenada, revalida a ambas personas administradoras y exige aprobador distinto
// del proponente y del destinatario; tampoco permite auto-bloqueo/reactivación.
//
// La aprobación es UNA transacción: consumo V3, CAS de persona, conjunto íntegro,
// revisiones de cuenta y continuidad administrativa IS9, cambios por las
// primitivas centrales IS2, historia, auditoría y recibo. Cada cuenta que cambia
// estado avanza exactamente una revisión; las ya coincidentes se cotejan sin
// otra escritura. Un fallo aborta el conjunto completo. No se admite implementar
// el efecto llamando por HTTP ni confirmando cuenta a cuenta.
//
// La barrera central de agrupación debe compartirse con TODOS los escritores
// que den de alta, retiren o re-enlacen cuentas y mantenerse hasta COMMIT.
// PersonaVersion y una huella/relectura, incluso bajo SERIALIZABLE, NO prueban
// por sí solas esa barrera. Además, el gobierno central debe impedir nuevas
// cuentas activas de la persona mientras siga bloqueada. Si la autoridad de
// identidad no puede acreditar ambas garantías, todos estos métodos deniegan:
// no se inventa un contador, bandera de cliente o almacén sustitutivo.
//
// Invalidar se hace mediante el avance de las revisiones centrales que cotejan
// revalidar_sesion_y_cuentas_v1/revalidar_autenticacion_actor_v1. Reactivar no
// devuelve vigencia a sesiones, credenciales, perfiles o asignaciones anteriores
// revocadas: exige nueva autenticación y la autorización central vigente.
//
// Idempotencia: misma operación/material recupera propuesta/cierre/recibo
// originales, otra huella falla. La recuperación revalida el acceso actual y
// registra su nueva correlación; el recibo conserva la correlación del efecto.
// ResolverConjuntoCuentasPersona recupera la preimagen inmutable original cuando
// ya existe esa operación, después de revalidar el acceso. No la sustituye por
// estados posteriores que impedirían recuperar una propuesta ya aplicada.
// Este corte NO aporta un adaptador nominal ni una garantía PostgreSQL instalada.
type AutoridadAdministracionAcceso interface {
	ResolverConjuntoCuentasPersona(context.Context, domain.SolicitudPropuestaAdministracionAcceso) (domain.PreimagenConjuntoCuentasAdministracionAcceso, error)
	ProponerCambioAccesoPersona(context.Context, domain.OrdenPropuestaAdministracionAcceso) (domain.PropuestaAdministracionAcceso, error)
	CerrarCambioAccesoPersona(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (domain.CierreAdministracionAcceso, error)
}
