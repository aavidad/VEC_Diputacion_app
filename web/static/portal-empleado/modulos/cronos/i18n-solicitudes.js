import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

/** Textos de movimientos (calendario, ausencias y olvidos) y de permisos propios. */
export const MENSAJES_CRONOS_SOLICITUDES = (await cargarTextos("cronos")).seccion("solicitudes");

const CLAVES = Object.freeze(Object.keys(MENSAJES_CRONOS_SOLICITUDES));

/** Traductor estricto: una clave desconocida o un catálogo incompleto fallan. */
export function crearTraductorSolicitudesCronos(mensajes = MENSAJES_CRONOS_SOLICITUDES) {
  const catalogo = { ...MENSAJES_CRONOS_SOLICITUDES, ...mensajes };
  if (CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo i18n de solicitudes de Cronos incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}

/** Cantidad de un permiso: días enteros o minutos presentados en horas y minutos. */
export function formatearCantidadCronos(cantidad, unidad, t, locale = LOCALIZACION_ACTUAL) {
  if (!Number.isSafeInteger(cantidad) || cantidad < 0) return t("sin_limite");
  const numero = new Intl.NumberFormat(locale);
  if (unidad === "dia") {
    const regla = new Intl.PluralRules(locale).select(cantidad);
    return t(regla === "one" ? "dias_uno" : "dias_otros", { n: numero.format(cantidad) });
  }
  const h = Math.floor(cantidad / 60); const m = cantidad % 60;
  if (h === 0) return t("minutos", { m: numero.format(m) });
  if (m === 0) return t("horas", { h: numero.format(h) });
  return t("horas_minutos", { h: numero.format(h), m: numero.format(m) });
}
