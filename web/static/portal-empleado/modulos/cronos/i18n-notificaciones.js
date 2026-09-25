import { MENSAJES_CRONOS_SOLICITUDES_ES } from "./i18n-solicitudes.js";

/** Textos de las notificaciones de la persona a RRHH y de la bandeja de RRHH. */
export const MENSAJES_CRONOS_NOTIFICACIONES_ES = Object.freeze({
  notificaciones_titulo: "Notificaciones a RRHH",
  nueva_titulo: "Nueva notificación",
  campo_tipo: "Tipo",
  campo_tipo_elegir: "Elija el tipo",
  campo_fecha: "Fecha a la que se refiere",
  campo_texto: "Mensaje",
  campo_texto_cuenta: "{usados} de {maximo} caracteres",
  campo_documento_ref: "Referencia del documento",
  campo_documento: "Documento",
  documento_calculando: "Calculando la huella del documento…",
  documento_listo: "Huella del documento calculada.",
  documento_error: "No se pudo leer el documento. Elija otro de hasta 10 MB.",
  enviar: "Enviar a RRHH",
  enviando: "Enviando…",
  enviada: "Notificación enviada a RRHH. Queda registrada con su recibo.",
  ya_enviada: "Esta notificación ya estaba registrada.",
  sin_tipos: "No hay tipos de notificación disponibles.",
  error_datos: "Revise el tipo, la fecha y el mensaje.",
  error_documento: "Indique la referencia y elija el documento, o deje ambos vacíos.",
  error_tipo_no_vigente: "Ese tipo ya no está disponible. La lista se ha actualizado.",
  error_conflicto_notificacion: "Ya existe otra notificación con esa referencia. Vuelva a enviarla.",
  error_envio: "No se pudo enviar la notificación. Puede reintentarlo sin duplicarla.",
  mis_notificaciones: "Mis notificaciones",
  sin_notificaciones: "No ha enviado notificaciones.",
  col_enviada: "Enviada",
  col_tipo: "Tipo",
  col_fecha: "Se refiere al",
  col_mensaje: "Mensaje",
  col_documento: "Documento",
  col_estado: "Estado",
  col_persona: "Persona",
  col_accion: "Acción",
  sin_documento: "—",
  huella_documento: "Huella SHA-256: {huella}",
  estado_registrada: "Pendiente de atender",
  estado_atendida: "Atendida el {fecha}",

  bandeja_notificaciones_titulo: "Notificaciones recibidas",
  filtro_pendientes: "Pendientes",
  filtro_atendidas: "Atendidas",
  persona_sin_nombre: "Sin nombre publicado",
  atender: "Marcar atendida",
  atender_notificacion: "Marcar atendida la notificación de {persona}",
  atendiendo: "Registrando…",
  atendida: "Notificación marcada como atendida.",
  ya_atendida: "Esta notificación ya estaba atendida.",
  sin_pendientes: "No hay notificaciones pendientes de atender.",
  sin_atendidas: "No hay notificaciones atendidas.",
  error_no_competente_notificacion: "No le corresponde atender esta notificación.",
  error_atender: "No se pudo marcar como atendida. Puede reintentarlo.",
});

const CATALOGO = Object.freeze({ ...MENSAJES_CRONOS_SOLICITUDES_ES, ...MENSAJES_CRONOS_NOTIFICACIONES_ES });
const CLAVES = Object.freeze(Object.keys(CATALOGO));

/** Traductor estricto: una clave desconocida o un catálogo incompleto fallan. */
export function crearTraductorNotificacionesCronos(mensajes = CATALOGO) {
  const catalogo = { ...CATALOGO, ...mensajes };
  if (CLAVES.some((c) => typeof catalogo[c] !== "string" || catalogo[c] === "")) throw new Error("catálogo i18n de notificaciones de Cronos incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}

/** Fecha civil visible, sin desplazarla por la zona horaria. */
export function fechaCivilVisibleCronos(fecha, locale = "es-ES") {
  const [a, m, d] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "medium" }).format(new Date(Date.UTC(a, m - 1, d, 12)));
}

/** Instante visible en la zona de la persona. */
export function instanteVisibleCronos(valor, locale = "es-ES", zonaHoraria = "Europe/Madrid") {
  return new Intl.DateTimeFormat(locale, { timeZone: zonaHoraria, dateStyle: "medium", timeStyle: "short" }).format(new Date(valor));
}

/** Número localizado (contador de caracteres). */
export function numeroVisibleCronos(valor, locale = "es-ES") {
  return new Intl.NumberFormat(locale).format(valor);
}
