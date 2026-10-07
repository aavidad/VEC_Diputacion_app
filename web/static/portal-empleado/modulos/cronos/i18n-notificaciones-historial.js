import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS = (await cargarTextos("cronos-notificaciones-historial")).seccion("historial");

export function crearTraductorNotificacionesHistorialCronos(mensajes = MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS) {
  const claves = Object.keys(MENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de historial de notificaciones Cronos incompleto");
  }
  return (clave, variables = {}) => {
    if (!claves.includes(clave)) throw new TypeError("clave de historial de notificaciones Cronos desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
