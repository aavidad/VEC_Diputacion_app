/**
 * Textos de los recuadros y datos de Bolsa que llevan a su lista o ficha
 * (recuento → lista filtrada, nombre → ficha). Catálogo propio y pequeño para
 * no renovar la caché de todo el catálogo común del portal.
 */
import { IDIOMA_ACTUAL } from "../comun/idioma.js";

export const MENSAJES_ENLACES_BOLSA_ES = Object.freeze({
  kpi_bolsas_aria: "Bolsas: {total}. Abrir el cuadro de mando con la relación de bolsas",
  kpi_ver_cuadro: "Ver en el cuadro",
  historico_abrir_ficha_aria: "Abrir la ficha de {nombre}",
  llamamientos_curso_aria: "{total} llamamientos en curso de {bolsa}. Abrir su histórico de llamamientos",
});

export const MENSAJES_ENLACES_BOLSA_EN = Object.freeze({
  kpi_bolsas_aria: "Pools: {total}. Open the dashboard with the list of pools",
  kpi_ver_cuadro: "View in dashboard",
  historico_abrir_ficha_aria: "Open the record for {nombre}",
  llamamientos_curso_aria: "{total} ongoing call-ups for {bolsa}. Open their call-up history",
});

export function traducirEnlacesBolsa(clave, variables = {}) {
  if (!Object.hasOwn(MENSAJES_ENLACES_BOLSA_ES, clave)) throw new Error(`clave i18n de enlaces de Bolsa desconocida: ${clave}`);
  const catalogo = IDIOMA_ACTUAL === "en" ? MENSAJES_ENLACES_BOLSA_EN : MENSAJES_ENLACES_BOLSA_ES;
  return catalogo[clave].replace(/\{([a-z_]+)\}/g,
    (_coincidencia, variable) => String(variables[variable] ?? ""));
}
