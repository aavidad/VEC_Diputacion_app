
export let MENSAJES_FICHAJE_CRONOS;
export function instalarMENSAJES_FICHAJE_CRONOS(mensajes) { MENSAJES_FICHAJE_CRONOS = mensajes; }
export function crearTraductorFichajeCronos(catalogo = MENSAJES_FICHAJE_CRONOS) {
  if (!MENSAJES_FICHAJE_CRONOS) throw new TypeError("catálogo de fichaje sin preparar");
  const CLAVES = Object.keys(MENSAJES_FICHAJE_CRONOS);
  if (!catalogo || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo de fichaje incompleto");
  }
  const vigente = MENSAJES_FICHAJE_CRONOS;
  return (clave) => {
    if (MENSAJES_FICHAJE_CRONOS !== vigente) throw new TypeError("catálogo de fichaje sustituido");
    if (!CLAVES.includes(clave)) throw new TypeError("clave de fichaje desconocida");
    return catalogo[clave];
  };
}
