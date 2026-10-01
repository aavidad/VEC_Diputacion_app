import { cargarTextos } from "../../../../comun/textos.js";
import { leerRecursoJSON } from "../../../../comun/idioma.js";
import { montarCatalogoTarifasDietas } from "./vista.js?v=20261001-dietas-catalogo-propuestas-v1";

// Previsualización aislada: el servidor local debe exponer la raíz del repositorio.
const textos = await cargarTextos("dietas-catalogo");
document.documentElement.lang = textos.idioma;
document.title = textos.traducir("catalogo.titulo");

const recurso = new URL("../../../../../../data/demo/dietas/catalogo-rrhh.json", import.meta.url);
montarCatalogoTarifasDietas(document.querySelector("main"), {
  traducir: textos.traducir,
  localizacion: textos.localizacion,
  fuente: () => leerRecursoJSON(recurso),
});
