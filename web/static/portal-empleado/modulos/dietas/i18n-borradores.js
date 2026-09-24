// Complemento del catálogo común de Dietas para la consulta de borradores propios.
export const MENSAJES_BORRADORES_ES = Object.freeze({
  borradores_propios_titulo_registrados: "Borradores registrados",
  borradores_propios_estado_sin_seleccion: "Ningún borrador seleccionado.",
  borradores_propios_ya_registrado: "Este borrador ya se registró. Consulte su recibo o cambie los datos para iniciar otro.",
  borradores_propios_detalle_no_actualizado: "No se ha podido actualizar el detalle. Se conserva el último recibo obtenido.",
  borradores_propios_detalle_denegado: "No tiene permiso para consultar este detalle. Se conserva el último recibo obtenido.",
  borradores_propios_creado_listado_denegado: "Borrador registrado. No tiene permiso para actualizar la bandeja; conserve el recibo y no vuelva a crear el borrador.",
  borradores_propios_fechas_invalidas: "El regreso debe ser posterior a la salida. Revise las fechas y horas.",
  borradores_propios_ruta_no_declarada: "Sin itinerario declarado",
  borradores_propios_consulta_denegada: "No tiene permiso para consultar sus borradores de Dietas.",
  borradores_propios_creacion_denegada: "No tiene permiso para crear un borrador de Dietas.",
  borradores_propios_autenticacion_requerida: "Debe identificarse de nuevo para consultar o crear borradores de Dietas.",
  borradores_propios_consultar_registrados: "Consultar borradores registrados",
  borradores_propios_consultando_registrados: "Consultando sus borradores registrados…",
  borradores_propios_consulta_registrados: "Borradores consultados. La lista no confirma altas anteriores.",
  borradores_propios_consulta_registrados_vacia: "Sin borradores en esta consulta. La lista no confirma altas anteriores.",
  borradores_propios_consulta_ayuda: "Si una creación quedó sin confirmar, consulte la lista. Elija un borrador para ver su recibo. Que aparezca en la lista no demuestra que corresponda a aquel intento; no cree otro solo para comprobarlo.",
  borradores_propios_pais: "País",
  borradores_propios_pais_espana: "España",
  borradores_propios_preparacion_ayuda: "Puede corregir los campos antes de registrar el borrador. El país se muestra por el catálogo provincial disponible; el servicio todavía no registra país por separado. Revisar no guarda ni envía datos.",
  borradores_propios_revisar: "Revisar preparación",
  borradores_propios_preparacion_titulo: "Preparación local sin registrar",
  borradores_propios_preparacion_editable: "Puede seguir editando los campos antes de crear el borrador.",
  borradores_propios_registrado_limite: "El borrador registrado solo puede consultarse. La edición y el envío requieren un servicio autorizado todavía no conectado.",
  borradores_propios_edicion_pendiente: "Editar borrador registrado",
  borradores_propios_envio_pendiente: "Enviar a revisión",
  borradores_propios_pais_no_registrado: "No consta como campo separado en el registro actual",
});

export function crearTraductorBorradoresDietas(traducirBase) {
  if (typeof traducirBase !== "function")
    throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_BORRADORES_ES, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_BORRADORES_ES[clave].replace(
      /\{([a-z_]+)\}/gu,
      (_coincidencia, variable) => String(variables[variable] ?? ""),
    );
  };
}
