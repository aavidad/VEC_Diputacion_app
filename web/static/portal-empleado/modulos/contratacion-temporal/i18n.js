/** Textos castellanos de la interfaz de alta; la vista solo consume claves. */

export const MENSAJES_CONTRATACION_TEMPORAL_ES = Object.freeze({
  sobrelinea: "Contratación temporal · Solicitud del centro",
  titulo: "Nueva solicitud de contratación temporal",
  descripcion:
    "Registre la necesidad del centro. La solicitud pasará a revisión de RRHH después de su confirmación.",
  alcance:
    "Esta pantalla registra el alta inicial. No valida la retención de crédito ni decide la vía de cobertura.",
  progreso_etiqueta: "Progreso del alta",
  progreso_datos: "Datos",
  progreso_revision: "Revisión",
  progreso_recibo: "Recibo",
  estado_disponible: "Servicio preparado para revisar y registrar la solicitud",
  estado_no_disponible: "Servicio no disponible",
  estado_no_disponible_detalle:
    "La capacidad, los catálogos y el ejecutor real deben estar conectados antes de registrar.",
  estado_enviando: "Registrando la solicitud. No cierre esta pantalla.",
  estado_cancelando: "Cancelando la espera de respuesta.",
  estado_cancelado:
    "La espera se canceló y el resultado puede ser indeterminado. Reintente solo con el mismo contenido.",
  estado_error:
    "No se pudo confirmar el registro. Los datos se conservan para un reintento seguro.",
  estado_recibo_invalido:
    "La respuesta no pudo verificarse. No se muestra ningún resultado; conserve los datos y contacte con soporte.",
  errores_titulo: "Revise los campos indicados",
  errores_descripcion: "La solicitud no está lista para pasar a revisión.",
  campo_obligatorio: "Campo obligatorio",
  seleccionar: "Seleccione una opción",
  centro_leyenda: "Centro y necesidad",
  centro_ref: "Centro solicitante",
  centro_ayuda: "Centro obtenido del catálogo interno vigente.",
  contacto_ref: "Persona responsable referenciada",
  contacto_ayuda:
    "Referencia interna asociada al centro. No introduzca nombre, correo, DNI ni otros datos personales.",
  categoria_ref: "Categoría",
  categoria_ayuda: "Categoría obtenida del catálogo interno.",
  grupo_subgrupo: "Grupo o subgrupo",
  grupo_ayuda: "Grupo asociado a la categoría seleccionada.",
  motivo_clave: "Motivo",
  motivo_ayuda: "Motivo gobernado por el catálogo vigente.",
  detalle_periodo_leyenda: "Detalle y periodo previsto",
  detalle: "Detalle de la necesidad",
  detalle_ayuda: "Explique la necesidad administrativa sin incluir datos personales innecesarios.",
  contador_caracteres: "{actual} de {maximo} caracteres",
  inicio: "Fecha prevista de inicio",
  fin: "Fecha prevista de fin",
  observaciones: "Observaciones",
  observaciones_ayuda: "Información complementaria necesaria para tramitar la solicitud.",
  rc_leyenda: "Retención de crédito",
  rc_existe: "¿Existe retención de crédito?",
  si: "Sí",
  no: "No",
  rc_aviso:
    "La declaración aportada por el centro no sustituye la validación presupuestaria posterior de RRHH.",
  rc_numero: "Número o referencia de RC",
  rc_fecha: "Fecha de RC",
  rc_importe: "Importe exacto",
  rc_importe_ayuda: "Euros con dos decimales. Se enviará como céntimos enteros y moneda EUR.",
  rc_documento_ref: "Documento de RC incorporado",
  documentos_leyenda: "Documentación incorporada",
  documentos_ayuda:
    "Seleccione solo referencias ya incorporadas al expediente documental. Esta pantalla no sube archivos.",
  documentos_vacios: "No hay documentos incorporados disponibles.",
  revisar: "Revisar solicitud",
  volver_editar: "Volver a editar",
  confirmar: "Confirmar y registrar",
  reintentar: "Reintentar registro",
  cancelar_envio: "Cancelar espera",
  revision_sobrelinea: "Comprobación previa",
  revision_titulo: "Revise la solicitud antes de registrarla",
  revision_aviso:
    "Al confirmar se solicitará el alta del expediente. Compruebe especialmente el periodo y la RC.",
  resumen_centro: "Centro",
  resumen_contacto: "Persona responsable",
  resumen_categoria: "Categoría",
  resumen_grupo: "Grupo o subgrupo",
  resumen_motivo: "Motivo",
  resumen_detalle: "Detalle",
  resumen_periodo: "Periodo previsto",
  resumen_rc: "Retención de crédito",
  resumen_rc_no: "No declarada",
  resumen_rc_si: "Declarada",
  resumen_documentos: "Documentos adjuntos",
  resumen_sin_documentos: "Sin documentación complementaria",
  resumen_observaciones: "Observaciones",
  resumen_sin_observaciones: "Sin observaciones",
  recibo_sobrelinea: "Registro confirmado",
  recibo_titulo: "Solicitud registrada",
  recibo_descripcion:
    "Conserve las referencias del expediente y del recibo para el seguimiento administrativo.",
  recibo_expediente_ref: "Referencia del expediente",
  recibo_numero_visible: "Número de expediente",
  recibo_version: "Versión",
  recibo_ref: "Referencia del recibo",
  recibo_fecha: "Fecha de confirmación",
  error_contrato_cerrado: "La entrada contiene campos no admitidos.",
  error_opcion_catalogo: "Seleccione una opción disponible del catálogo.",
  error_texto_obligatorio:
    "Introduzca un texto válido, sin espacios extremos y con un máximo de 4.000 caracteres.",
  error_texto_opcional:
    "Use un máximo de 4.000 caracteres y elimine espacios extremos o caracteres de control.",
  error_fecha: "Introduzca una fecha civil válida.",
  error_periodo: "La fecha de fin no puede ser anterior a la fecha de inicio.",
  error_booleano: "Indique si existe retención de crédito.",
  error_referencia: "Introduzca una referencia opaca válida.",
  error_importe: "Introduzca un importe positivo con dos decimales.",
  error_rc_residual: "Si no existe RC, sus datos asociados deben quedar vacíos.",
  error_adjuntos: "Seleccione como máximo 64 documentos válidos y sin duplicados.",
  error_generico: "El valor no es válido.",
});

export function crearTraductorContratacionTemporal(sobrescrituras = {}) {
  if (sobrescrituras === null || typeof sobrescrituras !== "object"
    || Array.isArray(sobrescrituras)) {
    throw new TypeError("mensajes de contratación temporal no válidos");
  }
  const mensajes = { ...MENSAJES_CONTRATACION_TEMPORAL_ES, ...sobrescrituras };
  for (const [clave, valor] of Object.entries(mensajes)) {
    if (typeof valor !== "string" || valor.trim() === "") {
      throw new TypeError(`mensaje ${clave} no válido`);
    }
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new Error(`falta la traducción ${clave}`);
    return Object.entries(variables).reduce(
      (texto, [nombre, valor]) => texto.replaceAll(`{${nombre}}`, String(valor)),
      mensajes[clave],
    );
  };
}
