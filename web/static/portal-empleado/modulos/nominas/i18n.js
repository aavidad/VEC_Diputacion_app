import { cargarTextos } from "../../../comun/textos.js";

/** Solo se piden los textos cuando se abre la vista de Nóminas. */
export function cargarTextosNominas(cargar = cargarTextos) {
  return cargar("nominas", { soloIdiomaActivo: true });
}

export function crearTraductorNominas(textos) {
  if (typeof textos?.traducir !== "function" || typeof textos?.fecha !== "function") {
    throw new TypeError("nominas:textos");
  }
  return (clave, valores = {}) => textos.traducir(`general.${clave}`, valores);
}
