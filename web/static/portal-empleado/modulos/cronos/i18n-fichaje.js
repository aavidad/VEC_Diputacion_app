import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_FICHAJE_CRONOS = (await cargarTextos("cronos-fichaje")).seccion("general");
const CLAVES = Object.freeze(Object.keys(MENSAJES_FICHAJE_CRONOS));

export function crearTraductorFichajeCronos(catalogo = MENSAJES_FICHAJE_CRONOS) {
  if (!catalogo || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo de fichaje incompleto");
  }
  return (clave) => {
    if (!CLAVES.includes(clave)) throw new TypeError("clave de fichaje desconocida");
    return catalogo[clave];
  };
}
