import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_CORRECCIONES_C5B = (await cargarTextos("cronos-correcciones-c5b")).seccion("correcciones");
const CLAVES = Object.freeze(Object.keys(MENSAJES_CORRECCIONES_C5B));

export function crearTraductorCorreccionesC5B(mensajes = MENSAJES_CORRECCIONES_C5B) {
  const catalogo = { ...MENSAJES_CORRECCIONES_C5B, ...mensajes };
  if (CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo de correcciones incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`texto de correcciones desconocido: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}
