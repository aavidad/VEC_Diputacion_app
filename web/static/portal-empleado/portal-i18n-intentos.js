import { cargarTextos } from "../comun/textos.js";

/** Textos del control de intentos de contacto del llamamiento: `textos/<idioma>/portal-bolsa.json`. */
export const MENSAJES_INTENTOS = (await cargarTextos("portal-bolsa")).seccion("intentos");

const CLAVES = Object.freeze(Object.keys(MENSAJES_INTENTOS));
export function crearTraductorIntentos(catalogo = MENSAJES_INTENTOS) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de intentos de contacto incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de intentos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, variable) => String(variables[variable] ?? ""));
  };
}
export const traducirIntentos = crearTraductorIntentos();
