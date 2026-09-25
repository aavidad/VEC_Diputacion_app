/** Textos del cambio de situación con reglas del catálogo. La ayuda «?» vive en el catálogo común. */
export const MENSAJES_REGLAS_SITUACION_ES = Object.freeze({
  reposicion: "Fin de la relación",
  fin_relacion: "Fecha de fin",
  modalidad: "Modalidad del nombramiento",
  modalidad_general: "Otra modalidad",
  modalidad_acumulacion_tareas: "Acumulación de tareas",
  modalidad_meses: "{modalidad} ({meses} meses)",
  propuesta: "Fecha propuesta: {meses} meses de fecha a fecha ({procedencia}).",
  procedencia_reglamento: "Reglamento, {articulo}",
  procedencia_ejemplo: "criterio de trabajo",
  regla_ejemplo: "Incluye una parte de ejemplo pendiente de RRHH.",
  propuesta_pasada: "La fecha ya ha pasado: elija «Disponible».",
  propuesta_no_disponible: "Sin propuesta: indique la fecha a mano.",
  causa_baja: "Causa de baja",
  causa_elegir: "Seleccione una causa",
  causa_otro: "Otro motivo",
  causa_detalle: "Detalle",
  causa_incompleta: "Elija una causa; con «Otro motivo» escriba el motivo (hasta 1000 caracteres).",
  motivo_causa: "{causa} ({procedencia})",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_REGLAS_SITUACION_ES));

export function traducirReglasSituacion(clave, variables = {}) {
  if (!CLAVES.includes(clave)) throw new Error(`clave i18n de reglas de situación desconocida: ${clave}`);
  return MENSAJES_REGLAS_SITUACION_ES[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
}

/** Etiqueta de una modalidad; una modalidad nueva del catálogo se muestra por su código. */
export function etiquetaModalidadReposicion(codigo) {
  const clave = `modalidad_${codigo}`;
  return CLAVES.includes(clave) ? MENSAJES_REGLAS_SITUACION_ES[clave] : String(codigo).replaceAll("_", " ");
}
