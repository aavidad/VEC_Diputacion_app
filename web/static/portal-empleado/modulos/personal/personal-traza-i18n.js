import { cargarTextos } from "../../../comun/textos.js";

const MENSAJES_TRAZA = (await cargarTextos("personal-traza")).seccion("general");

export function crearTraductorTraza(catalogo = MENSAJES_TRAZA) {
  const claves = Object.keys(MENSAJES_TRAZA);
  if (!catalogo || typeof catalogo !== "object" || claves.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new Error("catálogo de procedencia de Personal incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_TRAZA, clave)) throw new Error(`clave de procedencia desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
