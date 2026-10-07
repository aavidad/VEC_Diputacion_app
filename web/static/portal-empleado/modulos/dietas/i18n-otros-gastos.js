/** Textos de otros medios de transporte y otros gastos con justificante. */
export let MENSAJES_OTROS_GASTOS;
export function publicarMensajesOtrosGastos(mensajes) { MENSAJES_OTROS_GASTOS = mensajes; }

export function crearTraductorOtrosGastosDietas(traducirBase) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  if (!MENSAJES_OTROS_GASTOS) throw new Error("textos de Dietas sin preparar");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_OTROS_GASTOS, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_OTROS_GASTOS[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

/** Rótulo de un tipo del catálogo; nunca muestra el código interno. */
export function rotuloTipoOtroGasto(traducir, codigo) {
  if (!MENSAJES_OTROS_GASTOS) throw new Error("textos de Dietas sin preparar");
  const clave = `otros_gastos_tipo_${codigo}`;
  return typeof codigo === "string" && Object.hasOwn(MENSAJES_OTROS_GASTOS, clave)
    ? traducir(clave) : traducir("otros_gastos_tipo_desconocido");
}
