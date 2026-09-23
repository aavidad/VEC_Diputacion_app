/** Catálogo de la vista S6. La evidencia llega exclusivamente de una proyección autorizada. */
export const MENSAJES_SELECCION_COMUNICACIONES_ES = Object.freeze({
  sobrelinea: "Selección · comunicaciones", titulo: "Comunicaciones con aspirantes",
  descripcion: "Prepare y revise comunicaciones del proceso selectivo. El estado del transporte, la entrega y la lectura se consultan por separado.",
  ayuda: "S6 contempla correo y SMS. La selección de destinatarios, la plantilla y la revisión deben proceder de fuentes autorizadas. Un transporte aceptado no acredita entrega ni lectura; un aviso tampoco constituye una notificación administrativa.",
  ayuda_etiqueta: "Ayuda sobre comunicaciones de selección",
  estado_cargando: "Cargando comunicaciones autorizadas…", estado_no_configurado: "Sin fuente de comunicaciones de selección conectada.",
  estado_vacio: "No hay comunicaciones de selección en esta consulta.", estado_denegado: "La consulta de comunicaciones no está autorizada para este perfil y ámbito.",
  estado_error: "No se pudo consultar la fuente de comunicaciones. Inténtelo más tarde.",
  estado_disponible: "Comunicaciones consultadas desde la fuente autorizada.",
  fuente_pendiente: "No se han consultado destinatarios ni se ha generado ningún envío.",
  preparacion_titulo: "Preparar una nueva comunicación", preparacion_subtitulo: "Contenido, audiencia y aprobación antes de solicitar el transporte.",
  canal: "Canal", canal_pendiente: "Sin canal corporativo configurado", canal_configurado: "Canal corporativo declarado por la fuente",
  audiencia: "Destinatarios", audiencia_pendiente: "Pendientes de consulta autorizada", plantilla: "Plantilla", plantilla_pendiente: "Pendiente de plantilla versionada",
  revisar: "Revisar", revisar_motivo: "Falta el caso de uso de revisión con autorización e historia.",
  enviar: "Enviar", enviar_motivo_canal: "No hay canal corporativo configurado para el envío.",
  enviar_motivo_conector: "Falta el conector de envío autorizado, con idempotencia, auditoría y recibo real.",
  bandeja_titulo: "Comunicaciones preparadas", bandeja_subtitulo: "Consulta de comunicaciones del proceso, sin efecto de despacho.",
  buscar: "Buscar por asunto o referencia", filtrar: "Filtrar", todas: "Todos los canales", correo: "Correo electrónico", sms: "SMS",
  tabla_region: "Tabla de comunicaciones de selección", tabla_caption: "Comunicaciones consultadas", asunto: "Asunto", referencia: "Referencia", estado_preparacion: "Preparación", estado_revision: "Revisión", ver: "Ver seguimiento", sin_resultados: "Ninguna comunicación coincide con el filtro.",
  detalle_titulo: "Seguimiento de una comunicación", detalle_ayuda: "Cada hito muestra solo la evidencia recibida de la fuente. La ausencia de dato se conserva como sin constancia.",
  sin_seleccion: "Seleccione una comunicación de la lista para ver sus hitos.",
  transporte: "Transporte", entrega: "Entrega", lectura: "Lectura", preparacion: "Preparación", revision: "Revisión",
  destinatarios_resumen: "Audiencia autorizada", fecha: "Fecha", referencia_evidencia: "Referencia de evidencia",
  sin_constancia: "Sin constancia", dato_no_disponible: "No disponible", limite: "Sin entrega o lectura acreditada no se muestra un acuse.",
  estado_pendiente: "Pendiente", estado_preparada: "Preparada", estado_revisada: "Revisada", estado_observada: "Con observaciones",
  estado_solicitado: "Solicitado", estado_aceptado: "Aceptado por transporte", estado_fallido: "Fallido",
  estado_entregado: "Entregado", estado_leido: "Leído", estado_sin_constancia: "Sin constancia",
});
const CLAVES = Object.keys(MENSAJES_SELECCION_COMUNICACIONES_ES);
export function crearTraductorSeleccionComunicaciones(catalogo = MENSAJES_SELECCION_COMUNICACIONES_ES) {
  if (!catalogo || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) throw new TypeError("catálogo S6 incompleto");
  return (clave) => {
    if (!Object.hasOwn(catalogo, clave)) throw new TypeError(`clave i18n S6 desconocida: ${clave}`);
    return catalogo[clave];
  };
}
