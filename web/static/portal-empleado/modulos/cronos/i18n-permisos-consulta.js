import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_CONSULTA_PERMISOS_CRONOS = (await cargarTextos("cronos-permisos-consulta")).seccion("consulta");

export function crearTraductorConsultaPermisosCronos(mensajes = MENSAJES_CONSULTA_PERMISOS_CRONOS) {
  const claves = Object.keys(MENSAJES_CONSULTA_PERMISOS_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de consulta de permisos Cronos incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de consulta de permisos Cronos desconocida");
    return mensajes[clave];
  };
}
