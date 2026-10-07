
export let MENSAJES_HISTORIAL_CRONOS;
export function instalarMENSAJES_HISTORIAL_CRONOS(mensajes) { MENSAJES_HISTORIAL_CRONOS = mensajes; }

export function crearTraductorHistorialCronos(mensajes = MENSAJES_HISTORIAL_CRONOS) {
  if (!MENSAJES_HISTORIAL_CRONOS) throw new TypeError("catálogo de historial Cronos sin preparar");
  const claves = Object.keys(MENSAJES_HISTORIAL_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de historial Cronos incompleto");
  }
  const vigente = MENSAJES_HISTORIAL_CRONOS;
  return (clave, variables = {}) => {
    if (MENSAJES_HISTORIAL_CRONOS !== vigente) throw new TypeError("catálogo de historial Cronos sustituido");
    if (!claves.includes(clave)) throw new TypeError("clave de historial Cronos desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
