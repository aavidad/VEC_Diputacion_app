export const MENSAJES_DOCUMENTOS_ES = Object.freeze({
  error_vista: "vista de Documentos no disponible", error_documento: "documento de Documentos no disponible",
  sobrelinea: "DOCUMENTOS Y FIRMA", titulo: "Biblioteca documental y circuito de firma",
  descripcion: "Consulta visual de expedientes, versiones y evidencias. Ningún documento de este espacio está firmado, enviado ni descargable.",
  contexto: "Contexto de presentación", rol_rrhh: "RRHH", identidad_aviso: "La identidad de acceso se comprobará al conectar el backend; este ejemplo no concede autorización.",
  filtro: "Filtrar por tipo o título", filtro_placeholder: "Informe, resolución, diligencia…", aplicar_filtro: "Aplicar filtro",
  resumen_documentos: "Documentos mostrados", resumen_documentos_nota: "Ejemplos locales", resumen_firma: "Pendientes de firma", resumen_firma_valor: "2", resumen_firma_nota: "No equivale a firma", resumen_plantillas: "Plantillas", resumen_plantillas_nota: "Catálogo visual", resumen_trazabilidad: "Trazabilidad", resumen_trazabilidad_valor: "Pendiente", resumen_trazabilidad_nota: "Auditoría no conectada",
  entrega_resumen: "La biblioteca y el circuito son una demostración de interfaz; ningún efecto documental se ha realizado.", entrega_repositorio: "Repositorio documental y conservación", entrega_autorizacion: "Autorización por expediente y finalidad", entrega_huella: "Generación y huella original", entrega_firma: "Firma admitida, sellado y validación", entrega_auditoria: "Auditoría, envío y descarga autorizada", entrega_conexion: "Backend Go pendiente",
  biblioteca: "Biblioteca documental", biblioteca_descripcion: "Filtros y selección se realizan únicamente en esta pantalla.", tabla_documentos: "Documentos de presentación", col_documento: "Documento", col_tipo: "Tipo", col_expediente: "Expediente", col_version: "Versión", col_fecha: "Fecha", col_estado: "Estado", col_accion: "Acción", ver_ficha: "Ver ficha", sin_resultados: "No hay documentos que coincidan con el filtro.", ficha_seleccionada: "Ficha seleccionada: {titulo}", filtro_aplicado: "Filtro aplicado: {filtro}", filtro_eliminado: "Filtro eliminado",
  ficha: "Ficha y evidencia", responsable: "Responsable visible", huella: "Huella", conservacion: "Conservación", circuito: "Circuito previsto", acciones_pendientes: "Acciones pendientes de conexión", aclaracion_firma: "Un certificado de autenticación identifica una sesión; no firma este documento. Un certificado FNMT de ejemplo tampoco convierte esta demostración en firma documental.",
  generar_version: "Generar versión", generar_motivo: "La generación requiere plantilla gobernada y backend.", subir_original: "Subir original", subir_motivo: "La carga requiere repositorio, antivirus y autorización.", firmar: "Firmar documento", firmar_motivo: "No hay firma admitida conectada.", verificar: "Verificar externamente", verificar_motivo: "La validación externa no está conectada.", enviar: "Enviar al circuito", enviar_motivo: "El envío requiere autorización y auditoría.", descargar: "Descargar original", descargar_motivo: "La descarga autorizada no está conectada.", generar: "Generar", generar_plantilla_motivo: "La generación documental no está conectada.",
  plantillas: "Plantillas documentales", tabla_plantillas: "Plantillas disponibles para presentación", col_plantilla: "Plantilla", circuito_titulo: "Circuito de firma y verificación", paso_1: "Preparar versión y huella original", paso_2: "Comprobar autorización y firmante admitido", paso_3: "Firmar y sellar con proveedor conectado", paso_4: "Validar, conservar y auditar", paso_5: "Entregar o descargar conforme a autorización", paso_pendiente: "Pendiente de conexión",
});

export function crearTraductorDocumentos(mensajes = MENSAJES_DOCUMENTOS_ES) {
  return (clave, variables = {}) => {
    const texto = mensajes[clave];
    if (typeof texto !== "string") return clave;
    return texto.replace(/\{([a-z_]+)\}/gu, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
}
