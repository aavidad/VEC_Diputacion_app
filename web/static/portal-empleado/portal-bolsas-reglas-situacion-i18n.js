import { cargarTextos } from "../comun/textos.js";

/**
 * Textos del cambio de situación con reglas del catálogo
 * (`textos/<idioma>/portal-bolsa.json`). La ayuda «?» vive en el catálogo común.
 */
export const MENSAJES_REGLAS_SITUACION = (await cargarTextos("portal-bolsa")).seccion("reglas_situacion");

const CLAVES = Object.freeze(Object.keys(MENSAJES_REGLAS_SITUACION));

export function traducirReglasSituacion(clave, variables = {}) {
  if (!CLAVES.includes(clave)) throw new Error(`clave i18n de reglas de situación desconocida: ${clave}`);
  return MENSAJES_REGLAS_SITUACION[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
}

/** Etiqueta de una modalidad; una modalidad nueva del catálogo se muestra por su código. */
export function etiquetaModalidadReposicion(codigo) {
  const clave = `modalidad_${codigo}`;
  return CLAVES.includes(clave) ? MENSAJES_REGLAS_SITUACION[clave] : String(codigo).replaceAll("_", " ");
}
