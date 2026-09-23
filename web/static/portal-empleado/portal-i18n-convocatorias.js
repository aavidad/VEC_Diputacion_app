/** Catálogo castellano de la consulta S1. El montaje puede inyectar un traductor. */
export const MENSAJES_CONVOCATORIAS_S1_ES = Object.freeze({
  sobrelinea: "Selección · Bolsa",
  titulo: "Convocatorias, bases y calendario",
  descripcion: "Consulta de versiones, requisitos y fechas del ámbito autorizado.",
  ayuda_aria: "Ayuda sobre convocatorias y bases",
  ayuda_detalle: "La información procede de la consulta autorizada. Una versión publicada no se modifica desde esta pantalla. La firma y la publicación requieren sus circuitos propios.",
  estado_no_configurado_titulo: "Consulta no configurada",
  estado_no_configurado_detalle: "El servicio de convocatorias todavía no está conectado a esta pantalla.",
  estado_cargando_titulo: "Cargando convocatorias",
  estado_cargando_detalle: "Se está consultando el ámbito autorizado.",
  estado_cargando_detalle_titulo: "Cargando detalle",
  estado_cargando_detalle_detalle: "Se están consultando las bases y versiones de la convocatoria seleccionada.",
  estado_denegado_titulo: "Acceso denegado",
  estado_denegado_detalle: "La sesión no dispone de permiso para consultar esta información.",
  estado_error_titulo: "Consulta no disponible",
  estado_error_detalle: "No se pudo obtener la lista de convocatorias. Vuelva a intentarlo más tarde.",
  estado_error_detalle_titulo: "Detalle no disponible",
  estado_error_detalle_detalle: "No se pudieron obtener las bases y versiones de esta convocatoria.",
  estado_vacio_titulo: "Sin convocatorias",
  estado_vacio_detalle: "La fuente autorizada no ha devuelto convocatorias para este ámbito.",
  estado_sin_seleccion_titulo: "Seleccione una convocatoria",
  estado_sin_seleccion_detalle: "El detalle aparecerá al seleccionar una convocatoria de la lista.",
  listado: "Convocatorias",
  cantidad: "{numero} convocatorias",
  solo_lectura: "Solo lectura",
  version: "Versión",
  cierre: "Cierre",
  sin_fecha: "Fecha no disponible",
  sin_dato: "No consta",
  version_actual: "Versión actual",
  identificador_publico: "Identificador público",
  bases: "Bases",
  versiones: "Versiones de la convocatoria",
  versiones_vacias: "No se han devuelto versiones.",
  requisitos: "Requisitos de acceso",
  requisitos_vacios: "No se han devuelto requisitos estructurados para esta versión.",
  obligatorio: "Obligatorio",
  no_obligatorio: "No obligatorio",
  hito_exigibilidad: "Exigible en",
  hitos: "Hitos y plazos",
  hitos_vacios: "No se han devuelto hitos de calendario.",
  documentos: "Documentos de las bases",
  documentos_vacios: "No se han devuelto documentos de las bases.",
  acciones: "Tramitación",
  editar_bases: "Editar bases",
  enviar_firma: "Enviar a firma",
  publicar: "Publicar convocatoria",
  accion_no_conectada: "Acción no conectada al servidor",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_CONVOCATORIAS_S1_ES));

export function crearTraductorConvocatoriasS1(catalogo = MENSAJES_CONVOCATORIAS_S1_ES) {
  if (!catalogo || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("Catálogo de convocatorias S1 incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`Clave de convocatorias S1 desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirConvocatoriasS1 = crearTraductorConvocatoriasS1();
