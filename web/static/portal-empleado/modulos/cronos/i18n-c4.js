import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260925-aspecto-v1";

export const MENSAJES_CRONOS_C4_ES = Object.freeze({
  calendario_civil_titulo: "Calendario civil",
  calendario_anterior: "Año anterior",
  calendario_siguiente: "Año siguiente",
  calendario_mes: "{mes} de {anio}",
  calendario_dia: "{fecha}",
  calendario_seleccion: "Fecha seleccionada",
  calendario_estado: "Calendario laboral no configurado",
  calendario_estado_detalle: "Ausencias y calendario laboral no disponibles.",
});

const CLAVES_C4 = Object.freeze(Object.keys(MENSAJES_CRONOS_C4_ES));

export function crearTraductorCronosC4(mensajes = MENSAJES_CRONOS_C4_ES) {
  if (!mensajes || typeof mensajes !== "object" || CLAVES_C4.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new Error("catálogo i18n C4 de Cronos incompleto");
  }
  const general = crearTraductorCronos(MENSAJES_CRONOS_ES);
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_CRONOS_C4_ES, clave)) return general(clave, variables);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
