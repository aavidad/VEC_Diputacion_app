import { cargarTextos } from "../../../comun/textos.js";
import { crearTraductorNotificacionesCronos } from "./i18n-notificaciones.js";

export const MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS = (await cargarTextos("cronos-bandeja-notificaciones")).seccion("bandeja");

/** Aporta los textos de recuperación a la autoridad i18n ya existente. */
export function crearTraductorBandejaNotificacionesCronos(mensajes) {
  const base = crearTraductorNotificacionesCronos(mensajes);
  const catalogo = { ...MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS, ...mensajes };
  const claves = Object.keys(MENSAJES_BANDEJA_NOTIFICACIONES_CRONOS);
  if (claves.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) throw new TypeError("catálogo de bandeja Cronos incompleto");
  return (clave, variables = {}) => claves.includes(clave)
    ? catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""))
    : base(clave, variables);
}
