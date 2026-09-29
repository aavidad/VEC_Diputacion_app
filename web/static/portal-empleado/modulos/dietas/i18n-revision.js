import { cargarTextos } from "../../../comun/textos.js";

/** Textos del encuadre de consulta y revisión de Dietas. */
export const MENSAJES_REVISION_DIETAS = (await cargarTextos("dietas")).seccion("revision");

export function crearTraductorRevisionDietas(traducirBase) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_REVISION_DIETAS, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* Catálogos antiguos sin la extensión usan el texto castellano. */ }
    return MENSAJES_REVISION_DIETAS[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
