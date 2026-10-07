import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-borradores-publicados");

export const MENSAJES_BORRADORES_PUBLICADOS_ES = catalogos.exportaciones.ES;

export const MENSAJES_BORRADORES_PUBLICADOS_EN = catalogos.exportaciones.EN;
