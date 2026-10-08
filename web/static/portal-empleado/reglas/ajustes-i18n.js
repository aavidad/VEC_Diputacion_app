import { cargarTextos, reintentarTextos } from "../../comun/textos.js";
import { IDIOMA_POR_DEFECTO } from "../../comun/idioma.js";

let catalogo = null;

export async function cargarTextosAjustes() {
  catalogo = await cargarTextos("reglas-plazos");
  return catalogo;
}

export async function reintentarTextosAjustes() {
  catalogo = await reintentarTextos("reglas-plazos");
  return catalogo;
}

export const idiomaAjustes = () => catalogo?.idioma ?? IDIOMA_POR_DEFECTO;
export const idiomaDatosAjustes = () => IDIOMA_POR_DEFECTO;
export const usaIdiomaRespaldoAjustes = () => Boolean(catalogo?.incidenciaCatalogo);
export const existeClaveAjustes = (clave) => Boolean(catalogo?.mensajes?.general && Object.hasOwn(catalogo.mensajes.general, clave));
export function traducirAjustes(clave, variables = {}) {
  if (!catalogo) throw new Error("catálogo de plazos no cargado");
  return catalogo.traducir(`general.${clave}`, variables);
}
export function numeroAjustes(valor) {
  if (!catalogo) throw new Error("catálogo de plazos no cargado");
  return catalogo.numero(valor);
}
export function fechaAjustes(valor) {
  if (!catalogo) throw new Error("catálogo de plazos no cargado");
  return catalogo.fecha(valor, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" });
}
