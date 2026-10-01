import { cargarTextos } from "../../../comun/textos.js";

/** Textos del recorrido sin fuente de permisos. No habilita operaciones. */
export const MENSAJES_CRONOS_PERMISOS = (await cargarTextos("cronos")).seccion("permisos");

/** Consulta de pendientes; este catálogo no habilita el registro documental. */
export const MENSAJES_JUSTIFICACION_CRONOS = (await cargarTextos("cronos-permisos")).seccion("justificacion");

export function crearTraductorJustificacionCronos(mensajes = MENSAJES_JUSTIFICACION_CRONOS) {
  const claves = Object.keys(MENSAJES_JUSTIFICACION_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de justificación Cronos incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de justificación Cronos desconocida");
    return mensajes[clave];
  };
}
