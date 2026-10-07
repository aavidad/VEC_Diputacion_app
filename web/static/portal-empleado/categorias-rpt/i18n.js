import { cargarTextos } from "../../comun/textos.js";

export const TEXTOS_CATEGORIAS = await cargarTextos("rpt-categorias-rrhh");

export function t(clave, variables = {}) {
  return TEXTOS_CATEGORIAS.traducir(`general.${clave}`, variables);
}
