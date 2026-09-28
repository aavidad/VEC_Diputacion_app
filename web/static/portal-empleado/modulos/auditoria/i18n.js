import { IDIOMA_ACTUAL } from "../../../comun/idioma.js";

/** Catálogo del módulo registrado bajo el i18n común del portal. */
export const MENSAJES_AUDITORIA_ES = Object.freeze({
  titulo: "Auditoría",
  ayuda_aria: "Mostrar u ocultar ayuda de Auditoría",
  ayuda_titulo: "Cómo consultar la auditoría",
  ayuda_alcance: "Al abrir esta pantalla se consultan los últimos 30 días. Puede acotar las fechas; el intervalo máximo es de 31 días y «Hasta» no se incluye.",
  ayuda_lectura: "La consulta queda registrada. Solo se muestran campos autorizados y sus huellas; una huella no revela el valor original. La ausencia de datos no acredita ausencia de cambios.",
  filtros_titulo: "Filtros",
  expediente: "Expediente", participacion: "Participación", sin_expediente: "Abra un registro autorizado",
  desde: "Desde (opcional)", hasta: "Hasta (opcional, no incluido)", actor: "Persona",
  consultar: "Consultar", reintentar: "Reintentar", resultados_titulo: "Resultados",
  configuracion_ejemplo: "Muestra ficticia: los nombres y números de ejemplo no identifican a personas ni expedientes reales.",
  estado_no_configurado: "Abra Auditoría desde un expediente o participación autorizados.",
  estado_cargando_opciones: "Comprobando opciones de consulta…",
  estado_esperando: "Consultando los últimos 30 días…",
  estado_cargando: "Consultando auditoría…",
  estado_disponible: "Registros encontrados",
  estado_vacio: "Sin registros para estos filtros.",
  estado_denegado: "Acceso denegado. No se muestran datos.",
  estado_error: "No se pudo completar la consulta. Reintente.",
  estado_invalido: "Revise las fechas: «Hasta» debe ser posterior a «Desde» y el intervalo no puede superar 31 días.",
  tabla_aria: "Registros de auditoría", fecha: "Fecha y hora", accion: "Acción",
  resultado: "Resultado", detalle: "Detalles", ver_cambio: "Ver detalle técnico",
  numero: "Número", dato_ficticio: "ficticio", persona_no_disponible: "Nombre no disponible",
  numero_no_disponible: "Número no disponible", accion_relacion_actualizada: "Actualizó la relación de servicio",
  accion_participacion_cambiada: "Cambió la participación en la bolsa",
  accion_otra: "Registró una actuación", resultado_confirmado: "Confirmado",
  resultado_denegado: "Denegado", resultado_otro: "Otro resultado",
  detalle_tecnico: "Referencias técnicas", campo_actor_ref: "Referencia de persona",
  campo_accion_ref: "Código de acción", campo_resultado_ref: "Código de resultado",
  campo_registro_ref: "Referencia del registro", campo_expediente_ref: "Referencia del expediente",
  campo_modulo: "Módulo", campo_expediente: "Expediente relacionado",
  campo_recibo: "Recibo", campo_fuente: "Fuente", campo_motivo: "Motivo",
  sin_dato: "No consta", antes: "Antes", despues: "Después",
  sin_valores: "Sin valores visibles", huella: "Huella SHA-256",
  valores_no_disponibles: "La fuente no aporta valores anteriores y posteriores para este registro.",
  paginacion: "Páginas de auditoría", anterior: "Anterior", siguiente: "Siguiente",
  pagina: "Página {numero}",
});

export const MENSAJES_AUDITORIA_EN = Object.freeze({
  titulo: "Audit records", ayuda_aria: "Show or hide audit help", ayuda_titulo: "How to check audit records",
  ayuda_alcance: "The last 30 days are checked when you open this screen. You can narrow the dates; the maximum range is 31 days and ‘Until’ is excluded.",
  ayuda_lectura: "Your query is recorded. Only authorised fields and their fingerprints are shown. A fingerprint does not reveal the original value. Missing data does not mean no changes were made.",
  filtros_titulo: "Filters", expediente: "Case", participacion: "Participation", sin_expediente: "Open an authorised record",
  desde: "From (optional)", hasta: "Until (optional, excluded)", actor: "Person",
  consultar: "Search", reintentar: "Try again", resultados_titulo: "Results",
  configuracion_ejemplo: "Fictitious sample: the example names and numbers do not identify real people or cases.",
  estado_no_configurado: "Open Audit records from an authorised case or participation.",
  estado_cargando_opciones: "Checking query options…", estado_esperando: "Checking the last 30 days…",
  estado_cargando: "Checking audit records…", estado_disponible: "Records found",
  estado_vacio: "No records match these filters.", estado_denegado: "Access denied. No data is shown.",
  estado_error: "The query could not be completed. Try again.",
  estado_invalido: "Check the dates: ‘Until’ must be later than ‘From’ and the range cannot exceed 31 days.",
  tabla_aria: "Audit records", fecha: "Date and time", accion: "Action",
  resultado: "Result", detalle: "Details", ver_cambio: "View technical details",
  numero: "Number", dato_ficticio: "fictitious", persona_no_disponible: "Name unavailable",
  numero_no_disponible: "Number unavailable", accion_relacion_actualizada: "Updated the employment record",
  accion_participacion_cambiada: "Changed the pool participation",
  accion_otra: "Recorded an action", resultado_confirmado: "Confirmed",
  resultado_denegado: "Denied", resultado_otro: "Other result",
  detalle_tecnico: "Technical references", campo_actor_ref: "Person reference",
  campo_accion_ref: "Action code", campo_resultado_ref: "Result code",
  campo_registro_ref: "Record reference", campo_expediente_ref: "Case reference",
  campo_modulo: "Module", campo_expediente: "Related case", campo_recibo: "Receipt",
  campo_fuente: "Source", campo_motivo: "Reason", sin_dato: "Not recorded",
  antes: "Before", despues: "After", sin_valores: "No visible values", huella: "SHA-256 fingerprint",
  valores_no_disponibles: "The source does not provide before and after values for this record.",
  paginacion: "Audit pages", anterior: "Previous", siguiente: "Next", pagina: "Page {numero}",
});

export function crearTraductorAuditoria(mensajes = IDIOMA_ACTUAL === "en" ? MENSAJES_AUDITORIA_EN : MENSAJES_AUDITORIA_ES) {
  return (clave, parametros = {}) => {
    if (!Object.hasOwn(mensajes, clave) || typeof mensajes[clave] !== "string") throw new RangeError(`mensaje de Auditoría no definido: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(parametros[nombre] ?? `{${nombre}}`));
  };
}
