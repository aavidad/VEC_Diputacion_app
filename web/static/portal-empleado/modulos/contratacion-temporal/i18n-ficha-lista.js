/** Textos de la ficha y la lista de peticiones (dirección de diseño del 29/09/2026). */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";

const [catalogos, plazos] = await Promise.all([
  cargarCatalogosContratacion("contratacion-temporal-ficha-lista"),
  cargarCatalogosContratacion("contratacion-temporal-lista-plazos", "lista"),
]);

export const MENSAJES_FICHA_LISTA_ES = catalogos.exportaciones.ES
  ? Object.freeze({ ...plazos.exportaciones.ES, ...catalogos.exportaciones.ES }) : undefined;

export const MENSAJES_FICHA_LISTA_EN = catalogos.exportaciones.EN
  ? Object.freeze({ ...plazos.exportaciones.EN, ...catalogos.exportaciones.EN }) : undefined;
