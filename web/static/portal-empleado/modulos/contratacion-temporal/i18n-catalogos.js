/** Catálogos de Contratación temporal para el idioma de esta navegación. */
import { cargarTextos } from "../../../comun/textos.js";

/**
 * Las exportaciones históricas ES/EN apuntan al catálogo elegido en esta
 * navegación. El lector común concentra reintento, respaldo y su incidencia.
 */
export async function cargarCatalogosContratacionEnIdioma(modulo, idioma, seccion = "general") {
  // La primera carga resuelve el índice sin ocultar una incidencia de lectura.
  const compatibilidad = await cargarTextos("contratacion-temporal-compatibilidad", { idioma });
  const elegido = idioma ?? compatibilidad.idioma;
  const textos = await cargarTextos(modulo, { idioma: elegido });
  const actual = textos.seccion(seccion);
  const exportaciones = Object.freeze(Object.fromEntries(
    Object.keys(compatibilidad.seccion("idiomas_exportados"))
      .map((nombre) => [nombre, actual]),
  ));
  return Object.freeze({
    porIdioma: Object.freeze({ [textos.idioma]: actual }),
    exportaciones, actual, idioma: textos.idioma,
    incidenciaCatalogo: textos.incidenciaCatalogo,
    incidenciaIndice: textos.incidenciaIndice,
  });
}

export async function cargarCatalogosContratacion(modulo, seccion = "general") {
  return cargarCatalogosContratacionEnIdioma(modulo, undefined, seccion);
}
