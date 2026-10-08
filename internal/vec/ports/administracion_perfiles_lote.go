package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// AutoridadLotesAdministracionPerfiles amplía la autoridad durable existente.
// Debe ejecutar una sola transacción SERIALIZABLE para toda la orden: resolver
// actor/evidencia V2, consultar categoría Administrador Aplicación del catálogo
// central, revalidar PDP V3 y consumir autorización junto al CAS de TODAS las
// preimágenes, asignaciones VersionRol/AsignacionPerfil, historia, auditoría y
// recibo. No se admite invocar AplicarActoOrdinario en bucle ni commit parcial.
// Clase, ámbitos y permisos se reconstruyen desde fuentes centrales, nunca
// desde cliente, Principal.Roles, nombres UI o una bandera Sistemas. Sistemas
// no gestiona perfiles; nadie se otorga su propio perfil. Cada rol sensible
// aborta el lote ordinario completo y exige el circuito de doble control.
// El replay verifica nuevamente acceso vigente y devuelve el recibo original;
// una misma referencia con otra huella falla sin efecto. No migra asignaciones
// al publicar una versión del rol. Ámbito unidad/centro y vigencia son explícitos.
// La auditoría existente conserva persona actor, perfil/asignación activa,
// destinataria, cambios, ámbitos, vigencia, motivo, acto opcional, correlación,
// fecha y recibo. Los nombres provienen de lectura central minimizada, no de
// logs técnicos. Propuestas/cierres conservan ambas personas del doble control.
// La huella semántica no incluye correlación ni instantánea del acceso actual.
// El recibo conserva la correlación original del efecto; la auditoría común
// registra además cada acceso/replay con su correlación y autorización actuales.
// Contrato v3: la organización viene de la configuración y la SQL la coteja con
// la fuente central para el actor y para la persona destinataria. «Inmediato»
// significa el instante del COMMIT; un alta «programada» ya vencida o una
// vigencia final no posterior al inicio efectivo se rechazan sin efecto. El
// recibo devuelve la huella de los descriptores de ámbito cotejados.
type AutoridadLotesAdministracionPerfiles interface {
	AplicarLoteOrdinario(context.Context, domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error)
}

// PreparadorLotesAdministracionPerfiles devuelve, para una persona y unidad del
// conjunto del administrador, lo que hace falta para construir un lote: cuenta,
// versiones, procedencia y la huella de cada alta o baja posible. No cambia
// asignaciones; consume una decisión propia de la acción del lote (efecto
// «preparar», atributo preparacion_sha256) y deja registro y auditoría en la
// misma transacción. Una preparación no reserva nada: el lote compara de nuevo.
type PreparadorLotesAdministracionPerfiles interface {
	PrepararLoteOrdinario(context.Context, domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error)
}
