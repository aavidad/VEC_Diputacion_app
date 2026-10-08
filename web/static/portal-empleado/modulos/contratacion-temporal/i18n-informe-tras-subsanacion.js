/** Textos castellanos del informe jurídico nuevo tras subsanar un reparo (duda 5). */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-informe-tras-subsanacion");
export const MENSAJES_INFORME_TRAS_SUBSANACION_ACTUAL = catalogos.actual;

export const MENSAJES_INFORME_TRAS_SUBSANACION_ES = catalogos.exportaciones.ES;

export const MENSAJES_INFORME_TRAS_SUBSANACION_EN = catalogos.exportaciones.EN;
