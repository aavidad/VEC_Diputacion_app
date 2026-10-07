/** Textos del cese, el cierre y la modificación tras el nombramiento. */
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-seguimiento-cese");
export const MENSAJES_SEGUIMIENTO_CESE = catalogos.exportaciones.ES;
export const MENSAJES_SEGUIMIENTO_CESE_EN = catalogos.exportaciones.EN;

export function crearTraductorSeguimientoCese(mensajes = {}) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[`seguimiento_cese.${clave}`] === "string"
      ? mensajes[`seguimiento_cese.${clave}`]
      : MENSAJES_SEGUIMIENTO_CESE[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, nombre) => (Object.hasOwn(valores, nombre) ? String(valores[nombre]) : `{${nombre}}`));
  };
}
