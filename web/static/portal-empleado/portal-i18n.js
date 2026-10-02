import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../comun/idioma.js";
import { cargarTextos } from "../comun/textos.js";
import { mensajesTramitePortal } from "./modulos/contratacion-temporal/i18n-fases-rrhh.js?v=20261001-ct-a-i18n-v1";

/**
 * Textos del shell del portal: viven en `textos/<idioma>/portal.json`
 * (secciones `general`, `textos` y `panel_interno`), `portal-ayuda.json` y la
 * sección `portal` de `preferencias.json`, más las secciones `plazos` y
 * `politica_cese` de `bolsa.json` (políticas de ofertas y de cese de Bolsa).
 * Se combinan en un único catálogo plano que consulta `traducirPortal`; si al
 * idioma actual le falta una clave se usa la del idioma por defecto.
 */

/** Aplana una sección anidada a claves `a.b.c`, como las usa el portal. */
function aplanar(seccion, prefijo = "", salida = {}) {
  for (const [clave, valor] of Object.entries(seccion)) {
    const ruta = prefijo ? `${prefijo}.${clave}` : clave;
    if (typeof valor === "string") salida[ruta] = valor;
    else aplanar(valor, ruta, salida);
  }
  return salida;
}

/** Catálogo común del shell en `idioma` (por defecto, el de la interfaz). */
export async function cargarMensajesPortal(idioma = IDIOMA_ACTUAL) {
  const [portal, ayuda, preferencias, bolsa] = await Promise.all(["portal", "portal-ayuda", "preferencias", "bolsa"]
    .map((modulo) => cargarTextos(modulo, { idioma })));
  return Object.freeze({
    ...ayuda.seccion("ayuda"),
    ...portal.seccion("panel_interno"),
    ...portal.seccion("textos"),
    ...bolsa.seccion("plazos"),
    ...bolsa.seccion("politica_cese"),
    ...mensajesTramitePortal(idioma),
    ...portal.seccion("general"),
    ...aplanar(preferencias.seccion("portal")),
  });
}

/** Catálogo común de los estados del shell y del acceso a Borradores, en el idioma de la interfaz. */
export const MENSAJES_PORTAL = await cargarMensajesPortal();

const CLAVES = Object.freeze(Object.keys(MENSAJES_PORTAL));

export function crearTraductorPortal(catalogo = MENSAJES_PORTAL) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n del Portal del Empleado incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n del portal desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g,
      (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirPortal = crearTraductorPortal();

/** Texto del catálogo común ya escapado para insertarlo en una plantilla HTML. */
export function textoPortal(clave, variables = {}) {
  return traducirPortal(clave, variables).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

/** Textos comunes de las vistas internas de Bolsa (sección `bolsa_interna` de `portal.json`). */
export const MENSAJES_BOLSA_INTERNA = (await cargarTextos("portal")).seccion("bolsa_interna");

const CLAVES_BOLSA_INTERNA = Object.freeze(Object.keys(MENSAJES_BOLSA_INTERNA));
export function crearTraductorBolsaInterna(catalogo = MENSAJES_BOLSA_INTERNA) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES_BOLSA_INTERNA.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Bolsa interna incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES_BOLSA_INTERNA.includes(clave)) throw new Error(`clave i18n de Bolsa interna desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
export const traducirBolsaInterna = crearTraductorBolsaInterna();
/** Localización y zona horaria del portal: autoridad común para formatear fechas, horas, importes y cifras. */
export const LOCALIZACION_PORTAL = LOCALIZACION_ACTUAL;
export const ZONA_HORARIA_PORTAL = "Europe/Madrid";
export function formatearNumeroPortal(valor, opciones = {}) {
  const numero = Number(valor);
  return Number.isFinite(numero) ? new Intl.NumberFormat(LOCALIZACION_PORTAL, opciones).format(numero) : String(valor ?? "");
}
export function formatearFechaPortal(valor) {
  if (valor === undefined || valor === null || valor === "") return traducirBolsaInterna("fecha_sin_valor");
  const texto = String(valor).trim();
  const local = texto.match(/^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}))?$/) || texto.match(/^(\d{2})\/(\d{2})\/(\d{4})(?:\s+(\d{2}):(\d{2}))?$/);
  if (!local) return texto;
  const iso = texto.includes("-");
  const [dia, mes, ano, hora, minuto] = iso ? [local[3], local[2], local[1], local[4], local[5]] : [local[1], local[2], local[3], local[4], local[5]];
  const fecha = new Date(Number(ano), Number(mes) - 1, Number(dia), Number(hora || 0), Number(minuto || 0));
  if (!Number.isFinite(fecha.getTime())) return texto;
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, hora ? { dateStyle: "short", timeStyle: "short" } : { dateStyle: "short" }).format(fecha);
}

// Fin de vigencia de una bolsa, igual en la portada y en el cuadro de bolsas:
// solo la fecha (sin hora) o «Sin fecha de fin» cuando no consta el fin.
export function finVigenciaBolsaPortal(valor, traducir = traducirPortal) {
  const texto = typeof valor === "string" ? valor.trim() : "";
  const civil = texto.match(/^\d{4}-\d{2}-\d{2}$/);
  const fecha = new Date(civil ? `${texto}T00:00:00Z` : texto);
  if (texto === "" || !Number.isFinite(fecha.getTime())) return traducir("inicio_rrhh_sin_fin");
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, {
    dateStyle: "short", timeZone: civil ? "UTC" : ZONA_HORARIA_PORTAL,
  }).format(fecha);
}
