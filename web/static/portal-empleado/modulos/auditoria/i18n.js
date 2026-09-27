/** Catálogo del módulo registrado bajo el i18n común del portal. */
export const MENSAJES_AUDITORIA_ES = Object.freeze({
  titulo: "Auditoría",
  ayuda_aria: "Mostrar u ocultar ayuda de Auditoría",
  ayuda_titulo: "Cómo consultar la auditoría",
  ayuda_alcance: "Abra Auditoría desde un expediente autorizado y elija un intervalo máximo de 31 días. El instante «Hasta» queda fuera del intervalo. La finalidad y el motivo proceden del catálogo vigente.",
  ayuda_lectura: "La consulta queda registrada. Solo se muestran campos autorizados y sus huellas; una huella no revela el valor original. La ausencia de datos no acredita ausencia de cambios.",
  filtros_titulo: "Filtros",
  expediente: "Expediente", sin_expediente: "Abra un expediente autorizado",
  desde: "Desde", hasta: "Hasta", actor: "Actor",
  consultar: "Consultar", reintentar: "Reintentar", resultados_titulo: "Resultados",
  configuracion_ejemplo: "Configuración de ejemplo pendiente de RRHH",
  estado_no_configurado: "Abra Auditoría desde un expediente autorizado.",
  estado_cargando_opciones: "Comprobando opciones de consulta…",
  estado_esperando: "Indique los filtros para consultar.",
  estado_cargando: "Consultando auditoría…",
  estado_disponible: "Registros encontrados",
  estado_vacio: "Sin registros para estos filtros.",
  estado_denegado: "Acceso denegado. No se muestran datos.",
  estado_error: "No se pudo completar la consulta. Reintente.",
  estado_invalido: "Revise el expediente y el intervalo: debe ser mayor que cero y no superar 31 días.",
  tabla_aria: "Registros de auditoría", fecha: "Fecha y hora", accion: "Acción",
  resultado: "Resultado", detalle: "Cambio", ver_cambio: "Ver cambio",
  campo_modulo: "Módulo", campo_expediente: "Expediente relacionado",
  campo_recibo: "Recibo", campo_fuente: "Fuente", campo_motivo: "Motivo",
  sin_dato: "No consta", antes: "Antes", despues: "Después",
  sin_valores: "Sin valores visibles", huella: "Huella SHA-256",
  valores_no_disponibles: "La fuente no aporta valores anteriores y posteriores para este registro.",
  paginacion: "Páginas de auditoría", anterior: "Anterior", siguiente: "Siguiente",
  pagina: "Página {numero}",
});

export function crearTraductorAuditoria(mensajes = MENSAJES_AUDITORIA_ES) {
  return (clave, parametros = {}) => {
    if (!Object.hasOwn(mensajes, clave) || typeof mensajes[clave] !== "string") throw new RangeError(`mensaje de Auditoría no definido: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(parametros[nombre] ?? `{${nombre}}`));
  };
}
