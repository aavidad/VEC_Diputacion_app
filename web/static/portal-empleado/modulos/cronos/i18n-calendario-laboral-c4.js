import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_CALENDARIO_LABORAL_C4 = (await cargarTextos("cronos-calendario-laboral-c4")).seccion("calendario");

export function crearTraductorCalendarioLaboralC4(mensajes = MENSAJES_CALENDARIO_LABORAL_C4) {
  const claves = Object.keys(MENSAJES_CALENDARIO_LABORAL_C4);
  if (claves.some((clave) => typeof mensajes?.[clave] !== "string" || !mensajes[clave].trim())) {
    throw new TypeError("catálogo de calendario laboral incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de calendario laboral desconocida");
    return mensajes[clave];
  };
}
