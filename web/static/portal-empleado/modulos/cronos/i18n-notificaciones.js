import { MENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

/** Textos de las notificaciones de la persona a RRHH y de la bandeja de RRHH. */
export let MENSAJES_CRONOS_NOTIFICACIONES;
export function instalarMENSAJES_CRONOS_NOTIFICACIONES(mensajes) { MENSAJES_CRONOS_NOTIFICACIONES = mensajes; }

/** Traductor estricto: una clave desconocida o un catálogo incompleto fallan. */
export function crearTraductorNotificacionesCronos(mensajes) {
  const CATALOGO = Object.freeze({ ...MENSAJES_CRONOS_SOLICITUDES, ...MENSAJES_CRONOS_NOTIFICACIONES });
  const CLAVES = Object.keys(CATALOGO);
  if (!MENSAJES_CRONOS_NOTIFICACIONES || !MENSAJES_CRONOS_SOLICITUDES) throw new Error("catálogo de notificaciones Cronos sin preparar");
  const catalogo = { ...CATALOGO, ...mensajes };
  if (CLAVES.some((c) => typeof catalogo[c] !== "string" || catalogo[c] === "")) throw new Error("catálogo i18n de notificaciones de Cronos incompleto");
  const solicitudesVigentes = MENSAJES_CRONOS_SOLICITUDES;
  const notificacionesVigentes = MENSAJES_CRONOS_NOTIFICACIONES;
  return (clave, variables = {}) => {
    if (MENSAJES_CRONOS_SOLICITUDES !== solicitudesVigentes || MENSAJES_CRONOS_NOTIFICACIONES !== notificacionesVigentes) {
      throw new Error("catálogo de notificaciones Cronos sustituido");
    }
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}

/** Fecha civil visible, sin desplazarla por la zona horaria. */
export function fechaCivilVisibleCronos(fecha, locale = LOCALIZACION_ACTUAL) {
  const [a, m, d] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "medium" }).format(new Date(Date.UTC(a, m - 1, d, 12)));
}

/** Instante visible en la zona de la persona. */
export function instanteVisibleCronos(valor, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid") {
  return new Intl.DateTimeFormat(locale, { timeZone: zonaHoraria, dateStyle: "medium", timeStyle: "short" }).format(new Date(valor));
}

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
/**
 * Documento por referencia y huella. La huella se ofrece a todos (también al
 * lector de pantalla y al teclado) en un detalle desplegable, no sólo en un
 * title.
 */
export function documentoNotificacionCronos(n, t) {
  if (!n.adjunto_ref) return escaparHTML(t("sin_documento"));
  return `${escaparHTML(n.adjunto_ref)}<details class="cronos-huella"><summary>${escaparHTML(t("ver_huella"))}</summary>`
    + `<span class="cronos-huella-valor">${escaparHTML(t("huella_documento", { huella: n.adjunto_sha256 }))}</span></details>`;
}

/** Número localizado (contador de caracteres). */
export function numeroVisibleCronos(valor, locale = LOCALIZACION_ACTUAL) {
  return new Intl.NumberFormat(locale).format(valor);
}
