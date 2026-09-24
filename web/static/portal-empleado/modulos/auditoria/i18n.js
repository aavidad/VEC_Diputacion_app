export const MENSAJES_AUDITORIA_ES = Object.freeze({
  sobrelinea: "Control interno · Auditoría", titulo: "Consulta de auditoría",
  descripcion: "Lectura acotada por competencia, recurso y finalidad. Los resultados solo podrán proceder de la autoridad de auditoría.",
  ayuda_aria: "Mostrar u ocultar ayuda de Auditoría", ayuda_titulo: "Cómo funciona esta consulta",
  ayuda_alcance: "Cada lectura requiere una concesión positiva vigente para el actor, la finalidad, el recurso exacto, el periodo y los campos autorizados.",
  ayuda_lectura: "La autoridad de auditoría debe registrar la propia consulta y devolver únicamente los eventos permitidos. Un identificador, menú o perfil visible no concede acceso.",
  ayuda_segregacion: "La persona que administra un proceso no puede alterar sus trazas. La exportación y la conservación requieren un circuito segregado.",
  ayuda_abierta: "Ayuda de Auditoría abierta.", ayuda_cerrada: "Ayuda de Auditoría cerrada.",
  estado_operativo: "Estado operativo", estado_no_configurado: "Consulta no configurada", estado_denegado: "Acceso denegado",
  no_configurado_descripcion: "No hay cliente de consulta autorizado conectado a esta pantalla. No se han solicitado ni mostrado eventos.",
  denegado_descripcion: "La autoridad no ha concedido esta lectura. No se muestra información del recurso ni se conserva una copia en el navegador.",
  consulta_sobrelinea: "Alcance de lectura", consulta_titulo: "Definir una consulta exacta",
  consulta_subtitulo: "Los filtros estarán disponibles cuando el servidor pueda comprobar la concesión y registrar la lectura.",
  condiciones_aria: "Condiciones de consulta", condicion_competencia: "Competencia vigente", condicion_competencia_nota: "Perfil y finalidad autorizados para esta lectura.",
  condicion_ambito: "Recurso y periodo", condicion_ambito_nota: "Referencia exacta y muestra temporal acotada.",
  condicion_lectura: "Respuesta minimizada", condicion_lectura_nota: "Solo campos concedidos y lectura auditada.",
  filtros_aria: "Filtros de consulta no disponibles", recurso_exacto: "Referencia exacta del recurso", recurso_placeholder: "Pendiente de conector autorizado",
  finalidad: "Finalidad", finalidad_placeholder: "Pendiente de concesión", periodo: "Periodo", periodo_placeholder: "Pendiente de autorización",
  consultar: "Consultar eventos", consulta_motivo: "Consulta deshabilitada: falta un cliente que use la autoridad de auditoría y compruebe permiso, ámbito y finalidad.",
  limites_sobrelinea: "Protección de datos", limites_titulo: "Límites de esta pantalla",
  limite_campos: "Sin identidades ni documentos completos en esta vista.", limite_separacion: "Solo lectura; ninguna acción altera las trazas.",
  limite_exportacion: "La exportación necesita permiso, filtros y evidencia propios.",
  exportar: "Exportar resultados", exportar_motivo: "Exportación deshabilitada: falta un circuito autorizado y verificable.",
  resultados_sobrelinea: "Respuesta", resultados_titulo: "Eventos autorizados",
  sin_resultados: "Aún no hay una consulta disponible.", sin_resultados_denegado: "No hay eventos visibles con este acceso.",
  sin_resultados_nota: "La pantalla no utiliza muestras ficticias ni registros locales como resultado de auditoría.",
});

/** Traductor cerrado para esta superficie; evita silencios ante etiquetas no catalogadas. */
export function crearTraductorAuditoria(mensajes = MENSAJES_AUDITORIA_ES) {
  return (clave, parametros = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new RangeError(`mensaje de Auditoría no definido: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(parametros[nombre] ?? `{${nombre}}`));
  };
}
