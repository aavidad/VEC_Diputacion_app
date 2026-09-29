import { cargarTextos } from "../../../comun/textos.js";
import { MENSAJES_BORRADORES } from "./i18n-borradores.js?v=20260929-i18n-dietas-v1";
import { MENSAJES_REVISION_DIETAS } from "./i18n-revision.js?v=20260929-i18n-dietas-v1";
import { MENSAJES_CIRCUITO_DIETAS } from "./i18n-circuito.js?v=20260929-i18n-dietas-v1";
import { MENSAJES_RECTIFICACION_DIETAS } from "./i18n-rectificacion-dietas.js?v=20260929-i18n-dietas-v1";
import { MENSAJES_RECTIFICACION_ADMIN } from "./i18n-rectificacion-admin.js?v=20260929-i18n-dietas-v1";

const TEXTOS_DIETAS = await cargarTextos("dietas");

/** Catálogo completo de textos propios de la superficie Dietas. */
export const MENSAJES_DIETAS = Object.freeze({
  ...TEXTOS_DIETAS.seccion("general"),
  ...MENSAJES_BORRADORES,
  ...MENSAJES_REVISION_DIETAS,
  ...MENSAJES_CIRCUITO_DIETAS,
  ...MENSAJES_RECTIFICACION_DIETAS,
  ...MENSAJES_RECTIFICACION_ADMIN,
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_DIETAS));

export function crearTraductorDietas(catalogo = MENSAJES_DIETAS) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Dietas incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Dietas desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
