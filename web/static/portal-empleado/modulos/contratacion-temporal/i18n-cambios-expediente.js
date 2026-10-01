/** Textos castellanos del histórico de cambios del expediente (petición RRHH p.4). */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-cambios-expediente");

export const MENSAJES_CAMBIOS_EXPEDIENTE_ES = catalogos.exportaciones.ES;

export const MENSAJES_CAMBIOS_EXPEDIENTE_EN = catalogos.exportaciones.EN;
