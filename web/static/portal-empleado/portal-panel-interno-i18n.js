import { cargarTextos } from "../comun/textos.js";

/** Textos del panel interno de Bolsa: sección `panel_interno` de `textos/<idioma>/portal.json`. */
export const MENSAJES_PANEL_INTERNO = (await cargarTextos("portal")).seccion("panel_interno");

/** Aviso fijo del panel en el idioma de la interfaz; rechaza claves desconocidas. */
export function traducirAvisoPanelInterno(clave) {
  if (!Object.hasOwn(MENSAJES_PANEL_INTERNO, clave)) throw new Error(`Aviso de panel desconocido: ${clave}`);
  return MENSAJES_PANEL_INTERNO[clave];
}
