import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-moad");
export const MENSAJES_MOAD_ES = catalogos.exportaciones.ES;
export const MENSAJES_MOAD_EN = catalogos.exportaciones.EN;
