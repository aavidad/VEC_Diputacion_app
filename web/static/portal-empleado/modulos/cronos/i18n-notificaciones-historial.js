
export let MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS;
export function instalarMENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS(mensajes) { MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS = mensajes; }

export function crearTraductorNotificacionesHistorialCronos(mensajes = MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS) {
  if (!MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS) throw new TypeError("catálogo de historial de notificaciones Cronos sin preparar");
  const claves = Object.keys(MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de historial de notificaciones Cronos incompleto");
  }
  return (clave, variables = {}) => {
    if (!claves.includes(clave)) throw new TypeError("clave de historial de notificaciones Cronos desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
