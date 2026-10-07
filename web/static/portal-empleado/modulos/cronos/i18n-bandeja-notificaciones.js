import { crearTraductorNotificacionesCronos } from "./i18n-notificaciones.js";

export let MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS;
export function instalarMENSAJES_BANDEJA_NOTIFICACIONES_CRONOS(mensajes) { MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS = mensajes; }

/** Aporta los textos de recuperación a la autoridad i18n ya existente. */
export function crearTraductorBandejaNotificacionesCronos(mensajes) {
  if (!MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS) throw new TypeError("catálogo de bandeja Cronos sin preparar");
  const base = crearTraductorNotificacionesCronos(mensajes);
  const catalogo = { ...MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS, ...mensajes };
  const claves = Object.keys(MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS);
  if (claves.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) throw new TypeError("catálogo de bandeja Cronos incompleto");
  return (clave, variables = {}) => claves.includes(clave)
    ? catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""))
    : base(clave, variables);
}
