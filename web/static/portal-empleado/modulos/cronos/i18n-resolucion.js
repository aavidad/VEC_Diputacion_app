import { MENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";
import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

/** Textos de la resolución de permisos (jefatura y RRHH) y de los avisos propios. */
export const MENSAJES_CRONOS_RESOLUCION = (await cargarTextos("cronos")).seccion("resolucion");

const MENSAJES_BANDEJA = (await cargarTextos("cronos-resolucion")).seccion("bandeja");
const CATALOGO = Object.freeze({ ...MENSAJES_CRONOS_SOLICITUDES, ...MENSAJES_CRONOS_RESOLUCION, ...MENSAJES_BANDEJA });
const CLAVES = Object.freeze(Object.keys(CATALOGO));

/**
 * Traductor estricto de estas vistas: incluye los textos comunes de las
 * solicitudes (cantidades, periodos, estados). Una clave desconocida o un
 * catálogo incompleto fallan.
 */
export function crearTraductorResolucionCronos(mensajes = CATALOGO) {
  const catalogo = { ...CATALOGO, ...mensajes };
  if (CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo i18n de resolución de Cronos incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}

function fechaVisible(fecha, locale) {
  const [a, m, d] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "medium" }).format(new Date(Date.UTC(a, m - 1, d, 12)));
}

/** Periodo de una solicitud: días o tramo horario de un día. */
export function periodoSolicitudCronos(s, t, locale = LOCALIZACION_ACTUAL) {
  return s.hora_inicio ? t("periodo_horas", { fecha: fechaVisible(s.desde, locale), inicio: s.hora_inicio, fin: s.hora_fin })
    : t("periodo_dias", { desde: fechaVisible(s.desde, locale), hasta: fechaVisible(s.hasta, locale) });
}
