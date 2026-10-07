
export let MENSAJES_CONSULTA_CRONOS;
export function instalarMENSAJES_CONSULTA_CRONOS(mensajes) { MENSAJES_CONSULTA_CRONOS = mensajes; }

export function crearTraductorConsultaCronos(mensajes = MENSAJES_CONSULTA_CRONOS) {
  if (!MENSAJES_CONSULTA_CRONOS) throw new TypeError("catálogo de consulta Cronos sin preparar");
  const claves = Object.keys(MENSAJES_CONSULTA_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de consulta Cronos incompleto");
  }
  return (clave, variables = {}) => {
    if (!claves.includes(clave)) throw new TypeError("clave de consulta Cronos desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
