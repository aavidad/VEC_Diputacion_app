export const MENSAJES_DOCUMENTOS_ES = Object.freeze({
  error_vista: "Documentos no disponible",
  miga: "Portal del Empleado → Documentos",
  titulo: "Documentos del expediente",
  ayuda_etiqueta: "Ayuda sobre documentos",
  aclaracion_firma: "Pendiente de firma indica que aún no existe una firma acreditada. El número VEC es interno. Preparar una notificación no acredita su entrega. Un documento con custodia externa lo guarda otro sistema: VEC conserva su huella, no su contenido.",
  expediente: "Expediente",
  consultar: "Consultar",
  cargar_mas: "Cargar más",
  seleccione_expediente: "Seleccione un expediente.",
  referencia_invalida: "Introduzca una referencia de expediente válida.",
  no_configurado: "Consulta documental no disponible.",
  cargando: "Consultando documentos…",
  disponible: "Documentos disponibles",
  vacio: "No hay documentos en este expediente.",
  denegado: "Acceso denegado.",
  error: "No se pudieron consultar los documentos.",
  descargando: "Recuperando el original…",
  descarga_iniciada: "Descarga iniciada.",
  descarga_error: "No se pudo descargar el original.",
  tabla_documentos: "Documentos del expediente",
  col_documento: "Número VEC",
  col_tipo: "Tipo",
  col_version: "Versión",
  col_firma: "Firma",
  col_accion: "Acción",
  firma_borrador: "Borrador",
  firma_pendiente_firma: "Pendiente de firma",
  firma_sin_acreditar: "Firma sin acreditar",
  tipo_comision: "Comisión de servicio",
  tipo_justificante: "Justificante de comisión",
  tipo_contratacion: "Documento de contratación temporal",
  tipo_generico: "Documento",
  descargar: "Descargar original",
  descargar_de: "Descargar original del documento {numero}",
  custodia_externa: "Custodia externa",
  huella_de: "Huella SHA-256: {huella}",
});

export function crearTraductorDocumentos(mensajes = MENSAJES_DOCUMENTOS_ES) {
  return (clave, variables = {}) => {
    const texto = mensajes[clave];
    if (typeof texto !== "string") return clave;
    return texto.replace(/\{([a-z_]+)\}/gu, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
}
