/**
 * Catálogo único de rótulos de las peticiones de personal temporal: las ocho
 * fases del procedimiento de RRHH y los estados. Portada, lista y ficha lo usan todas, para que un mismo
 * expediente se nombre igual en cualquier pantalla («Fase 5 de 8: Obtención
 * del candidato», «En trámite»).
 *
 * Solo presenta: no decide el flujo ni concede nada. Las equivalencias entre
 * la fase administrativa que guarda el servidor y la fase de RRHH son las del
 * manifiesto config/contratacion_temporal_flujo_visual_rrhh_v1.json. No
 * deduce responsables ni tareas: eso solo lo dice el servidor.
 */
import { cargarTextos } from "../../../comun/textos.js";
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";

export const FASES_RRHH = Object.freeze([
  "solicitud", "analisis_rrhh", "gestion_bolsa", "fiscalizacion",
  "obtencion_candidato", "nombramiento", "incorporacion", "seguimiento",
]);

// Fase administrativa del servidor → fase del procedimiento de RRHH.
export const FASE_RRHH_DE_ORIGEN = Object.freeze({
  solicitud: "solicitud", solicitud_registrada: "solicitud",
  analisis: "analisis_rrhh", analisis_rrhh: "analisis_rrhh",
  cobertura: "gestion_bolsa", asignacion: "gestion_bolsa", asignacion_unidad: "gestion_bolsa",
  informe: "gestion_bolsa", informe_juridico: "gestion_bolsa", gestion_bolsa: "gestion_bolsa",
  fiscalizacion: "fiscalizacion", subsanacion_unidad: "fiscalizacion",
  llamamiento: "obtencion_candidato", obtencion_candidato: "obtencion_candidato",
  nombramiento: "nombramiento", incorporacion: "incorporacion",
  seguimiento: "seguimiento", cierre: "seguimiento",
});

// Estado del servidor o de la vista → estado único visible.
const ESTADO_UNICO = Object.freeze({
  pendiente: "pendiente", en_curso: "en_curso", espera: "espera", espera_externa: "espera",
  incidencia: "incidencia", completado: "completado", cerrado: "completado", cancelado: "cancelado",
});


let rotulosActivos;
try {
  rotulosActivos = (await cargarTextos("portal", { idioma: IDIOMA_ACTUAL })).seccion("fases_rrhh");
} catch (error) {
  if (IDIOMA_ACTUAL === IDIOMA_POR_DEFECTO) throw error;
  console.warn(error);
  rotulosActivos = (await cargarTextos("portal", { idioma: IDIOMA_POR_DEFECTO })).seccion("fases_rrhh");
}

/** Rótulo del catálogo en el idioma de la interfaz. */
export function rotuloTramite(clave, variables = {}, idioma = IDIOMA_ACTUAL) {
  const catalogo = rotulosActivos;
  if (!Object.hasOwn(catalogo, clave)) throw new Error(`falta el rótulo ${clave}`);
  return Object.entries(variables).reduce(
    (texto, [nombre, valor]) => texto.replaceAll(`{${nombre}}`, String(valor)), catalogo[clave],
  );
}

/** Rótulos de las ocho fases en la forma de claves i18n de otros catálogos. */
export function rotulosFasesComoMensajes(prefijo, idioma = IDIOMA_ACTUAL) {
  return Object.fromEntries(FASES_RRHH.map((fase) => [`${prefijo}${fase}`, rotuloTramite(`fase_${fase}`, {}, idioma)]));
}

/** Fase de RRHH (clave, orden y total) de una fase del servidor; null si no tiene. */
export function faseRRHH(claveOrigen) {
  const clave = FASE_RRHH_DE_ORIGEN[claveOrigen];
  if (!clave) return null;
  return Object.freeze({ clave, orden: FASES_RRHH.indexOf(clave) + 1, total: FASES_RRHH.length });
}

export function nombreFaseRRHH(claveOrigen, idioma = IDIOMA_ACTUAL) {
  const fase = faseRRHH(claveOrigen);
  return fase ? rotuloTramite(`fase_${fase.clave}`, {}, idioma) : "";
}

/** «Fase 5 de 8» de una fase del servidor; «» si no pertenece al procedimiento. */
export function ordenFaseRRHH(claveOrigen, idioma = IDIOMA_ACTUAL) {
  const fase = faseRRHH(claveOrigen);
  return fase ? rotuloTramite("fase_de", { orden: fase.orden, total: fase.total }, idioma) : "";
}

export function estadoUnico(claveEstado) {
  return ESTADO_UNICO[claveEstado] ?? "";
}

export function nombreEstado(claveEstado, idioma = IDIOMA_ACTUAL) {
  const clave = estadoUnico(claveEstado);
  return clave ? rotuloTramite(`estado_${clave}`, {}, idioma) : "";
}


/**
 * Claves que otros catálogos i18n toman de aquí para no repetir rótulos:
 * las ocho fases (etiqueta_fase_*) y los estados (fase_* y espera externa).
 */
export function mensajesTramite(idioma = IDIOMA_ACTUAL) {
  const estado = (clave) => rotuloTramite(`estado_${clave}`, {}, idioma);
  return Object.freeze({
    ...rotulosFasesComoMensajes("etiqueta_fase_", idioma),
    fase_pendiente: estado("pendiente"),
    fase_en_curso: estado("en_curso"),
    fase_espera: estado("espera"),
    fase_completado: estado("completado"),
    fase_incidencia: estado("incidencia"),
    fase_cancelado: estado("cancelado"),
    etiqueta_estado_espera_externa: estado("espera"),
    fase_rrhh_orden: rotuloTramite("fase_de", {}, idioma),
    fase_rrhh_orden_nombre: rotuloTramite("fase_de_nombre", {}, idioma),
    linea_fase_hecho: rotuloTramite("linea_hecho", {}, idioma),
    linea_fase_ahora: rotuloTramite("linea_ahora", {}, idioma),
    linea_fase_falta: rotuloTramite("linea_falta", {}, idioma),
    linea_fase_incidencia: rotuloTramite("linea_incidencia", {}, idioma),
  });
}

/** Las mismas claves para el catálogo del portal (portada), con prefijo propio. */
export function mensajesTramitePortal(idioma = IDIOMA_ACTUAL) {
  const catalogo = rotulosActivos;
  return Object.freeze(Object.fromEntries(Object.entries(catalogo).map(([clave, texto]) => [`tramite_${clave}`, texto])));
}
