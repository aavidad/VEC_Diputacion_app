
/**
 * Textos de Cronos: viven en `textos/<idioma>/cronos.json` (una sección por
 * apartado; esta es `general`). Los valores de dominio llegan ya localizados
 * por API.
 */
export let MENSAJES_CRONOS;
export function instalarMENSAJES_CRONOS(mensajes) { MENSAJES_CRONOS = mensajes; }

export function crearTraductorCronos(catalogo = MENSAJES_CRONOS) {
  if (!MENSAJES_CRONOS) throw new Error("catálogo i18n de Cronos sin preparar");
  const CLAVES = Object.keys(MENSAJES_CRONOS);
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Cronos incompleto");
  }
  const vigente = MENSAJES_CRONOS;
  return (clave, variables = {}) => {
    if (MENSAJES_CRONOS !== vigente) throw new Error("catálogo i18n de Cronos sustituido");
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
