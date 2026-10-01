import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_CONSULTA_CRONOS = (await cargarTextos("cronos-consulta")).seccion("consulta");

export function crearTraductorConsultaCronos(mensajes = MENSAJES_CONSULTA_CRONOS) {
  const claves = Object.keys(MENSAJES_CONSULTA_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de consulta Cronos incompleto");
  }
  return (clave, variables = {}) => {
    if (!claves.includes(clave)) throw new TypeError("clave de consulta Cronos desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
