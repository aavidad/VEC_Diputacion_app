/** Textos del encuadre de consulta y revisión de Dietas. */
export let MENSAJES_REVISION_DIETAS;
export function publicarMensajesRevisionDietas(mensajes) { MENSAJES_REVISION_DIETAS = mensajes; }

export function crearTraductorRevisionDietas(traducirBase) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  if (!MENSAJES_REVISION_DIETAS) throw new Error("textos de Dietas sin preparar");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_REVISION_DIETAS, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_REVISION_DIETAS[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
