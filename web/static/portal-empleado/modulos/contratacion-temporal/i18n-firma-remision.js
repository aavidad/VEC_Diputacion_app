/** Textos del rechazo por falta de firma antes de la remisión a Intervención. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-firma-remision");
export const MENSAJES_FIRMA_REMISION_ACTUAL = catalogos.actual;
export const MENSAJES_FIRMA_REMISION_ES = catalogos.exportaciones.ES;
export const MENSAJES_FIRMA_REMISION_EN = catalogos.exportaciones.EN;
