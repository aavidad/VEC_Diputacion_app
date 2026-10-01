/** Textos de la cancelación del expediente antes de la fiscalización. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-cancelacion");

export const MENSAJES_CANCELACION = catalogos.exportaciones.ES;

export const MENSAJES_CANCELACION_EN = catalogos.exportaciones.EN;

export function crearTraductorCancelacion(mensajes = {}) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[`cancelacion.${clave}`] === "string"
      ? mensajes[`cancelacion.${clave}`]
      : MENSAJES_CANCELACION[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, nombre) => (Object.hasOwn(valores, nombre) ? String(valores[nombre]) : `{${nombre}}`));
  };
}
