import { cargarTextos } from "/web/static/comun/textos.js";
import { leerRecursoJSON } from "/web/static/comun/idioma.js";
import { montarCatalogoTarifasDietas } from "/web/static/portal-empleado/modulos/dietas/catalogo/vista.js?v=20261001-dietas-catalogo-propuestas-v1";

// Previsualización aislada; los datos de ejemplo solo se sirven en el recorrido local.
const textos = await cargarTextos("dietas-catalogo");
document.documentElement.lang = textos.idioma;
document.title = textos.traducir("catalogo.titulo");

const recurso = new URL("/data/demo/dietas/catalogo-rrhh.json", import.meta.url);
montarCatalogoTarifasDietas(document.querySelector("main"), {
  traducir: textos.traducir,
  localizacion: textos.localizacion,
  fuente: () => leerRecursoJSON(recurso),
});
