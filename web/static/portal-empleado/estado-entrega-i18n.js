import { cargarTextos } from "../comun/textos.js";

/**
 * Textos fijos de la interfaz común de estado de entrega: sección
 * `estado_entrega` de `textos/<idioma>/portal.json`. Los textos variables
 * pertenecen a quien monta la vista y se escapan al renderizar.
 */
export const MENSAJES_ESTADO_ENTREGA = (await cargarTextos("portal")).seccion("estado_entrega");

export const CLAVES_ESTADO_ENTREGA = Object.freeze(Object.keys(MENSAJES_ESTADO_ENTREGA));

function esCatalogoCerrado(catalogo) {
  if (catalogo === null || typeof catalogo !== "object" || Array.isArray(catalogo)
    || Object.getPrototypeOf(catalogo) !== Object.prototype) return false;
  const claves = Object.keys(catalogo);
  return claves.length === CLAVES_ESTADO_ENTREGA.length
    && CLAVES_ESTADO_ENTREGA.every((clave) => typeof catalogo[clave] === "string" && catalogo[clave].trim() !== "");
}

/** Crea un traductor estricto: no admite catálogos ni claves incompletas. */
export function crearTraductorEstadoEntrega(catalogo = MENSAJES_ESTADO_ENTREGA) {
  if (!esCatalogoCerrado(catalogo)) throw new TypeError("catálogo de estado de entrega incompleto o no permitido");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_ESTADO_ENTREGA, clave)) {
      throw new RangeError(`clave de estado de entrega desconocida: ${String(clave)}`);
    }
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
}
