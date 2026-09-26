/**
 * Textos con los que Bolsa presenta filas cuyo único identificador es una
 * referencia interna: número de orden, datos legibles y botón «Copiar
 * referencia». Los textos del justificante son los comunes del panel.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260926-pulido-portal-v1";

export const MENSAJES_REFERENCIAS_ES = Object.freeze({
  persona_fuera_de_pagina: "Persona seleccionada fuera de esta página",
  borrador_justificante: "Justificante",
  borrador_version: "Versión {version}",
  borrador_referencia_copiar_aria: "Copiar la referencia del borrador",
  borrador_justificante_copiar_aria: "Copiar la referencia del justificante del guardado",
  borrador_configuracion_copiar_aria: "Copiar la referencia de {nombre}",
  registros: "{cantidad} registros",
  sin_fecha_limite: "Sin fecha límite",
  col_numero: "Nº",
  col_referencia: "Referencia",
  convocatorias_titulo: "Convocatorias del ámbito autorizado",
  convocatorias_caption: "Convocatorias del ámbito autorizado, numeradas por orden de llegada",
  convocatorias_vacio: "La fuente autorizada no ha devuelto convocatorias para este ámbito.",
  convocatoria_col_categoria: "Bolsa / Categoría",
  convocatoria_col_estado: "Estado",
  convocatoria_col_cierre: "Cierre de plazo",
  convocatoria_col_solicitudes: "Solicitudes",
  convocatoria_col_pendientes: "Pendientes",
  convocatoria_copiar_aria: "Copiar la referencia de la convocatoria {numero}, {categoria}",
  actuaciones_titulo: "Actuaciones pendientes",
  actuaciones_caption: "Trabajo administrativo pendiente sin identidad de personas interesadas",
  actuaciones_vacio: "La fuente autorizada no ha devuelto actuaciones pendientes para este ámbito.",
  actuacion_col_tipo: "Actuación",
  actuacion_col_estado: "Estado",
  actuacion_col_prioridad: "Prioridad",
  actuacion_col_fecha_limite: "Fecha límite",
  actuacion_col_elementos: "Elementos",
  actuacion_copiar_aria: "Copiar la referencia de la actuación {numero}, {tipo}",
  actuacion_tipo_revisar_bases: "Revisar bases",
  actuacion_tipo_firmar_documento: "Firmar documento",
  actuacion_tipo_resolver_incidencia: "Resolver incidencia",
  actuacion_tipo_revisar_solicitudes: "Revisar solicitudes",
  actuacion_tipo_llamamiento: "Llamamiento",
  revision_fuente_copiar_aria: "Copiar la referencia de la revisión de la fuente",
  aviso_solicitud_valida: "Se valida con la operación de pausa o reactivación citando la referencia de la solicitud.",
  aviso_solicitud_copiar_aria: "Copiar la referencia de la solicitud",
  aviso_causa: "Causa: {causa}.",
  aviso_justificante_copiar_aria: "Copiar la referencia del justificante de la respuesta",
  actor_rrhh: "Personal de RRHH",
  actor_copiar_aria: "Copiar la referencia de quien registró la actuación",
});

export function tieneTextoReferencia(clave) {
  return Object.hasOwn(MENSAJES_REFERENCIAS_ES, clave);
}

/** Traductor estricto; los textos del botón de copia son los del justificante común. */
export function traducirReferencia(clave, variables = {}) {
  if (clave.startsWith("justificante_")) return traducirPortal(`panel_${clave}`, variables);
  const plantilla = MENSAJES_REFERENCIAS_ES[clave];
  if (typeof plantilla !== "string") throw new Error(`clave i18n de referencias desconocida: ${clave}`);
  return plantilla.replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
}
