/**
 * Textos de los recuadros y datos de Bolsa que llevan a su lista o ficha
 * (recuento → lista filtrada, nombre → ficha). Catálogo propio y pequeño para
 * no renovar la caché de todo el catálogo común del portal.
 */
export const MENSAJES_ENLACES_BOLSA_ES = Object.freeze({
  kpi_bolsas_aria: "Bolsas: {total}. Abrir el cuadro de mando con la relación de bolsas",
  kpi_ver_cuadro: "Ver en el cuadro",
  historico_abrir_ficha_aria: "Abrir la ficha de {nombre}",
  llamamientos_curso_aria: "{total} llamamientos en curso de {bolsa}. Abrir su histórico de llamamientos",
});

export function traducirEnlacesBolsa(clave, variables = {}) {
  if (!Object.hasOwn(MENSAJES_ENLACES_BOLSA_ES, clave)) throw new Error(`clave i18n de enlaces de Bolsa desconocida: ${clave}`);
  return MENSAJES_ENLACES_BOLSA_ES[clave].replace(/\{([a-z_]+)\}/g,
    (_coincidencia, variable) => String(variables[variable] ?? ""));
}
