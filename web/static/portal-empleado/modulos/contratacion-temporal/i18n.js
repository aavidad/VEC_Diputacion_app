/** Textos castellanos del módulo; las vistas solo consumen claves. */

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
    "No se pudo confirmar el registro. Si coincide con una solicitud ya registrada, no la envíe de nuevo: revise el expediente existente o cambie el detalle solo si es una petición distinta.",
  estado_recibo_invalido:
    "La respuesta no pudo verificarse. No se muestra ningún resultado; conserve los datos y contacte con soporte.",
  estado_operacion_pendiente:
    "El resultado no puede determinarse todavía. No repita la operación.",
  estado_operacion_pendiente_ayuda:
    "La solicitud queda bloqueada hasta consultar su recibo mediante la recuperación protegida o recibir asistencia de soporte.",
  errores_titulo: "Revise los campos indicados",
  errores_descripcion: "La solicitud no está lista para pasar a revisión.",
  campo_obligatorio: "Campo obligatorio",
  campos_obligatorios: "Los campos marcados con * son obligatorios.",
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
  periodo_ayuda: "El periodo previsto no puede superar cien años civiles.",
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
  rc_importe_ayuda:
    "Euros con dos decimales. Máximo 9.223.372.036.854,77 €. Se enviará como céntimos enteros y moneda EUR.",
  rc_importe_placeholder: "0,00",
  moneda_eur: "EUR",
  rc_documento_ref: "Documento de RC incorporado",
  documentos_leyenda: "Documentación incorporada",
  documentos_adjuntos: "Documentos adjuntos",
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
  error_periodo_maximo: "El periodo previsto no puede superar cien años civiles.",
  error_booleano: "Indique si existe retención de crédito.",
  error_referencia: "Introduzca una referencia opaca válida.",
  error_importe: "Introduzca un importe positivo con dos decimales.",
  error_importe_maximo: "El importe no puede superar 9.223.372.036.854,77 €.",
  error_rc_residual: "Si no existe RC, sus datos asociados deben quedar vacíos.",
  error_adjuntos: "Seleccione como máximo 64 documentos válidos y sin duplicados.",
  error_generico: "El valor no es válido.",
  analisis_sobrelinea: "Contratación temporal · Análisis de RRHH",
  analisis_titulo_registrar: "Registrar análisis de RRHH",
  analisis_titulo_rectificar: "Rectificar análisis de RRHH",
  analisis_descripcion:
    "Revise los campos funcionales gobernados antes de confirmar la operación.",
  analisis_alcance_etiqueta: "Alcance de la operación",
  analisis_alcance:
    "La identidad, la organización, el perfil y la autorización se resuelven fuera de este formulario.",
  analisis_estado_listo: "El análisis está preparado para su revisión.",
  analisis_estado_validacion: "Revise los campos indicados antes de continuar.",
  analisis_estado_enviando: "Enviando una única petición. Espere la confirmación.",
  analisis_estado_cancelando: "Cancelando la espera de respuesta.",
  analisis_estado_confirmado: "El recibo del análisis se ha verificado.",
  analisis_estado_indeterminado:
    "El resultado no puede determinarse. La operación queda bloqueada.",
  analisis_estado_acceso_denegado: "No dispone de autorización para esta operación.",
  analisis_estado_conflicto:
    "El expediente cambió o la intención entra en conflicto. Revise su estado antes de continuar.",
  analisis_estado_rechazado: "La operación fue rechazada sin confirmar el análisis.",
  analisis_estado_error: "No se pudo confirmar el análisis.",
  analisis_campos_leyenda: "Datos funcionales del análisis",
  analisis_modalidad: "Modalidad",
  analisis_modalidad_ayuda: "Modalidad procedente del catálogo vigente.",
  analisis_categoria: "Categoría",
  analisis_categoria_ayuda: "Referencia opaca de la categoría validada.",
  analisis_grupo: "Grupo o subgrupo",
  analisis_grupo_ayuda: "Grupo gobernado para la categoría seleccionada.",
  analisis_causa: "Causa",
  analisis_causa_ayuda: "Causa procedente del catálogo vigente.",
  analisis_inicio: "Fecha de inicio",
  analisis_fin: "Fecha de fin",
  analisis_periodo_ayuda:
    "Use fechas civiles; el periodo técnico no puede superar cien años.",
  analisis_jornada: "Jornada en diezmilésimas",
  analisis_jornada_ayuda:
    "Introduzca un entero entre 1 y 10.000; 10.000 equivale a jornada completa.",
  analisis_entrada_rc: "Entrada de retención de crédito",
  analisis_entrada_rc_ayuda:
    "Seleccione la referencia opaca preparada para este expediente.",
  analisis_motivo_rectificacion: "Motivo de la rectificación",
  analisis_motivo_rectificacion_ayuda:
    "La rectificación exige un motivo del catálogo gobernado.",
  analisis_registrar: "Registrar análisis",
  analisis_rectificar: "Rectificar análisis",
  analisis_cancelar: "Cancelar espera",
  analisis_errores_titulo: "Revise los errores del análisis",
  analisis_errores_descripcion:
    "Corrija los campos indicados. No se ha enviado ninguna petición.",
  analisis_error_opcion: "Seleccione una opción gobernada disponible.",
  analisis_error_fecha: "Introduzca una fecha civil válida.",
  analisis_error_periodo:
    "El fin no puede preceder al inicio ni superar cien años civiles.",
  analisis_error_jornada: "Introduzca una jornada entera entre 1 y 10.000.",
  analisis_error_motivo: "Seleccione un motivo gobernado para rectificar.",
  analisis_error_contrato: "Los datos no respetan el contrato cerrado.",
  analisis_indeterminado_titulo: "Resultado indeterminado",
  analisis_indeterminado_descripcion:
    "No repita la operación. Consulte el estado por el canal protegido o solicite asistencia.",
  analisis_recibo_sobrelinea: "Confirmación verificada",
  analisis_recibo_titulo: "Análisis confirmado",
  analisis_recibo_descripcion:
    "El recibo corresponde a la operación, el expediente y la versión enviados.",
  analisis_recibo_expediente: "Referencia del expediente",
  analisis_recibo_version: "Versión resultante",
  analisis_recibo_referencia: "Referencia del recibo",
  analisis_recibo_fecha: "Fecha de confirmación",
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
