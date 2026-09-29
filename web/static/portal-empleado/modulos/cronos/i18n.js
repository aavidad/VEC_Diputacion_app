import { cargarTextos } from "../../../comun/textos.js";

/**
 * Textos de Cronos: viven en `textos/<idioma>/cronos.json` (una sección por
 * apartado; esta es `general`). Los valores de dominio llegan ya localizados
 * por API.
 */
export const MENSAJES_CRONOS = (await cargarTextos("cronos")).seccion("general");

const CLAVES = Object.freeze(Object.keys(MENSAJES_CRONOS));

export function crearTraductorCronos(catalogo = MENSAJES_CRONOS) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Cronos incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
