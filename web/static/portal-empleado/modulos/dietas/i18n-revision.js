/** Textos del encuadre de consulta y revisión de Dietas. */
export const MENSAJES_REVISION_DIETAS_ES = Object.freeze({
  revision_mis_comisiones: "Mis comisiones",
  revision_titulo: "Revisión de comisión",
});

export function crearTraductorRevisionDietas(traducirBase) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_REVISION_DIETAS_ES, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_REVISION_DIETAS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
