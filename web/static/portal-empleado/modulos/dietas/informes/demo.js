import { leerRecursoJSON } from "../../../../comun/idioma.js";
import { cargarTextos } from "../../../../comun/textos.js";
import { montarInformesDietas } from "./vista.js";

const catalogo = await cargarTextos("dietas-informes");
document.documentElement.lang = catalogo.idioma;
document.title = catalogo.traducir("general.titulo");

const urlDatos = new URL("../../../../../../data/demo/dietas/informes.json", import.meta.url);
montarInformesDietas(document.querySelector("[data-dietas-informes-demo]"), {
  cargarDatos: () => leerRecursoJSON(urlDatos),
  traducir: (clave, variables) => catalogo.traducir(`general.${clave}`, variables),
  localizacion: catalogo.localizacion,
});
