/** Textos del análisis que dependen del catálogo de reglas: duración máxima y urgencia. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";
const catalogos = await cargarCatalogosContratacion("contratacion-temporal-analisis-catalogo");

export const MENSAJES_ANALISIS_CATALOGO_ES = catalogos.exportaciones.ES;

export const MENSAJES_ANALISIS_CATALOGO_EN = catalogos.exportaciones.EN;
