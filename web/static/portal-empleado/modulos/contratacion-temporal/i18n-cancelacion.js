/** Textos de la cancelación del expediente antes de la fiscalización. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-cancelacion");

export const MENSAJES_CANCELACION = catalogos.actual;

export const MENSAJES_CANCELACION_EN = catalogos.exportaciones.EN;

export function crearTraductorCancelacion(mensajes = {}) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[`cancelacion.${clave}`] === "string"
      ? mensajes[`cancelacion.${clave}`]
      : catalogos.actual[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, nombre) => (Object.hasOwn(valores, nombre) ? String(valores[nombre]) : `{${nombre}}`));
  };
}
