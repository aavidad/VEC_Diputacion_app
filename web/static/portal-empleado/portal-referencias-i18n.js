/**
 * Textos con los que Bolsa presenta filas cuyo único identificador es una
 * referencia interna: número de orden, datos legibles y botón «Copiar
 * referencia». Viven en la sección `referencias` de `textos/<idioma>/portal.json`;
 * los textos del justificante son los comunes del panel.
 */
import { cargarTextos } from "../comun/textos.js";
import { traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";

export const MENSAJES_REFERENCIAS = (await cargarTextos("portal")).seccion("referencias");

export function tieneTextoReferencia(clave) {
  return Object.hasOwn(MENSAJES_REFERENCIAS, clave);
}

/** Traductor estricto; los textos del botón de copia son los del justificante común. */
export function traducirReferencia(clave, variables = {}) {
  if (clave.startsWith("justificante_")) return traducirPortal(`panel_${clave}`, variables);
  const plantilla = MENSAJES_REFERENCIAS[clave];
  if (typeof plantilla !== "string") throw new Error(`clave i18n de referencias desconocida: ${clave}`);
  return plantilla.replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
}
