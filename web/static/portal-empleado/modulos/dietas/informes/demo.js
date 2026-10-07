import { leerRecursoJSON } from "../../../../comun/idioma.js";
import { cargarTextos } from "../../../../comun/textos.js";
import { montarInformesDietas } from "./vista.js?v=20261004-a-dietas-situacion-v1";

const catalogo = await cargarTextos("dietas-informes");
document.documentElement.lang = catalogo.idioma;
document.title = catalogo.traducir("general.titulo");

const urlDatos = new URL("../../../../../../data/demo/dietas/informes.json", import.meta.url);
const urlConfiguracion = new URL("../../../../../../data/catalogos/dietas/informes-ejemplo-v1.json", import.meta.url);
montarInformesDietas(document.querySelector("[data-dietas-informes-demo]"), {
  cargarDatos: () => leerRecursoJSON(urlDatos),
  cargarConfiguracion: () => leerRecursoJSON(urlConfiguracion),
  traducir: (clave, variables) => catalogo.traducir(`general.${clave}`, variables),
  localizacion: catalogo.localizacion,
});
