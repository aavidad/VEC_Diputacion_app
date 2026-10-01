/** Textos pendientes de incorporar a los agregadores del portal y expedientes. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js";

const GRUPOS_EXPEDIENTES = Object.freeze([
  "incorporacion", "hito", "continuidad", "firma", "borrador", "firma_pendiente",
]);
const portal = await cargarCatalogosContratacion("contratacion-temporal-firma-incorporacion-portal");
const expedientes = Object.fromEntries(await Promise.all(GRUPOS_EXPEDIENTES.map(async (grupo) => [
  grupo, await cargarCatalogosContratacion("contratacion-temporal-firma-incorporacion-expedientes", grupo),
])));

export const MENSAJES_FIRMA_INCORPORACION_PORTAL_ES = portal.exportaciones.ES;
export const MENSAJES_FIRMA_INCORPORACION_PORTAL_EN = portal.exportaciones.EN;
export const MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_ES = Object.freeze(Object.assign({},
  ...GRUPOS_EXPEDIENTES.map((grupo) => expedientes[grupo].exportaciones.ES)));
export const MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_EN = Object.freeze(Object.assign({},
  ...GRUPOS_EXPEDIENTES.map((grupo) => expedientes[grupo].exportaciones.EN)));

export const MENSAJES_EXPEDIENTES_INCORPORACION_ES = expedientes.incorporacion.exportaciones.ES;
export const MENSAJES_EXPEDIENTES_INCORPORACION_EN = expedientes.incorporacion.exportaciones.EN;
export const MENSAJES_EXPEDIENTES_HITO_ES = expedientes.hito.exportaciones.ES;
export const MENSAJES_EXPEDIENTES_HITO_EN = expedientes.hito.exportaciones.EN;
export const MENSAJES_EXPEDIENTES_CONTINUIDAD_ES = expedientes.continuidad.exportaciones.ES;
export const MENSAJES_EXPEDIENTES_CONTINUIDAD_EN = expedientes.continuidad.exportaciones.EN;
export const MENSAJES_EXPEDIENTES_FIRMA_ES = expedientes.firma.exportaciones.ES;
export const MENSAJES_EXPEDIENTES_FIRMA_EN = expedientes.firma.exportaciones.EN;
export const MENSAJES_EXPEDIENTES_BORRADOR_ES = expedientes.borrador.exportaciones.ES;
export const MENSAJES_EXPEDIENTES_BORRADOR_EN = expedientes.borrador.exportaciones.EN;
export const MENSAJES_EXPEDIENTES_FIRMA_PENDIENTE_ES = expedientes.firma_pendiente.exportaciones.ES;
export const MENSAJES_EXPEDIENTES_FIRMA_PENDIENTE_EN = expedientes.firma_pendiente.exportaciones.EN;
