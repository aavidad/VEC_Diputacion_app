/** Textos de otros medios de transporte y otros gastos con justificante. */
export const MENSAJES_OTROS_GASTOS_ES = Object.freeze({
  otros_gastos_ayuda: "Añada cada billete, taxi, aparcamiento o peaje de la comisión con su fecha e importe. El justificante se queda con usted: indique una referencia para localizarlo y su huella SHA-256, que puede calcular eligiendo el fichero; el fichero no se envía.",
  otros_gastos_grupo_medio: "Otros medios de transporte",
  otros_gastos_grupo_gasto: "Otros gastos",
  otros_gastos_tipo: "Tipo",
  otros_gastos_elegir_tipo: "Elija un tipo",
  otros_gastos_fecha: "Fecha",
  otros_gastos_descripcion: "Descripción",
  otros_gastos_justificante_ref: "Referencia del justificante",
  otros_gastos_justificante_sha256: "Huella SHA-256",
  otros_gastos_calcular_huella: "Calcular huella desde el fichero",
  otros_gastos_huella_error: "No se ha podido calcular la huella del fichero.",
  otros_gastos_huella_calculada: "Huella calculada a partir del fichero.",
  otros_gastos_huella_grande: "El fichero supera 25 MB y no se puede calcular su huella aquí.",
  otros_gastos_quitar_linea: "Quitar línea {numero}",
  otros_gastos_linea_anterior: "Faltan el tipo, la fecha o el justificante de este gasto; complételos antes de guardar.",
  otros_gastos_error: "Revise cada gasto: tipo, fecha dentro de la comisión, descripción, importe, referencia del justificante (letras, números, «:», «_» o «-») y huella SHA-256.",
  otros_gastos_sin_catalogo: "Los tipos de gasto no están disponibles en este momento.",
  otros_gastos_linea: "{tipo} · {fecha} · {descripcion}",
  otros_gastos_tipo_tren: "Tren",
  otros_gastos_tipo_autobus: "Autobús",
  otros_gastos_tipo_metro_tranvia: "Metro o tranvía",
  otros_gastos_tipo_taxi: "Taxi",
  otros_gastos_tipo_avion: "Avión",
  otros_gastos_tipo_barco: "Barco",
  otros_gastos_tipo_vehiculo_alquiler: "Vehículo de alquiler",
  otros_gastos_tipo_aparcamiento: "Aparcamiento",
  otros_gastos_tipo_peaje: "Peaje",
  otros_gastos_tipo_consigna: "Consigna",
  otros_gastos_tipo_desconocido: "Otro tipo",
});

export function crearTraductorOtrosGastosDietas(traducirBase) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_OTROS_GASTOS_ES, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_OTROS_GASTOS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

/** Rótulo de un tipo del catálogo; nunca muestra el código interno. */
export function rotuloTipoOtroGasto(traducir, codigo) {
  const clave = `otros_gastos_tipo_${codigo}`;
  return typeof codigo === "string" && Object.hasOwn(MENSAJES_OTROS_GASTOS_ES, clave)
    ? traducir(clave) : traducir("otros_gastos_tipo_desconocido");
}
