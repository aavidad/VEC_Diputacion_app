/** Textos visibles de vistas de Contratación temporal antes escritos en el código. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js";
const catalogos = await cargarCatalogosContratacion("contratacion-temporal-textos-vistas");

export const MENSAJES_TEXTOS_VISTAS_ES = catalogos.exportaciones.ES;

export const MENSAJES_TEXTOS_VISTAS_EN = catalogos.exportaciones.EN;
