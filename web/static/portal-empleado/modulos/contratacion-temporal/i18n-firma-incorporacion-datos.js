/** Textos pendientes de incorporar a los agregadores del portal y expedientes. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const GRUPOS_EXPEDIENTES = Object.freeze([
  "incorporacion", "hito", "continuidad", "firma", "borrador", "firma_pendiente",
]);
const portal = await cargarCatalogosContratacion("contratacion-temporal-firma-incorporacion-portal");
const expedientes = Object.fromEntries(await Promise.all(GRUPOS_EXPEDIENTES.map(async (grupo) => [
  grupo, await cargarCatalogosContratacion("contratacion-temporal-firma-incorporacion-expedientes", grupo),
])));

const grupos = Object.freeze(Object.fromEntries(GRUPOS_EXPEDIENTES.map((grupo) => [
  grupo, expedientes[grupo].exportaciones,
])));

export const MENSAJES_FIRMA_INCORPORACION = Object.freeze({
  portal: portal.exportaciones,
  grupos,
  expedientes: Object.freeze({
    ES: Object.freeze(Object.assign({}, ...GRUPOS_EXPEDIENTES.map((grupo) => grupos[grupo].ES))),
    EN: Object.freeze(Object.assign({}, ...GRUPOS_EXPEDIENTES.map((grupo) => grupos[grupo].EN))),
  }),
});
