/** Textos de las comprobaciones automáticas de la vía de cobertura (fase 3). */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js";
const catalogos = await cargarCatalogosContratacion("contratacion-temporal-avisos-via-cobertura");

export const MENSAJES_AVISOS_VIA_COBERTURA_ES = catalogos.exportaciones.ES;

export const MENSAJES_AVISOS_VIA_COBERTURA_EN = catalogos.exportaciones.EN;
