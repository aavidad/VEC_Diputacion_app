/** Textos del rechazo por falta de firma antes de la remisión a Intervención. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-firma-remision");
export const MENSAJES_FIRMA_REMISION_ES = catalogos.exportaciones.ES;
export const MENSAJES_FIRMA_REMISION_EN = catalogos.exportaciones.EN;
