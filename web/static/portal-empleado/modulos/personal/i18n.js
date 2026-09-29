import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

// `portal.js` importa este fichero de forma estática (fuera del grafo
// perezoso de módulos): la carga de `cargarTextos` se hace con `import()`
// para no incorporar `comun/textos.js` a su precarga estática, que es de
// `index.html` y no se toca en esta migración.
const { cargarTextos } = await import("../../../comun/textos.js");

/** Textos propios de la presentación DEMO de Personal. */
export const MENSAJES_PERSONAL = (await cargarTextos("personal")).seccion("general");

const CLAVES = Object.freeze(Object.keys(MENSAJES_PERSONAL));

export function crearTraductorPersonal(catalogo = MENSAJES_PERSONAL) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Personal incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Personal desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export function formatearRecuentoCategorias(total, locale = LOCALIZACION_ACTUAL, catalogo = MENSAJES_PERSONAL) {
  if (!Number.isSafeInteger(total) || total < 0 || typeof locale !== "string" || locale === "") {
    throw new TypeError("recuento de categorías no válido");
  }
  const t = crearTraductorPersonal(catalogo);
  const numero = new Intl.NumberFormat(locale).format(total);
  return t(total === 1 ? "catalogo_recuento_uno" : "catalogo_recuento_otro", { total: numero });
}

export function formatearRecuentoRPT(total, locale = LOCALIZACION_ACTUAL, catalogo = MENSAJES_PERSONAL) {
  if (!Number.isSafeInteger(total) || total < 0 || typeof locale !== "string" || locale === "") throw new TypeError("recuento RPT no válido");
  const t = crearTraductorPersonal(catalogo);
  const numero = new Intl.NumberFormat(locale).format(total);
  return t(total === 1 ? "rpt_recuento_uno" : "rpt_recuento_otro", { total: numero });
}
export function formatearRecuentoEstructura(total, catalogo = MENSAJES_PERSONAL) { if (total !== 66) throw new TypeError("recuento de estructura organizativa no válido"); return crearTraductorPersonal(catalogo)("estructura_recuento"); }

// Conserva el ISO en el contrato y lo traduce únicamente al pintar una fecha
// inequívoca para la zona operativa de la presentación.
export function formatearFechaEstructuraOrganizativa(iso) {
  if (typeof iso !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/u.test(iso)) throw new TypeError("fecha de estructura organizativa no válida");
  const fecha = new Date(iso);
  if (!Number.isFinite(fecha.getTime()) || fecha.toISOString() !== `${iso.slice(0, -1)}.000Z`) throw new TypeError("fecha de estructura organizativa no válida");
  return `${new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha)} (Europe/Madrid)`;
}
