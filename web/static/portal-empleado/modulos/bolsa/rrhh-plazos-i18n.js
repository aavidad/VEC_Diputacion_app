/** Textos de la configuración versionada de ofertas de Bolsa: sección `plazos` de `textos/<idioma>/bolsa.json`. */
import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_RRHH_PLAZOS = (await cargarTextos("bolsa")).seccion("plazos");

export function crearTraductorRRHHPlazos(catalogo = MENSAJES_RRHH_PLAZOS) {
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave) || typeof catalogo[clave] !== "string") {
      throw new TypeError(`clave i18n de política de ofertas desconocida: ${clave}`);
    }
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
