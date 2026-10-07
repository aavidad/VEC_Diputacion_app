/** Catálogos de Contratación temporal para el idioma de esta navegación. */
import { IDIOMA_ACTUAL, prepararIdiomas } from "../../../comun/idioma.js";
import { cargarTextos } from "../../../comun/textos.js";

/**
 * Las exportaciones históricas ES/EN apuntan al catálogo elegido en esta
 * navegación. El lector común concentra reintento, respaldo y su incidencia.
 */
export async function cargarCatalogosContratacionEnIdioma(modulo, idioma, seccion = "general") {
  // El índice se resuelve antes de elegir el idioma: al importar, el valor de
  // IDIOMA_ACTUAL todavía puede ser el provisional del documento.
  await prepararIdiomas().catch(() => null);
  const elegido = idioma ?? IDIOMA_ACTUAL;
  const [compatibilidad, textos] = await Promise.all([
    cargarTextos("contratacion-temporal-compatibilidad", { idioma: elegido }),
    cargarTextos(modulo, { idioma: elegido }),
  ]);
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
