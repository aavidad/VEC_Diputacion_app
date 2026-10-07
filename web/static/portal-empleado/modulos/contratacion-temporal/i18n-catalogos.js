/** Catálogos de Contratación temporal para el idioma de esta navegación. */
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, localizacionDe } from "../../../comun/idioma.js";
import { crearTextos, leerCatalogoUnaVez, urlCatalogo } from "../../../comun/textos.js";

async function leerSeccion(modulo, idioma, seccion) {
  const datos = await leerCatalogoUnaVez(urlCatalogo(idioma, modulo));
  return crearTextos({
    modulo, idioma, localizacion: localizacionDe(idioma), respaldo: datos,
  }).seccion(seccion);
}

async function leerConReintento(modulo, idioma, seccion) {
  try {
    return await leerSeccion(modulo, idioma, seccion);
  } catch (primeraCausa) {
    console.warn(`No se pudo cargar el catálogo ${modulo} (${idioma}); se reintenta.`, primeraCausa);
    return leerSeccion(modulo, idioma, seccion);
  }
}

async function leerActivo(modulo, seccion, idioma = IDIOMA_ACTUAL) {
  try {
    return await leerConReintento(modulo, idioma, seccion);
  } catch (causa) {
    if (idioma === IDIOMA_POR_DEFECTO) throw causa;
    console.warn(`No se pudo cargar el catálogo ${modulo} (${idioma}); se usa el idioma por defecto.`, causa);
    return leerConReintento(modulo, IDIOMA_POR_DEFECTO, seccion);
  }
}

/**
 * Las exportaciones históricas ES/EN se conservan para los importadores de CT.
 * En una navegación solo se resuelve el catálogo activo. Ambas referencias
 * apuntan a ese catálogo hasta que los importadores adopten `actual`.
 */
export async function cargarCatalogosContratacionEnIdioma(modulo, idioma, seccion = "general") {
  const [idiomasExportados, actual] = await Promise.all([
    leerActivo("contratacion-temporal-compatibilidad", "idiomas_exportados", idioma),
    leerActivo(modulo, seccion, idioma),
  ]);
  const exportaciones = Object.freeze(Object.fromEntries(
    Object.keys(idiomasExportados).map((nombre) => [nombre, actual]),
  ));
  return Object.freeze({
    porIdioma: Object.freeze({ [idioma]: actual }),
    exportaciones, actual,
  });
}

export async function cargarCatalogosContratacion(modulo, seccion = "general") {
  return cargarCatalogosContratacionEnIdioma(modulo, IDIOMA_ACTUAL, seccion);
}
