/**
 * Textos de la pantalla de reglas vigentes. Viven en los catálogos
 * `textos/<idioma>/reglas.json`; aquí solo se leen.
 *
 * `enlace.js` (y con él este fichero) entra en el grafo estático del portal:
 * los lectores comunes se cargan con `import()` para no añadirlos a la
 * precarga de `index.html`, como en Documentos.
 */
const { cargarTextos } = await import("../../comun/textos.js");
const { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, LOCALIZACION_ACTUAL } = await import("../../comun/idioma.js");

/** Idioma de la interfaz, para el atributo `lang` de la página. */
export const IDIOMA_REGLAS = IDIOMA_ACTUAL;

/** Idioma en que el catálogo de reglas escribe sus textos: el idioma por defecto. */
export const IDIOMA_DATOS_REGLAS = IDIOMA_POR_DEFECTO;

export const MENSAJES_REGLAS = (await cargarTextos("reglas")).seccion("general");

const CLAVES = Object.freeze(Object.keys(MENSAJES_REGLAS));

export function crearTraductorReglas(catalogo = MENSAJES_REGLAS) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de reglas incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`clave i18n de reglas desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}

/** ¿Existe la clave? Para textos que dependen de un valor del servidor. */
export const existeClaveReglas = (clave) => Object.hasOwn(MENSAJES_REGLAS, clave);

export function formatearNumero(n) {
  return new Intl.NumberFormat(LOCALIZACION_ACTUAL).format(n);
}

/** Minúsculas según el idioma de la interfaz, para buscar sin distinguir mayúsculas. */
export const minusculas = (texto) => String(texto ?? "").toLocaleLowerCase(LOCALIZACION_ACTUAL);
