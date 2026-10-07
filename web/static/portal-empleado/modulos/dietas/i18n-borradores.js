/** Complemento del catálogo común de Dietas para la consulta de borradores propios. */
export let MENSAJES_BORRADORES;
export function publicarMensajesBorradores(mensajes) { MENSAJES_BORRADORES = mensajes; }

export function crearTraductorBorradoresDietas(traducirBase) {
  if (typeof traducirBase !== "function")
    throw new TypeError("traductor de Dietas no disponible");
  if (!MENSAJES_BORRADORES) throw new Error("textos de Dietas sin preparar");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_BORRADORES, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_BORRADORES[clave].replace(
      /\{([a-z_]+)\}/gu,
      (_coincidencia, variable) => String(variables[variable] ?? ""),
    );
  };
}
