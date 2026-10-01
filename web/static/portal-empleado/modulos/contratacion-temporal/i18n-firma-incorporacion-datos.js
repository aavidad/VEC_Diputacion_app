/** Textos pendientes de incorporar a los agregadores del portal y expedientes. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js";

const [portal, expedientes] = await Promise.all([
  cargarCatalogosContratacion("contratacion-temporal-firma-incorporacion-portal"),
  cargarCatalogosContratacion("contratacion-temporal-firma-incorporacion-expedientes"),
]);

export const MENSAJES_FIRMA_INCORPORACION_PORTAL_ES = portal.exportaciones.ES;
export const MENSAJES_FIRMA_INCORPORACION_PORTAL_EN = portal.exportaciones.EN;
export const MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_ES = expedientes.exportaciones.ES;
export const MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_EN = expedientes.exportaciones.EN;
