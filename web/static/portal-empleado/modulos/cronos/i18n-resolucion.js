import { MENSAJES_CRONOS_SOLICITUDES_ES } from "./i18n-solicitudes.js";

/** Textos de la resolución de permisos (jefatura y RRHH) y de los avisos propios. */
export const MENSAJES_CRONOS_RESOLUCION_ES = Object.freeze({
  bandeja_titulo: "Solicitudes por resolver",
  paso_selector: "Resolver como",
  paso_responsable: "Jefatura",
  paso_administracion: "RRHH",
  col_persona: "Persona",
  col_solicitada: "Solicitada",
  persona_sin_nombre: "Sin nombre publicado",
  requiere_justificante: "Requiere justificante",
  resolver: "Resolver",
  resolver_solicitud: "Resolver {permiso} de {persona}",
  decision: "Decisión",
  decision_aprobar_responsable: "Dar conformidad",
  decision_aprobar_administracion: "Conceder",
  decision_denegar: "Denegar",
  motivo: "Motivo",
  resolucion_enviar: "Registrar resolución",
  resolucion_enviando: "Registrando…",
  resolucion_pendiente_administracion: "Conformidad registrada. La solicitud pasa a RRHH.",
  resolucion_concedido: "Permiso concedido. La persona tiene el aviso en Cronos.",
  resolucion_denegado: "Permiso denegado. La persona tiene el aviso en Cronos.",
  resolucion_ya_registrada: "Esta resolución ya estaba registrada.",
  bandeja_vacia: "No hay solicitudes pendientes de resolver.",
  error_motivo: "Indique el motivo de la denegación.",
  error_no_competente: "No le corresponde resolver esta solicitud.",
  error_estado_cambiado: "La solicitud ha cambiado desde que la abrió. La bandeja se ha actualizado.",
  error_conflicto_resolucion: "Ya existe otra resolución con esa referencia. Vuelva a abrir la solicitud.",
  error_resolucion: "No se pudo registrar la resolución. Puede reintentarla sin duplicarla.",
  error_peticion_resolucion: "No se pudo registrar: revise la decisión y el motivo.",
  estado_pendiente_asignacion: "Pendiente de asignar jefatura",
  sin_jefatura_asignada: "Sin jefatura asignada: no se puede resolver hasta asignarla",
  error_pendiente_asignacion: "Esta persona no tiene jefatura asignada. La solicitud no se puede resolver hasta que se asigne.",

  avisos_titulo: "Avisos de resolución",
  avisos_recibidos: "Recibidos",
  avisos_archivados: "Archivados",
  avisos_vacio: "No hay avisos.",
  aviso_concedido: "Se concede el permiso {permiso}: {periodo} ({cantidad}).",
  aviso_denegado: "Se deniega el permiso {permiso}: {periodo} ({cantidad}).",
  aviso_motivo: "Motivo: {motivo}",
  aviso_resuelto: "Resuelto el {fecha}",
  aviso_archivado_el: "Archivado el {fecha}",
  archivar: "Archivar",
  archivar_aviso: "Archivar el aviso de {permiso}",
  aviso_archivado: "Aviso archivado.",
  aviso_ya_archivado: "Este aviso ya estaba archivado.",
  error_archivar: "No se pudo archivar el aviso. Puede reintentarlo.",
});

const CATALOGO = Object.freeze({ ...MENSAJES_CRONOS_SOLICITUDES_ES, ...MENSAJES_CRONOS_RESOLUCION_ES });
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
export function periodoSolicitudCronos(s, t, locale = "es-ES") {
  return s.hora_inicio ? t("periodo_horas", { fecha: fechaVisible(s.desde, locale), inicio: s.hora_inicio, fin: s.hora_fin })
    : t("periodo_dias", { desde: fechaVisible(s.desde, locale), hasta: fechaVisible(s.hasta, locale) });
}
