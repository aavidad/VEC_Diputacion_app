export const MENSAJES_APROBACIONES_ES = Object.freeze({
  sobrelinea: "PORTAFIRMAS · PRESENTACIÓN", titulo: "Aprobaciones y Portafirmas",
  descripcion: "Bandeja visual de revisiones pendientes. No concede autoridad, no remite documentos y no realiza firma.",
  circuito: "Circuito", buscar: "Buscar en la bandeja", buscar_placeholder: "Expediente, asunto o persona", aplicar: "Aplicar filtros",
  resumen_entrega: "Esta bandeja visual permite filtrar y revisar ejemplos. Las decisiones, la firma, el envío y las descargas permanecen deshabilitados.",
  pendiente_identidad: "Identidad y autorización por actor, acción, recurso y ámbito", pendiente_matriz: "Matriz vigente de competencia, delegación y suplencia", pendiente_firma: "Portafirmas, firma electrónica y conservación de originales", pendiente_operacion: "Operación transaccional con auditoría, recibo e idempotencia", pendiente_origen: "Conexión a los módulos origen y sus documentos", conexion: "Backend Go, auditoría y conectores pendientes",
  pendientes_visibles: "Pendientes visibles", bandeja_local: "Bandeja local", alta_prioridad: "Alta prioridad", sin_plazos: "Sin plazos atribuidos", circuitos: "Circuitos", revision_visual: "Revisión visual", firma_multiple: "Firma múltiple", portafirmas_pendiente: "Portafirmas no conectado", firmas_pendientes: "{cantidad} pendiente(s)",
  bandeja: "Bandeja de pendientes", tabla_pendientes: "Pendientes de aprobación", referencia: "Referencia", tipo: "Tipo", solicitante: "Solicitante", prioridad: "Prioridad", recibido: "Recibido", estado: "Estado", acciones: "Acciones", ver_detalle: "Ver detalle", sin_resultados: "No hay elementos para los filtros aplicados.",
  expediente_seleccionado: "EXPEDIENTE SELECCIONADO", unidad: "Unidad", impacto: "Impacto mostrado", seguimiento: "Seguimiento", documentos: "Documentos originales", original_pendiente: "Original no conectado", descarga_pendiente: "Descarga pendiente", motivo_original: "No hay documento original conectado.", linea: "Línea de revisiones", matriz: "Matriz de autoridad", recibo: "Recibo visual: {referencia}. La auditoría durable no está conectada.",
  aprobar: "Aprobar", devolver: "Devolver", rechazar: "Rechazar", firmar: "Firmar en Portafirmas", remitir: "Remitir al siguiente firmante", descargar: "Descargar original", delegar: "Delegar o suplir", motivo_aprobar: "Requiere autorización, operación durable y recibo.", motivo_devolver: "Requiere motivo, autorización y auditoría.", motivo_rechazar: "Requiere motivo, autorización y auditoría.", motivo_firmar: "Portafirmas y firma electrónica no conectados.", motivo_remitir: "Circuito, identidad y envío pendientes.", motivo_descargar: "Documento original no disponible.", motivo_delegar: "Matriz, vigencia y autorización pendientes.",
  suplencias: "Suplencias y delegaciones visibles", suplencias_ayuda: "Información de presentación: no habilita una sustitución ni altera competencias.", tabla_suplencias: "Suplencias y delegaciones", figura: "Figura", titular: "Titular", cobertura: "Cobertura", alcance: "Alcance", vigencia: "Vigencia", detalle_anunciado: "Detalle de {referencia}", visibles_anunciado: "{cantidad} pendientes visibles",
});

export function crearTraductorAprobaciones(mensajes = MENSAJES_APROBACIONES_ES) {
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new TypeError(`clave de Aprobaciones no definida: ${clave}`);
    return String(mensajes[clave]).replace(/\{([a-z_]+)\}/gu, (_, nombre) => String(variables[nombre] ?? `{${nombre}}`));
  };
}
