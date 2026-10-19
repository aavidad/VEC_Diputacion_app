import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=VEC_WEB_VERSION";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-borradores-publicados");

export const MENSAJES_BORRADORES_PUBLICADOS_ES = catalogos.exportaciones.ES;

export const MENSAJES_BORRADORES_PUBLICADOS_EN = catalogos.exportaciones.EN;
