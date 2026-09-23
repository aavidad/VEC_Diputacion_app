// Complemento del catálogo común de Dietas para la consulta de borradores propios.
export const MENSAJES_BORRADORES_ES = Object.freeze({
  borradores_propios_titulo_registrados: "Borradores registrados",
  borradores_propios_ya_registrado: "Este borrador ya se registró. Consulte su recibo o cambie los datos para iniciar otro.",
  borradores_propios_detalle_no_actualizado: "No se ha podido actualizar el detalle. Se conserva el último recibo obtenido.",
  borradores_propios_detalle_denegado: "No tiene permiso para consultar este detalle. Se conserva el último recibo obtenido.",
  borradores_propios_creado_listado_denegado: "Borrador registrado. No tiene permiso para actualizar la bandeja; conserve el recibo y no vuelva a crear el borrador.",
  borradores_propios_fechas_invalidas: "El regreso debe ser posterior a la salida. Revise las fechas y horas.",
  borradores_propios_ruta_no_declarada: "Sin itinerario declarado",
  borradores_propios_consulta_denegada: "No tiene permiso para consultar sus borradores de Dietas.",
  borradores_propios_creacion_denegada: "No tiene permiso para crear un borrador de Dietas.",
  borradores_propios_autenticacion_requerida: "Debe identificarse de nuevo para consultar o crear borradores de Dietas.",
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
