/** Textos castellanos del módulo; las vistas solo consumen claves. */
import { MENSAJES_LLAMAMIENTO_ES } from "./i18n-llamamiento.js";

export const MENSAJES_CONTRATACION_TEMPORAL_ES = Object.freeze({
  ...MENSAJES_LLAMAMIENTO_ES,
  "contratacion_temporal.flujo.rrhh": "Gestión de expedientes de personal temporal",
  "contratacion_temporal.fase.solicitud": "Solicitud",
  "contratacion_temporal.fase.analisis_rrhh": "Análisis RRHH",
  "contratacion_temporal.fase.gestion_bolsa": "Gestión de bolsa",
  "contratacion_temporal.fase.fiscalizacion": "Fiscalización",
  "contratacion_temporal.fase.obtencion_candidato": "Obtención del candidato",
  "contratacion_temporal.fase.nombramiento": "Nombramiento",
  "contratacion_temporal.fase.incorporacion": "Incorporación",
  "contratacion_temporal.fase.seguimiento": "Seguimiento",
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
  cobertura_sobrelinea: "Contratación temporal · Vía de cobertura",
  cobertura_titulo: "Decidir la vía de cobertura",
  cobertura_descripcion:
    "Revise la propuesta calculada con el análisis confirmado antes de aplicar la decisión.",
  cobertura_alcance_etiqueta: "Alcance de la decisión",
  cobertura_alcance:
    "La propuesta, la identidad, la autorización y la persistencia proceden del servidor.",
  cobertura_estado_cargando: "Calculando la propuesta de cobertura.",
  cobertura_estado_lista: "La propuesta de cobertura está preparada para su confirmación.",
  cobertura_estado_sin_via: "No existe una vía de cobertura viable.",
  cobertura_estado_error_propuesta: "No se pudo obtener una propuesta de cobertura verificable.",
  cobertura_estado_enviando: "Aplicando una única decisión de cobertura. Espere la confirmación.",
  cobertura_estado_confirmada: "La decisión de cobertura ha quedado confirmada.",
  cobertura_estado_asignacion_no_disponible:
    "La cobertura está confirmada, pero la asignación no está disponible. Conserve el recibo y solicite revisión.",
  cobertura_estado_indeterminado:
    "El resultado no puede determinarse todavía. No repita la decisión.",
  cobertura_estado_rechazada: "La decisión no se confirmó. Revise el estado del expediente.",
  cobertura_estado_consultando: "Consultando el resultado de la decisión original.",
  cobertura_estado_no_observable: "El resultado original todavía no es observable.",
  cobertura_estado_consulta_error: "No se pudo consultar el resultado. No repita la decisión.",
  cobertura_estado_error: "No se pudo preparar la decisión de cobertura.",
  cobertura_cargando_descripcion:
    "Se consulta el análisis durable y las fuentes gobernadas de cobertura.",
  cobertura_error_descripcion:
    "La operación permanece cerrada. Revise el estado antes de volver a intentarlo.",
  cobertura_propuesta_titulo: "Propuesta calculada",
  cobertura_via_recomendada: "Vía recomendada",
  cobertura_via_bolsa_vigente: "Bolsa vigente",
  cobertura_via_oferta_sae: "Oferta SAE",
  cobertura_via_nueva_convocatoria_bolsa: "Nueva convocatoria de Bolsa",
  cobertura_via_elegida: "Vía elegida por RRHH",
  cobertura_via_ayuda:
    "Seleccione expresamente una vía viable.",
  cobertura_motivo_alternativa: "Motivo gobernado para una vía alternativa",
  "contratacion_temporal.cobertura.motivo.eleccion_procedimiento_rrhh":
    "Elección del procedimiento de cobertura por RRHH",
  cobertura_estado_via_obligatoria: "Seleccione una vía viable antes de confirmar.",
  cobertura_estado_motivo_obligatorio:
    "Seleccione un motivo gobernado vinculado a la vía alternativa.",
  cobertura_evaluacion: "Evaluación",
  cobertura_evaluacion_viable: "Viable",
  cobertura_evaluacion_incompleta: "Incompleta",
  cobertura_evaluacion_conflictiva: "Con conflictos",
  cobertura_evaluacion_no_viable: "No viable",
  cobertura_confirmacion_ayuda:
    "Confirme solo después de comprobar que la vía recomendada corresponde a este expediente.",
  cobertura_confirmar: "Confirmar vía de cobertura",
  cobertura_confirmacion_advertencia:
    "Esta acción registrará la decisión y avanzará el expediente a asignación de unidad.",
  cobertura_indeterminado_titulo: "Resultado pendiente de comprobación",
  cobertura_indeterminado_descripcion:
    "Use la consulta protegida con la misma clave; no vuelva a enviar la decisión.",
  cobertura_consultar_resultado: "Consultar resultado original",
  cobertura_sin_via_titulo: "Sin vía viable",
  cobertura_sin_via_descripcion:
    "La propuesta no permite continuar. Revise las ausencias y conflictos del análisis.",
  cobertura_recibo_sobrelinea: "Decisión confirmada",
  cobertura_recibo_titulo: "Vía de cobertura aplicada",
  cobertura_recibo_descripcion:
    "El recibo acredita la decisión, la nueva versión del expediente y su persistencia.",
  cobertura_recibo_expediente: "Referencia del expediente",
  cobertura_recibo_version: "Versión resultante",
  cobertura_recibo_referencia: "Referencia del recibo",
  cobertura_recibo_decision: "Referencia de la decisión",
  cobertura_recibo_fecha: "Fecha de confirmación",
  asignacion_sobrelinea: "Contratación temporal · Asignación de unidad",
  asignacion_titulo: "Asignar expediente a la unidad responsable",
  asignacion_descripcion:
    "Revise el destino sintético y confirme la asignación del expediente.",
  asignacion_alcance_etiqueta: "Alcance de la asignación",
  asignacion_alcance:
    "La identidad, la organización, la autorización y la persistencia proceden del servidor.",
  asignacion_estado_lista: "La asignación está preparada para su confirmación.",
  asignacion_estado_enviando: "Registrando una única asignación. Espere la confirmación.",
  asignacion_estado_recuperando: "Recuperando la asignación original con la misma clave.",
  asignacion_estado_confirmada: "La asignación ha quedado confirmada.",
  asignacion_estado_informe_no_disponible:
    "La asignación está confirmada, pero el informe jurídico no pudo abrirse. Conserve el recibo.",
  asignacion_estado_indeterminado:
    "El resultado no puede determinarse todavía. Recupere la operación original; no cree otra.",
  asignacion_estado_rechazada:
    "La asignación no se confirmó. Revise el estado antes de volver a intentarlo.",
  asignacion_estado_error: "No se pudo preparar una solicitud de asignación válida.",
  asignacion_destino_leyenda: "Destino de la asignación",
  asignacion_unidad: "Unidad responsable",
  asignacion_unidad_ayuda: "Unidad sintética cerrada para este recorrido de desarrollo.",
  asignacion_responsable: "Responsable referenciado",
  asignacion_responsable_ayuda:
    "Referencia opaca sintética; no contiene nombre, correo ni documento identificativo.",
  asignacion_confirmacion:
    "He comprobado el expediente, la unidad y la referencia responsable.",
  asignacion_resumen: "Expediente {expediente}, versión actual {version}.",
  asignacion_confirmar: "Confirmar asignación",
  asignacion_confirmacion_advertencia:
    "Esta acción registrará la unidad y la persona responsable referenciada en una nueva versión.",
  asignacion_indeterminada_titulo: "Resultado pendiente de recuperación",
  asignacion_indeterminada_descripcion:
    "Tras una interrupción, reenvíe exactamente la misma operación para recuperar su recibo.",
  asignacion_recuperar: "Recuperar asignación original",
  asignacion_recibo_sobrelinea: "Asignación confirmada",
  asignacion_recibo_titulo: "Expediente asignado",
  asignacion_recibo_descripcion:
    "El recibo acredita la asignación y la nueva versión persistida del expediente.",
  asignacion_recibo_expediente: "Referencia del expediente",
  asignacion_recibo_version: "Versión resultante",
  asignacion_recibo_referencia: "Referencia del recibo",
  asignacion_recibo_fecha: "Fecha de confirmación",
  informe_sobrelinea: "Contratación temporal · Informe jurídico",
  informe_titulo: "Preparar informe jurídico",
  informe_descripcion:
    "Genere el documento de desarrollo a partir del expediente asignado y de sus datos gobernados.",
  informe_alcance_etiqueta: "Alcance del informe",
  informe_alcance:
    "El contenido, la identidad, la autorización y la persistencia proceden del servidor.",
  informe_estado_listo: "El informe jurídico está preparado para su confirmación.",
  informe_estado_enviando: "Preparando un único informe jurídico. Espere la confirmación.",
  informe_estado_recuperando: "Recuperando el informe original con la misma clave.",
  informe_estado_confirmado: "El informe jurídico y su recibo han quedado confirmados.",
  informe_estado_indeterminado:
    "El resultado no puede determinarse todavía. Recupere la operación original; no cree otra.",
  informe_estado_rechazado:
    "El informe no se confirmó. Revise el estado del expediente antes de volver a intentarlo.",
  informe_estado_error: "No se pudo preparar una solicitud de informe jurídico válida.",
  informe_estado_historial_no_disponible:
    "El informe está confirmado, pero el historial no está disponible temporalmente.",
  informe_confirmacion_leyenda: "Confirmación del informe jurídico",
  informe_resumen: "Expediente {expediente}, versión asignada {version}.",
  informe_confirmacion:
    "He comprobado el expediente y entiendo que se generará un documento de desarrollo sin firma.",
  informe_confirmar: "Confirmar y preparar informe",
  informe_confirmacion_advertencia:
    "Esta acción generará y registrará una nueva versión del informe jurídico de desarrollo.",
  informe_indeterminado_titulo: "Resultado pendiente de recuperación",
  informe_indeterminado_descripcion:
    "Tras una interrupción, reenvíe exactamente la misma operación para recuperar su recibo.",
  informe_recuperar: "Recuperar informe original",
  informe_recibo_sobrelinea: "Informe registrado",
  informe_recibo_titulo: "Recibo del informe jurídico",
  informe_recibo_descripcion:
    "El recibo acredita el informe, el documento y la nueva versión persistida del expediente.",
  informe_recibo_expediente: "Referencia del expediente",
  informe_recibo_version: "Versión resultante",
  informe_recibo_informe: "Referencia del informe",
  informe_recibo_documento: "Referencia del documento",
  informe_recibo_referencia: "Referencia del recibo",
  informe_recibo_auditoria: "Referencia de auditoría",
  informe_recibo_evento: "Referencia del evento",
  informe_recibo_fecha: "Fecha de confirmación",
  informe_documento_rotulo: "Documento de desarrollo · Pendiente de revisión y firma",
  informe_documento_advertencia:
    "Este contenido no está firmado ni tiene validez jurídica.",
  informe_documento_version: "Versión del documento",
  informe_documento_formato: "Formato",
  informe_documento_huella: "Huella SHA-256",
  informe_documento_contenido: "Contenido del documento",
  informe_historial_titulo: "Historial real del expediente",
  informe_historial_descripcion:
    "Actuaciones persistidas recuperadas después de registrar el informe.",
  informe_historial_cargando: "Recuperando el historial actualizado.",
  informe_historial_no_disponible:
    "El historial no se pudo recuperar. El informe y su recibo permanecen confirmados y visibles.",
  informe_historial_vacio: "El expediente no contiene hitos visibles.",
  informe_historial_tabla: "Historial persistido del expediente",
  informe_historial_secuencia: "Secuencia",
  informe_historial_version: "Versión",
  informe_historial_accion: "Actuación",
  informe_historial_fase: "Fase de origen y destino",
  informe_historial_estado: "Estado de origen y destino",
  informe_historial_fecha: "Fecha",
  fiscalizacion_sobrelinea: "Contratación temporal · Fiscalización",
  fiscalizacion_titulo: "Registrar resultado de fiscalización",
  fiscalizacion_descripcion:
    "Revise el expediente y registre el resultado comunicado por Intervención.",
  fiscalizacion_alcance_etiqueta: "Alcance del registro",
  fiscalizacion_alcance:
    "El actor, la unidad, la autorización, la transición y la persistencia proceden del servidor.",
  fiscalizacion_estado_lista: "El resultado de fiscalización está preparado para su registro.",
  fiscalizacion_estado_enviando: "Registrando un único resultado. Espere la confirmación.",
  fiscalizacion_estado_recuperando:
    "Recuperando el resultado original con la misma clave de idempotencia.",
  fiscalizacion_estado_confirmada: "El resultado y su recibo han quedado confirmados.",
  fiscalizacion_estado_indeterminado:
    "El resultado no puede determinarse todavía. Recupere la operación original; no cree otra.",
  fiscalizacion_estado_rechazada:
    "El resultado no se confirmó. Revise el expediente antes de volver a intentarlo.",
  fiscalizacion_estado_validacion:
    "Seleccione un resultado y aporte observaciones cuando sean obligatorias.",
  fiscalizacion_contexto_expediente: "Referencia del expediente",
  fiscalizacion_contexto_version: "Versión actual",
  fiscalizacion_contexto_fase: "Fase actual",
  fiscalizacion_contexto_informe: "Informe jurídico",
  fiscalizacion_fase_informe_juridico: "Informe jurídico registrado",
  fiscalizacion_informe_registrado: "Registrado en la versión {version}",
  fiscalizacion_resultado_leyenda: "Resultado comunicado por Intervención",
  fiscalizacion_resultado_ayuda:
    "El resultado determina si el expediente continúa o vuelve para subsanación.",
  fiscalizacion_resultado_favorable: "Favorable",
  fiscalizacion_resultado_favorable_con_observaciones: "Favorable con observaciones",
  fiscalizacion_resultado_desfavorable: "Desfavorable",
  fiscalizacion_observaciones: "Observaciones",
  fiscalizacion_observaciones_ayuda:
    "Obligatorias para favorable con observaciones y desfavorable.",
  fiscalizacion_confirmar: "Registrar resultado",
  fiscalizacion_confirmacion_continuar:
    "Esta acción registrará la fiscalización y permitirá continuar el expediente.",
  fiscalizacion_confirmacion_desfavorable:
    "Esta acción registrará el resultado desfavorable y devolverá el expediente para subsanación sin sustituir su histórico.",
  fiscalizacion_indeterminada_titulo: "Resultado pendiente de recuperación",
  fiscalizacion_indeterminada_descripcion:
    "Tras una interrupción, reenvíe exactamente la misma operación para recuperar su recibo.",
  fiscalizacion_recuperar: "Recuperar resultado original",
  fiscalizacion_recibo_sobrelinea: "Fiscalización registrada",
  fiscalizacion_recibo_titulo: "Recibo del resultado de fiscalización",
  fiscalizacion_recibo_descripcion:
    "El recibo acredita el resultado, el actor y el estado resultante del expediente.",
  fiscalizacion_recibo_expediente: "Referencia del expediente",
  fiscalizacion_recibo_resultado: "Resultado",
  fiscalizacion_recibo_fase: "Fase resultante",
  fiscalizacion_recibo_estado: "Estado resultante",
  fiscalizacion_recibo_version: "Versión resultante",
  fiscalizacion_recibo_referencia: "Referencia del recibo",
  fiscalizacion_recibo_auditoria: "Referencia de auditoría",
  fiscalizacion_recibo_evento: "Referencia del evento",
  fiscalizacion_recibo_actor: "Actor registrado",
  fiscalizacion_recibo_unidad_retorno: "Unidad de retorno",
  fiscalizacion_recibo_responsable_retorno: "Responsable del retorno",
  fiscalizacion_recibo_fecha: "Fecha de registro",
  anotacion_titulo: "Anotación administrativa",
  anotacion_limite: "Primera anotación administrativa sintética: conserva la fase y el estado. No inicia seguimiento, cese, firma ni correo.",
  anotacion_observaciones: "Observación administrativa", anotacion_enviar: "Registrar anotación administrativa",
  anotacion_enviando: "Registrando; espere la respuesta.", anotacion_pendiente: "La observación es obligatoria y admite hasta 2.000 caracteres.",
  anotacion_valida: "Escriba una observación de hasta 2.000 caracteres.", anotacion_cancelada: "No se ha iniciado ningún registro.",
  anotacion_confirmada: "Recibo verificado de la anotación administrativa.", anotacion_consultando: "Resultado incierto: consultando el recibo protegido.",
  anotacion_incierta: "No se pudo verificar el recibo. La misma intención queda conservada para recuperarla o reintentarla.",
  anotacion_conflicto: "La anotación entra en conflicto con el estado actual. No se ha reintentado automáticamente.",
  anotacion_rechazada: "La solicitud ha sido rechazada. Revise la observación antes de continuar.", anotacion_reintentar: "Reintentar la misma intención",
  anotacion_recibo: "Recibo original", anotacion_expediente: "Expediente", anotacion_version_anterior: "Versión anterior",
  anotacion_version_resultante: "Versión resultante", anotacion_fase: "Fase conservada", anotacion_estado: "Estado conservado",
  anotacion_seguimiento: "Seguimiento original", anotacion_recibo_ref: "Referencia del recibo", anotacion_registrada_en: "Fecha original de registro",
  anotacion_clave_original: "Clave original para recuperar esta intención:", anotacion_recuperar_etiqueta: "Recuperar por clave original", anotacion_recuperar_boton: "Recuperar recibo",
  anotacion_detalle_actualizado: "Anotación administrativa registrada. Recibo original: {recibo}. El detalle se ha actualizado a la versión {version}.",
  cierre_titulo: "Cierre administrativo sin cese",
  cierre_limite: "Ejercicio sintético: este cierre no registra cese ni eficacia jurídica, y no envía datos a Personal, correo o GINPIX.",
  cierre_preparar: "Preparar cierre", cierre_continuar: "Cerrar administrativamente", cierre_recibo: "Recibo original",
  cierre_recibo_version_seguimiento: "Versión del seguimiento", cierre_recibo_verificado: "Recibo verificado.",
  cierre_archivo: "Archivo de recuperación (máximo 8 KiB)", cierre_guardar: "Guardar datos de recuperación", cierre_recuperar: "Recuperar o completar la misma solicitud",
  cierre_contexto_invalido: "El cierre administrativo no está disponible porque falta o no es válida la preparación autorizada del servidor.",
  cierre_recuperacion_titulo: "Recuperar una solicitud de cierre", cierre_recuperacion_ayuda: "Cargue únicamente los datos de recuperación originales. Cargar el archivo no envía ninguna solicitud.",
  cierre_datos_validados: "Datos de recuperación validados. Revise los datos y confirme su recuperación.", cierre_archivo_invalido: "El archivo de recuperación no es válido.",
  cierre_archivo_ajeno: "El archivo no corresponde a este expediente o seguimiento.", cierre_archivo_validando: "Validando el archivo de recuperación.",
  cierre_motivo_autorizado: "Motivo autorizado", cierre_motivo_cierre_administrativo_ejercicio: "Cierre administrativo del ejercicio", cierre_motivo_publicado: "Motivo publicado por el servidor",
  cierre_dato_expediente: "Expediente", cierre_dato_seguimiento: "Seguimiento", cierre_dato_version_esperada: "Versión esperada",
  cierre_dato_clave_recuperacion: "Clave de recuperación", cierre_dato_transicion: "Transición", cierre_dato_motivo: "Motivo",
  cierre_preparada_ayuda: "La intención está preparada. Guarde los datos de recuperación antes de confirmar el cierre.",
  cierre_motivo_invalido: "Seleccione un motivo autorizado.", cierre_cancelada: "No se ha iniciado ningún cierre. Puede continuar con la misma intención preparada.",
  cierre_enviando: "Registrando; espere la respuesta.", cierre_confirmar_titulo: "Cierre administrativo sin cese",
  cierre_confirmar_replay: "Esta acción puede completar una intención pendiente sin cese ni eficacia jurídica.", cierre_confirmar_nuevo: "Esta acción registra el cierre preparado, sin cese ni eficacia jurídica.",
  cierre_error_401: "Debe identificarse para cerrar.", cierre_error_403: "No dispone de permiso para cerrar.", cierre_error_409: "El cierre entra en conflicto; no se ha repetido.",
  cierre_incierto: "Resultado incierto: guarde los datos de recuperación; no se ha repetido automáticamente.",
  cierre_estado_actual: "Estado actual del seguimiento: {estado}.", cierre_estado_vigente: "Vigente", cierre_estado_cerrado: "Cerrado", cierre_estado_desconocido: "No disponible para esta pantalla",
  cierre_estado_cerrado_confirmado: "El cierre administrativo ya consta registrado.", cierre_actualizando_estado: "Recibo verificado. Actualizando el estado del seguimiento.", cierre_actualizacion_pendiente: "Recibo verificado. La actualización del estado sigue pendiente; conserve los datos de recuperación.",
  cierre_estado_con_preparacion: "Seleccione un motivo publicado por el servidor.", cierre_estado_sin_preparacion: "No hay nueva preparación autorizada; puede cargar una solicitud original.",
  cierre_lectura_error: "No se pudo recuperar la preparación del cierre. El cierre no está habilitado; puede reintentar la lectura.",
  cierre_reintentar_lectura: "Reintentar la lectura de cierre", cierre_detalle_obsoleto: "La anotación se registró en la versión {version}. El detalle mostrado (v{version_anterior}) está obsoleto y el cierre permanece deshabilitado hasta actualizarlo.",
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
