import { crearTraductorCronos, MENSAJES_CRONOS } from "./i18n.js?v=20260929-i18n-textos-v1";
import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_CRONOS_C4 = (await cargarTextos("cronos")).seccion("c4");

const CLAVES_C4 = Object.freeze(Object.keys(MENSAJES_CRONOS_C4));

export function crearTraductorCronosC4(mensajes = MENSAJES_CRONOS_C4) {
  if (!mensajes || typeof mensajes !== "object" || CLAVES_C4.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new Error("catálogo i18n C4 de Cronos incompleto");
  }
  const general = crearTraductorCronos(MENSAJES_CRONOS);
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_CRONOS_C4, clave)) return general(clave, variables);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
