import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260924-cronos-integrado-v1";

export const MENSAJES_CRONOS_C4_ES = Object.freeze({
  calendario_civil_titulo: "Calendario civil",
  calendario_civil_descripcion: "Seleccione una fecha para consultarla. Esta cuadrícula solo muestra días naturales.",
  calendario_anterior: "Año anterior",
  calendario_siguiente: "Año siguiente",
  calendario_mes: "{mes} de {anio}",
  calendario_dia: "{fecha}",
  calendario_seleccion: "Fecha seleccionada",
  calendario_natural: "Día natural",
  calendario_fin_semana: "Sábado o domingo (dato civil)",
  calendario_leyenda: "Leyenda",
  calendario_leyenda_dia: "Fecha civil seleccionable",
  calendario_leyenda_fin_semana: "Sábado y domingo: clasificación civil, sin efecto laboral inferido",
  calendario_estado: "Calendario laboral no configurado",
  calendario_estado_detalle: "No constan aquí festivos, apertura del centro, jornada individual ni versiones oficiales. La fecha elegida no determina si es hábil o laborable.",
  calendario_fuente: "Pendiente de la fuente autorizada de Calendarios para la organización, centro y periodo.",
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
