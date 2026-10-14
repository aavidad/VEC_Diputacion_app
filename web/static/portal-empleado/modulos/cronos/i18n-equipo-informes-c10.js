import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_EQUIPO_INFORMES_CRONOS = (await cargarTextos("cronos-equipo-informes-c10")).seccion("equipo");

export function crearTraductorEquipoInformesCronos(mensajes = MENSAJES_EQUIPO_INFORMES_CRONOS) {
  const claves = Object.keys(MENSAJES_EQUIPO_INFORMES_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) throw new TypeError("catálogo de equipo Cronos incompleto");
  return (clave, variables = {}) => {
    if (!claves.includes(clave)) throw new TypeError("clave de equipo Cronos desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
