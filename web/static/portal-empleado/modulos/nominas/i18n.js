import { cargarTextos } from "../../../comun/textos.js";

export const TEXTOS_NOMINAS = await cargarTextos("nominas");

export function crearTraductorNominas(textos = TEXTOS_NOMINAS) {
  if (typeof textos?.traducir !== "function" || typeof textos?.fecha !== "function") {
    throw new TypeError("nominas:textos");
  }
  return (clave, valores = {}) => textos.traducir(`general.${clave}`, valores);
}
