import { cargarTextos } from "../comun/textos.js";

/**
 * Textos de los recuadros y datos de Bolsa que llevan a su lista o ficha
 * (recuento → lista filtrada, nombre → ficha): sección `enlaces_bolsa` de
 * `textos/<idioma>/portal.json`.
 */
export const MENSAJES_ENLACES_BOLSA = (await cargarTextos("portal")).seccion("enlaces_bolsa");

export function traducirEnlacesBolsa(clave, variables = {}) {
  if (!Object.hasOwn(MENSAJES_ENLACES_BOLSA, clave)) throw new Error(`clave i18n de enlaces de Bolsa desconocida: ${clave}`);
  return MENSAJES_ENLACES_BOLSA[clave].replace(/\{([a-z_]+)\}/g,
    (_coincidencia, variable) => String(variables[variable] ?? ""));
}
