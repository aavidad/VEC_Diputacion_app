import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-subsanacion-reparos");

export const MENSAJES_SUBSANACION_REPAROS_ES = catalogos.exportaciones.ES;

export const MENSAJES_SUBSANACION_REPAROS_EN = catalogos.exportaciones.EN;
