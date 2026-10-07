/** Textos castellanos del histórico de cambios del expediente (petición RRHH p.4). */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-cambios-expediente");

export const MENSAJES_CAMBIOS_EXPEDIENTE_ES = catalogos.exportaciones.ES;

export const MENSAJES_CAMBIOS_EXPEDIENTE_EN = catalogos.exportaciones.EN;
