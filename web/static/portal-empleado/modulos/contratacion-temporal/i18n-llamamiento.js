/** Textos del paso 6; el adaptador de desarrollo no acredita envío de correo. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";
const catalogos = await cargarCatalogosContratacion("contratacion-temporal-llamamiento");

export const MENSAJES_LLAMAMIENTO_ES = catalogos.exportaciones.ES;

/** Step 6 text; the development adapter does not constitute evidence that an email was sent. */
export const MENSAJES_LLAMAMIENTO_EN = catalogos.exportaciones.EN;
