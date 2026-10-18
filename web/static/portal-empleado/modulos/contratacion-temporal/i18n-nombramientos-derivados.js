import { cargarTextos, reintentarTextos } from "../../../comun/textos.js";

export async function cargarTextosNombramientos(locale, reintentar = false) {
  const idioma = locale.split("-")[0];
  const leer = reintentar ? reintentarTextos : cargarTextos;
  const catalogo = await leer("contratacion-temporal-nombramientos", { idioma });
  if (catalogo.idioma !== idioma) throw new Error("catalogo_nombramientos_idioma_no_disponible");
  return catalogo.seccion("general");
}
