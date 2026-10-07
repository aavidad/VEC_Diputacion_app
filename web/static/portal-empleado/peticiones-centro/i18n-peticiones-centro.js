/** Textos propios de la petición del centro, cargados solo en el idioma elegido. */
import { cargarTextos } from "../../comun/textos.js";

const textos = await cargarTextos("peticiones-centro");
const generales = textos.seccion("general");
const ayuda = textos.seccion("ayuda");
let analisis = null;
let promesaAnalisis = null;

export const LOCALIZACION_PETICIONES_CENTRO = textos.localizacion;
export const IDIOMA_EFECTIVO_PETICIONES_CENTRO = textos.idioma;
export const MENSAJES_AYUDA_PETICIONES_CENTRO = ayuda;
export const MENSAJES_INCORPORACIONES_CENTRO = textos.seccion("incorporaciones");
export const MENSAJES_CANCELACIONES_CENTRO = textos.seccion("cancelaciones");
export const TEXTOS_LOCALES_PETICIONES_CENTRO = Object.freeze(Object.fromEntries(
  Object.entries(generales).filter(([clave]) => clave.startsWith("local_"))
    .map(([clave, valor]) => [clave.slice(6), valor]),
));

export function traducirPeticionesCentro(clave, variables = {}) {
  const plantilla = generales[clave] ?? analisis?.[clave];
  if (typeof plantilla !== "string") throw new Error(`clave de petición del centro desconocida: ${clave}`);
  return Object.entries(variables).reduce((texto, [nombre, valor]) =>
    texto.replaceAll(`{${nombre}}`, String(valor)), plantilla);
}

/** Las causas de fin solo se leen cuando una petición o un formulario las necesita. */
export function prepararAnalisisPeticionesCentro() {
  if (analisis) return Promise.resolve(analisis);
  if (!promesaAnalisis) {
    promesaAnalisis = (async () => {
      const catalogo = await cargarTextos("contratacion-temporal-analisis-catalogo", { idioma: textos.idioma });
      if (catalogo.idioma !== textos.idioma) throw new Error("catálogo de análisis en otro idioma");
      analisis = catalogo.seccion("general");
      return analisis;
    })().catch((error) => {
      promesaAnalisis = null;
      throw error;
    });
  }
  return promesaAnalisis;
}
