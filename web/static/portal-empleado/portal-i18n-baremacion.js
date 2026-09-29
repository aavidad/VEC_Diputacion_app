import { cargarTextos } from "../comun/textos.js";

/** Textos de baremación, ranking y alegaciones: `textos/<idioma>/portal-bolsa.json`. */
export const MENSAJES_BAREMACION = (await cargarTextos("portal-bolsa")).seccion("baremacion");

const CLAVES = Object.freeze(Object.keys(MENSAJES_BAREMACION));

export function crearTraductorBaremacion(catalogo = MENSAJES_BAREMACION) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new Error("catálogo i18n de baremación incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de baremación desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirBaremacion = crearTraductorBaremacion();
