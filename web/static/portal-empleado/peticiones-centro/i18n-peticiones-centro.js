/** Textos propios de la petición del centro, cargados solo en el idioma elegido. */
import { cargarTextos, reintentarTextos } from "../../comun/textos.js";

let textos = null;
let generales = null;
let analisis = null;
let promesaAnalisis = null;
let promesaReintento = null;

export let LOCALIZACION_PETICIONES_CENTRO;
export let IDIOMA_EFECTIVO_PETICIONES_CENTRO;
export let MENSAJES_AYUDA_PETICIONES_CENTRO;
export let MENSAJES_INCORPORACIONES_CENTRO;
export let MENSAJES_CANCELACIONES_CENTRO;
export let TEXTOS_LOCALES_PETICIONES_CENTRO;
export let ERROR_TEXTOS_PETICIONES_CENTRO = null;

function instalar(catalogo) {
  textos = catalogo;
  generales = textos.seccion("general");
  LOCALIZACION_PETICIONES_CENTRO = textos.localizacion;
  IDIOMA_EFECTIVO_PETICIONES_CENTRO = textos.idioma;
  MENSAJES_AYUDA_PETICIONES_CENTRO = textos.seccion("ayuda");
  MENSAJES_INCORPORACIONES_CENTRO = textos.seccion("incorporaciones");
  MENSAJES_CANCELACIONES_CENTRO = textos.seccion("cancelaciones");
  TEXTOS_LOCALES_PETICIONES_CENTRO = Object.freeze(Object.fromEntries(
    Object.entries(generales).filter(([clave]) => clave.startsWith("local_"))
      .map(([clave, valor]) => [clave.slice(6), valor]),
  ));
  ERROR_TEXTOS_PETICIONES_CENTRO = null;
  return textos;
}

try { instalar(await cargarTextos("peticiones-centro")); }
catch (error) { ERROR_TEXTOS_PETICIONES_CENTRO = error; }

/** Un fallo de catálogo deja el módulo importable para ofrecer un reintento visible. */
export function prepararTextosPeticionesCentro() {
  if (textos) return Promise.resolve(textos);
  if (!promesaReintento) {
    promesaReintento = reintentarTextos("peticiones-centro").then(instalar).catch((error) => {
      ERROR_TEXTOS_PETICIONES_CENTRO = error;
      throw error;
    }).finally(() => { promesaReintento = null; });
  }
  return promesaReintento;
}

export function traducirPeticionesCentro(clave, variables = {}) {
  const plantilla = generales?.[clave] ?? analisis?.[clave];
  if (typeof plantilla !== "string") throw new Error(`clave de petición del centro desconocida: ${clave}`);
  return Object.entries(variables).reduce((texto, [nombre, valor]) =>
    texto.replaceAll(`{${nombre}}`, String(valor)), plantilla);
}

/** Las causas de fin solo se leen cuando una petición o un formulario las necesita. */
export function prepararAnalisisPeticionesCentro() {
  if (!textos) return Promise.reject(ERROR_TEXTOS_PETICIONES_CENTRO ?? new Error("textos de petición no disponibles"));
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
