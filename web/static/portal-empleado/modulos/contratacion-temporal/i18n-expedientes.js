/** Textos castellanos de la superficie de expedientes de contratación temporal. */

export const MENSAJES_EXPEDIENTES_CONTRATACION_ES = Object.freeze({
  sobrelinea: "Gestión de contratación temporal",
  titulo: "Expedientes de contratación",
  descripcion:
    "Flujo guiado desde la petición del centro hasta la incorporación, GINPIX, seguimiento y cierre.",
  presentacion:
    "Presentación RRHH: expedientes, personas, recibos y actuaciones son sintéticos y no producen efectos reales.",
  navegacion: "Áreas de contratación temporal",
  nav_cuadro: "Cuadro de mando",
  nav_alta: "Nueva petición",
  nav_expediente: "Expediente",
  nav_documentos: "Documentos",
  nav_auditoria: "Auditoría",
  estado_inicial: "La superficie está preparada para cargar los expedientes.",
  estado_cargando: "Cargando cuadro de contratación temporal.",
  estado_listo: "Cuadro de contratación temporal actualizado.",
  estado_vacio: "No hay expedientes que coincidan con los filtros.",
  estado_error_carga: "No se pudo cargar el cuadro. Reintente o contacte con soporte.",
  estado_denegado: "No dispone de acceso a esta superficie.",
  estado_denegado_expediente: "No dispone de acceso al detalle del expediente.",
  estado_cargando_expediente: "Cargando expediente y trazabilidad.",
  estado_expediente_listo: "Expediente cargado.",
  estado_error_expediente: "No se pudo cargar el expediente. Reintente desde el cuadro.",
  estado_accion_denegada: "La actuación no está disponible para el perfil activo.",
  estado_registrando_actuacion: "Registrando la actuación. No cierre esta pantalla.",
  estado_actuacion_registrada: "Actuación registrada y expediente actualizado.",
  estado_confirmada_actualizacion_pendiente:
    "Actuación confirmada. La actualización de la vista está pendiente; no repita el efecto.",
  estado_actualizacion_pendiente:
    "Actualización pendiente. Recargue el expediente antes de realizar otra actuación.",
  estado_error_actuacion:
    "No se pudo confirmar la actuación. El expediente se conserva sin cambios visibles.",
  estado_cancelado:
    "Se canceló la espera. El resultado puede ser indeterminado; recargue antes de repetir.",
  cargando_titulo: "Comprobando acceso y datos",
  cargando_detalle: "Espere mientras se obtiene la proyección autorizada.",
  error_titulo: "La superficie no está disponible",
  vacio_titulo: "Sin resultados",
  vacio_detalle: "Cambie o quite algún filtro para ampliar la búsqueda.",
  denegado_titulo: "Acceso denegado",
  reintentar: "Reintentar",
  volver_cuadro: "Volver al cuadro",
  indicadores: "Resumen de expedientes",
  trabajo_sobrelinea: "Espacio de trabajo",
  trabajo_titulo: "Prioridades de contratación temporal",
  trabajo_descripcion:
    "Mis tareas, distribución del trabajo y accesos rápidos, calculados desde la proyección autorizada.",
  mis_tareas: "Mis tareas prioritarias",
  distribucion_fases: "Distribución por fase",
  accesos_rapidos: "Accesos rápidos",
  crear_peticion: "Registrar nueva petición",
  continuar_tramitacion: "Continuar tramitación prioritaria",
  filtros: "Filtros del cuadro",
  filtro_texto: "Buscar expediente, centro o categoría",
  filtro_texto_placeholder: "Número, centro, categoría o unidad",
  filtro_estado: "Estado",
  filtro_fase: "Fase actual",
  filtro_todos: "Todos",
  aplicar_filtros: "Aplicar filtros",
  limpiar_filtros: "Limpiar",
  resultados: "{total} expedientes en el resultado",
  tabla_expedientes: "Expedientes de contratación temporal",
  columna_numero: "N.º expediente",
  columna_centro: "Centro",
  columna_categoria: "Categoría",
  columna_modalidad: "Modalidad",
  columna_estado: "Estado",
  columna_fase: "Fase actual",
  columna_plazo: "Plazo",
  columna_acciones: "Acciones",
  abrir: "Abrir expediente",
  flujo_expediente: "Progreso del expediente",
  expediente_etiqueta: "Expediente",
  metadatos_tecnicos: "Metadatos técnicos del expediente",
  referencia_interna: "Referencia interna",
  flujo_definicion: "Definición de flujo",
  flujo_version: "Versión del flujo",
  flujo_huella: "Huella de la definición",
  fases_expediente: "Fases del procedimiento",
  tareas_expediente: "Pantallas y tareas del procedimiento",
  tarea_actual: "Tarea actual",
  responsable: "Responsable",
  unidad: "Unidad",
  entrada: "Entrada",
  salida: "Salida",
  tiempo: "Tiempo empleado",
  recibo: "Recibo",
  decision: "Decisión",
  panel_sin_datos: "No hay datos disponibles en este panel.",
  tabla_panel: "{titulo}",
  formulario_tarea: "Datos de la tarea {tarea}",
  tarea_solo_lectura:
    "Vista histórica o de consulta. Los datos no pueden modificarse con el perfil y estado actuales.",
  campos_obligatorios: "Los campos marcados con * son obligatorios.",
  obligatorio: "Obligatorio",
  posicion_tarea: "{actual} de {total}",
  confirmar_titulo: "Confirmar actuación",
  confirmar_aviso:
    "La composición real volverá a validar identidad, competencia, versión y autorización en servidor.",
  cancelar_espera: "Cancelar espera",
  accion_no_disponible: "Actuación no disponible: {motivo}",
  recibo_titulo: "Recibo de actuación",
  recibo_descripcion:
    "Conserve estas referencias para el seguimiento. En presentación no tienen validez administrativa.",
  recibo_referencia: "Referencia de recibo",
  recibo_expediente: "Expediente",
  recibo_version: "Nueva versión",
  recibo_actuacion: "Actuación",
  recibo_estado: "Estado resultante",
  recibo_fecha: "Fecha de registro",
  expediente_sin_seleccionar: "Seleccione un expediente desde el cuadro.",
  documentos_titulo: "Documentos del expediente",
  documentos_descripcion:
    "Versiones, firmas, estados y descarga autorizada de cada pieza documental.",
  documentos_tabla: "Índice documental del expediente",
  documento: "Documento",
  tipo: "Formato",
  version: "Versión",
  firma: "Firma",
  fecha: "Fecha",
  descarga: "Descarga",
  descargar: "Descargar",
  descarga_conector_pendiente: "Descarga pendiente de conectar",
  no_disponible: "No disponible",
  auditoria_titulo: "Cronología y auditoría",
  auditoria_descripcion:
    "Historia de solo adición con actor, unidad, instante, resultado y documento asociado.",
  auditoria_tabla: "Actuaciones del expediente",
  actuacion: "Actuación",
  actor: "Actor",
  unidad: "Unidad",
  observaciones: "Observaciones",
  documento_asociado: "Documento asociado",
  nueva_peticion_titulo: "Nueva petición de personal",
  nueva_peticion_descripcion:
    "El alta reutiliza el contrato definitivo O2-09B y recibe catálogos y ejecutor por inyección.",
  fase_pendiente: "Pendiente",
  fase_en_curso: "En tramitación",
  fase_espera: "Pendiente de otro departamento",
  fase_completado: "Completado",
  fase_incidencia: "Con incidencia",
  fase_cancelado: "Cancelado",
});

export function crearTraductorExpedientesContratacion(sobrescrituras = {}) {
  if (sobrescrituras === null || typeof sobrescrituras !== "object"
    || Array.isArray(sobrescrituras)) {
    throw new TypeError("mensajes de expedientes no válidos");
  }
  const mensajes = { ...MENSAJES_EXPEDIENTES_CONTRATACION_ES, ...sobrescrituras };
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
