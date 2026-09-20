export const MENSAJES_AUDITORIA_ES = Object.freeze({
  vista_todos: "Toda la muestra", vista_accesos: "Accesos", vista_cambios: "Cambios", vista_autorizacion: "Autorización", vista_documentos: "Documentos", vista_conectores: "Conectores",
  sobrelinea: "Auditoría · presentación RRHH", titulo: "Auditoría y trazabilidad", descripcion: "Superficie visual con datos ficticios y minimizados. No constituye un registro de auditoría, una evidencia ni una autorización.",
  muestra_navegable: "Muestra navegable", eventos_visibles: "Eventos sintéticos visibles", contador_visibles: "{visibles} de {total}", tabla_region: "Tabla de eventos sintéticos", tabla_caption: "Esta tabla no es un registro de auditoría real.",
  instante: "Instante", modulo: "Módulo", operacion: "Operación", resultado: "Resultado", detalle: "Detalle", ver_muestra: "Ver muestra", sin_resultados: "No hay eventos sintéticos que coincidan con los filtros.",
  detalle_no_disponible: "Detalle no disponible", detalle_sin_seleccion: "Seleccione un evento de la muestra para revisar su estructura minimizada.", detalle_seleccionado: "Detalle seleccionado · sintético", actor_enmascarado: "Actor enmascarado", recurso: "Recurso", decision: "Decisión", recibo: "Recibo", huella: "Huella", correlacion: "Correlación",
  sin_validez_titulo: "Sin validez probatoria.", sin_validez_descripcion: "Las referencias son ficticias y no se pueden verificar desde esta pantalla.", verificar_evidencia: "Verificar evidencia", verificar_evidencia_motivo: "Requiere evidencia durable, permiso y servicio de verificación.", abrir_dato_personal: "Abrir dato personal", abrir_dato_personal_motivo: "Requiere autorización por campos, ámbito y finalidad.",
  vistas_aria: "Vistas de Auditoría", todos_modulos: "Todos los módulos", filtrar_operacion: "Filtrar operación", todos_resultados: "Todos los resultados", todo_periodo: "Todo el periodo", periodo_18: "18 septiembre 2026", periodo_17: "17 septiembre 2026", periodo_16: "16 septiembre 2026", aplicar_filtros: "Aplicar filtros",
  acciones_reservadas: "Acciones reservadas", exportacion_conservacion: "Exportación y conservación", limites_descripcion: "La interfaz permite explorar la muestra local; no exporta, verifica, abre información personal, conserva ni elimina registros.", exportar_resultados: "Exportar resultados", exportar_resultados_motivo: "Pendiente de exportación firmada, filtros autorizados y control de alcance.", conservar_eliminar: "Conservar / eliminar", conservar_eliminar_motivo: "Pendiente de política de conservación, bloqueo y autorización segregada.",
  anuncio_vista: "Vista {vista} seleccionada sobre datos ficticios.", anuncio_filtros: "Filtros aplicados únicamente sobre la muestra sintética.",
});

/** Traductor cerrado para esta superficie; evita silencios ante etiquetas no catalogadas. */
export function crearTraductorAuditoria(mensajes = MENSAJES_AUDITORIA_ES) {
  return (clave, parametros = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new RangeError(`mensaje de Auditoría no definido: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(parametros[nombre] ?? `{${nombre}}`));
  };
}
