import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261001-ct-a-i18n-v1";

const { cargarTextos, crearTextos } = await import("../../../comun/textos.js");
const textos = await cargarTextos("solicitudes");
export const MENSAJES_SOLICITUDES = textos.seccion("general");
const CLAVES = Object.freeze(Object.keys(MENSAJES_SOLICITUDES));
const OPCIONES_FECHA = Object.freeze({ day: "2-digit", month: "2-digit", year: "numeric", timeZone: ZONA_HORARIA_PORTAL });

export function crearTraductorSolicitudes(catalogo = MENSAJES_SOLICITUDES, localizacion = LOCALIZACION_ACTUAL) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo i18n de Solicitudes incompleto");
  }
  const traducciones = crearTextos({ modulo: "solicitudes", localizacion, respaldo: { general: catalogo } });
  const t = (clave, variables) => traducciones.traducir(`general.${clave}`, variables);
  return Object.assign(t, {
    numero: traducciones.numero,
    minusculas: (valor) => String(valor).toLocaleLowerCase(localizacion),
    fecha: (valor) => formatearFechaSolicitudes(valor, localizacion, t("sin_dato")),
  });
}

/** Fecha civil o instante validados y presentados con el formateo común de textos. */
export function formatearFechaSolicitudes(valor, localizacion = LOCALIZACION_ACTUAL, sinDato = MENSAJES_SOLICITUDES.sin_dato) {
  if (typeof valor !== "string" || !/^\d{4}-\d\d-\d\d(?:T.*)?$/.test(valor)) return sinDato;
  const dia = new Date(`${valor.slice(0, 10)}T12:00:00Z`);
  if (!Number.isFinite(dia.getTime()) || dia.toISOString().slice(0, 10) !== valor.slice(0, 10)) return sinDato;
  const instante = valor.length === 10 ? dia : new Date(valor);
  return Number.isFinite(instante.getTime())
    ? new Intl.DateTimeFormat(localizacion, OPCIONES_FECHA).format(instante)
    : sinDato;
}
