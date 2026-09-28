/**
 * Textos con los que Bolsa presenta filas cuyo único identificador es una
 * referencia interna: número de orden, datos legibles y botón «Copiar
 * referencia». Los textos del justificante son los comunes del panel.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260928-rrhh-i18n-unificada-v1";
import { IDIOMA_ACTUAL } from "../comun/idioma.js";

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

export const MENSAJES_REFERENCIAS_EN = Object.freeze({
  persona_fuera_de_pagina: "Selected person outside this page",
  borrador_justificante: "Receipt",
  borrador_version: "Version {version}",
  borrador_referencia_copiar_aria: "Copy the draft reference",
  borrador_justificante_copiar_aria: "Copy the save receipt reference",
  borrador_configuracion_copiar_aria: "Copy the reference for {nombre}",
  registros: "{cantidad} records",
  sin_fecha_limite: "No deadline",
  col_numero: "No.",
  col_referencia: "Reference",
  convocatorias_titulo: "Calls for applications in the authorised scope",
  convocatorias_caption: "Calls for applications in the authorised scope, numbered by arrival order",
  convocatorias_vacio: "The authorised source returned no calls for applications in this scope.",
  convocatoria_col_categoria: "Pool / Category",
  convocatoria_col_estado: "Status",
  convocatoria_col_cierre: "Closing date",
  convocatoria_col_solicitudes: "Applications",
  convocatoria_col_pendientes: "Pending",
  convocatoria_copiar_aria: "Copy the reference for call {numero}, {categoria}",
  actuaciones_titulo: "Pending actions",
  actuaciones_caption: "Pending administrative work without the identities of the people concerned",
  actuaciones_vacio: "The authorised source returned no pending actions in this scope.",
  actuacion_col_tipo: "Action",
  actuacion_col_estado: "Status",
  actuacion_col_prioridad: "Priority",
  actuacion_col_fecha_limite: "Deadline",
  actuacion_col_elementos: "Items",
  actuacion_copiar_aria: "Copy the reference for action {numero}, {tipo}",
  actuacion_tipo_revisar_bases: "Review terms",
  actuacion_tipo_firmar_documento: "Sign document",
  actuacion_tipo_resolver_incidencia: "Resolve issue",
  actuacion_tipo_revisar_solicitudes: "Review applications",
  actuacion_tipo_llamamiento: "Call-up",
  revision_fuente_copiar_aria: "Copy the source revision reference",
  aviso_solicitud_valida: "Validate it with the pause or reactivation operation, citing the application reference.",
  aviso_solicitud_copiar_aria: "Copy the application reference",
  aviso_causa: "Reason: {causa}.",
  aviso_justificante_copiar_aria: "Copy the response receipt reference",
  actor_rrhh: "HR staff",
  actor_copiar_aria: "Copy the reference of the person who recorded the action",
});

export function tieneTextoReferencia(clave) {
  return Object.hasOwn(MENSAJES_REFERENCIAS_ES, clave);
}

/** Traductor estricto; los textos del botón de copia son los del justificante común. */
export function traducirReferencia(clave, variables = {}) {
  if (clave.startsWith("justificante_")) return traducirPortal(`panel_${clave}`, variables);
  const plantilla = (IDIOMA_ACTUAL === "en" ? MENSAJES_REFERENCIAS_EN : MENSAJES_REFERENCIAS_ES)[clave];
  if (typeof plantilla !== "string") throw new Error(`clave i18n de referencias desconocida: ${clave}`);
  return plantilla.replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
}
