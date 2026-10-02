/** Textos de las comprobaciones automáticas de la vía de cobertura (fase 3). */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";
const catalogos = await cargarCatalogosContratacion("contratacion-temporal-avisos-via-cobertura");

export const MENSAJES_AVISOS_VIA_COBERTURA_ES = catalogos.exportaciones.ES;

export const MENSAJES_AVISOS_VIA_COBERTURA_EN = catalogos.exportaciones.EN;
