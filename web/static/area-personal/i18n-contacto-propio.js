import { traducir } from "./i18n.js";

export function textoContactoPropio(clave, valores = {}) {
  return traducir(`areaPersonal.contacto.${clave}`, valores);
}
