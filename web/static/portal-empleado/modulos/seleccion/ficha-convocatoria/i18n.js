import { cargarTextos } from "../../../../comun/textos.js";

export function cargarTextosFicha(opciones = {}) {
  return cargarTextos("seleccion-ficha-convocatoria", opciones);
}
