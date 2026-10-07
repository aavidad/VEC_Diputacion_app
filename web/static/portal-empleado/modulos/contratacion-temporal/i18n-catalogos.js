/** Catálogos comunes y compatibilidad de las exportaciones existentes. */
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";
import { cargarTextos } from "../../../comun/textos.js";

const idiomasExportados = (await cargarTextos("contratacion-temporal-compatibilidad"))
  .seccion("idiomas_exportados");

export async function cargarCatalogosContratacion(modulo, seccion = "general") {
  const entradas = await Promise.all(IDIOMAS_DISPONIBLES.map(async ({ codigo }) => [
    codigo, (await cargarTextos(modulo, { idioma: codigo })).seccion(seccion),
  ]));
  const porIdioma = Object.freeze(Object.fromEntries(entradas));
  const exportaciones = Object.freeze(Object.fromEntries(
    Object.entries(idiomasExportados).map(([nombre, codigo]) => {
      return [nombre, porIdioma[codigo] ?? porIdioma[IDIOMA_POR_DEFECTO]];
    }),
  ));
  return Object.freeze({ porIdioma, exportaciones, actual: porIdioma[IDIOMA_ACTUAL] });
}
