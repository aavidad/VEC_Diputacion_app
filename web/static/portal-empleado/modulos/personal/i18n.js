/** Textos propios de la presentación DEMO de Personal. */
export let MENSAJES_PERSONAL;
let CLAVES;
let localizacionPersonal;
let preparacion = 0;

// El shell importa este módulo antes de abrir Personal. La lectura del catálogo
// comienza al preparar la vista, sin bloquear el grafo estático del portal.
export async function prepararTextosPersonal(opciones = {}) {
  const turno = ++preparacion;
  MENSAJES_PERSONAL = undefined;
  CLAVES = undefined;
  localizacionPersonal = undefined;
  const { cargarTextos } = await import("../../../comun/textos.js");
  const textos = await cargarTextos("personal", opciones);
  const mensajes = textos.seccion("general");
  if (turno === preparacion) {
    MENSAJES_PERSONAL = mensajes;
    CLAVES = Object.freeze(Object.keys(mensajes));
    localizacionPersonal = textos.localizacion;
  }
  return Object.freeze({
    idioma: textos.idioma,
    localizacion: textos.localizacion,
    incidenciaCatalogo: textos.incidenciaCatalogo,
    incidenciaIndice: textos.incidenciaIndice,
  });
}

function catalogoPreparado() {
  if (!MENSAJES_PERSONAL || !CLAVES) throw new Error("textos de Personal pendientes de preparación");
  return MENSAJES_PERSONAL;
}

export function crearTraductorPersonal(catalogo = catalogoPreparado()) {
  const claves = CLAVES ?? Object.freeze(Object.keys(catalogo ?? {}));
  if (!catalogo || typeof catalogo !== "object"
    || claves.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Personal incompleto");
  }
  return (clave, variables = {}) => {
    if (!claves.includes(clave)) throw new Error(`clave i18n de Personal desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export function formatearRecuentoCategorias(total, locale = localizacionPersonal, catalogo = catalogoPreparado()) {
  if (!Number.isSafeInteger(total) || total < 0 || typeof locale !== "string" || locale === "") {
    throw new TypeError("recuento de categorías no válido");
  }
  const t = crearTraductorPersonal(catalogo);
  const numero = new Intl.NumberFormat(locale).format(total);
  return t(total === 1 ? "catalogo_recuento_uno" : "catalogo_recuento_otro", { total: numero });
}

export function formatearRecuentoRPT(total, locale = localizacionPersonal, catalogo = catalogoPreparado()) {
  if (!Number.isSafeInteger(total) || total < 0 || typeof locale !== "string" || locale === "") throw new TypeError("recuento RPT no válido");
  const t = crearTraductorPersonal(catalogo);
  const numero = new Intl.NumberFormat(locale).format(total);
  return t(total === 1 ? "rpt_recuento_uno" : "rpt_recuento_otro", { total: numero });
}
export function formatearRecuentoEstructura(total, catalogo = catalogoPreparado()) { if (total !== 66) throw new TypeError("recuento de estructura organizativa no válido"); return crearTraductorPersonal(catalogo)("estructura_recuento"); }

// Conserva el ISO en el contrato y lo traduce únicamente al pintar una fecha
// inequívoca para la zona operativa de la presentación.
export function formatearFechaEstructuraOrganizativa(iso) {
  catalogoPreparado();
  if (typeof iso !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/u.test(iso)) throw new TypeError("fecha de estructura organizativa no válida");
  const fecha = new Date(iso);
  if (!Number.isFinite(fecha.getTime()) || fecha.toISOString() !== `${iso.slice(0, -1)}.000Z`) throw new TypeError("fecha de estructura organizativa no válida");
  return `${new Intl.DateTimeFormat(localizacionPersonal, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha)} (Europe/Madrid)`;
}
