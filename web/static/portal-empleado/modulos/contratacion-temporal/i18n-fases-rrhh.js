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
import { IDIOMA_ACTUAL } from "../../../comun/idioma.js";
import { cargarCatalogosContratacion, cargarCatalogosContratacionEnIdioma } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";
import { FASES_RRHH, faseRRHH } from "./fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";
export { FASES_RRHH, FASE_RRHH_DE_ORIGEN, faseRRHH } from "./fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";

// Estado del servidor o de la vista → estado único visible.
const ESTADO_UNICO = Object.freeze({
  pendiente: "pendiente", en_curso: "en_curso", espera: "espera", espera_externa: "espera",
  incidencia: "incidencia", completado: "completado", cerrado: "completado", cancelado: "cancelado",
});


const ROTULOS = (await cargarCatalogosContratacion("portal", "fases_rrhh")).actual;

function comprobarIdiomaCargado(idioma) {
  if (idioma !== IDIOMA_ACTUAL) throw new RangeError("catálogo del idioma solicitado no cargado");
}

/** Rótulo del catálogo en el idioma de la interfaz. */
export function rotuloTramite(clave, variables = {}, idioma = IDIOMA_ACTUAL) {
  comprobarIdiomaCargado(idioma);
  const catalogo = ROTULOS;
  if (!Object.hasOwn(catalogo, clave)) throw new Error(`falta el rótulo ${clave}`);
  return Object.entries(variables).reduce(
    (texto, [nombre, valor]) => texto.replaceAll(`{${nombre}}`, String(valor)), catalogo[clave],
  );
}

/** Rótulos de las ocho fases en la forma de claves i18n de otros catálogos. */
export function rotulosFasesComoMensajes(prefijo, idioma = IDIOMA_ACTUAL) {
  return Object.fromEntries(FASES_RRHH.map((fase) => [`${prefijo}${fase}`, rotuloTramite(`fase_${fase}`, {}, idioma)]));
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
  comprobarIdiomaCargado(idioma);
  return mensajesTramiteDe(ROTULOS);
}

function mensajesTramiteDe(catalogo) {
  const rotulo = (clave) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`falta el rótulo ${clave}`);
    return catalogo[clave];
  };
  const estado = (clave) => rotulo(`estado_${clave}`);
  return Object.freeze({
    ...Object.fromEntries(FASES_RRHH.map((fase) => [`etiqueta_fase_${fase}`, rotulo(`fase_${fase}`)])),
    fase_pendiente: estado("pendiente"),
    fase_en_curso: estado("en_curso"),
    fase_espera: estado("espera"),
    fase_completado: estado("completado"),
    fase_incidencia: estado("incidencia"),
    fase_cancelado: estado("cancelado"),
    etiqueta_estado_espera_externa: estado("espera"),
    fase_rrhh_orden: rotulo("fase_de"),
    fase_rrhh_orden_nombre: rotulo("fase_de_nombre"),
    linea_fase_hecho: rotulo("linea_hecho"),
    linea_fase_ahora: rotulo("linea_ahora"),
    linea_fase_falta: rotulo("linea_falta"),
    linea_fase_incidencia: rotulo("linea_incidencia"),
  });
}

/** El idioma distinto del activo se solicita de forma explícita y asíncrona. */
export async function cargarMensajesTramiteEnIdioma(idioma) {
  const { actual } = await cargarCatalogosContratacionEnIdioma("portal", idioma, "fases_rrhh");
  return mensajesTramiteDe(actual);
}

/** Las mismas claves para el catálogo del portal (portada), con prefijo propio. */
export function mensajesTramitePortal(idioma = IDIOMA_ACTUAL) {
  comprobarIdiomaCargado(idioma);
  const catalogo = ROTULOS;
  return Object.freeze(Object.fromEntries(Object.entries(catalogo).map(([clave, texto]) => [`tramite_${clave}`, texto])));
}

/** Para consultas explícitas en otro idioma, sin precargarlo en la navegación. */
export async function cargarMensajesTramitePortalEnIdioma(idioma) {
  const { actual } = await cargarCatalogosContratacionEnIdioma("portal", idioma, "fases_rrhh");
  return Object.freeze(Object.fromEntries(Object.entries(actual)
    .map(([clave, texto]) => [`tramite_${clave}`, texto])));
}
