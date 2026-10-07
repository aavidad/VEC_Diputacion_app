import * as textosComunes from "../../../comun/textos.js";
import { publicarMensajesBorradores } from "./i18n-borradores.js?v=20260929-i18n-dietas-v1";
import { publicarMensajesRevisionDietas } from "./i18n-revision.js?v=20260929-i18n-dietas-v1";
import { publicarMensajesCircuitoDietas } from "./i18n-circuito.js?v=20260929-i18n-dietas-v1";
import { publicarMensajesRectificacionDietas } from "./i18n-rectificacion-dietas.js?v=20260929-i18n-dietas-v1";
import { publicarMensajesRectificacionAdmin } from "./i18n-rectificacion-admin.js?v=20260929-i18n-dietas-v1";
import { publicarMensajesOtrosGastos } from "./i18n-otros-gastos.js?v=20260929-i18n-dietas-v1";

const PUBLICAR = Object.freeze({
  borradores: publicarMensajesBorradores,
  revision: publicarMensajesRevisionDietas,
  circuito: publicarMensajesCircuitoDietas,
  rectificacion_dietas: publicarMensajesRectificacionDietas,
  rectificacion_admin: publicarMensajesRectificacionAdmin,
  otros_gastos: publicarMensajesOtrosGastos,
});
export let MENSAJES_DIETAS;
let CLAVES;
let secuencia = 0;

function limpiar() {
  MENSAJES_DIETAS = undefined;
  CLAVES = undefined;
  for (const publicar of Object.values(PUBLICAR)) publicar(undefined);
}

function validarSeccion(seccion) {
  if (!seccion || typeof seccion !== "object" || Array.isArray(seccion))
    throw new TypeError("catálogo i18n de Dietas incompleto");
  const copia = Object.entries(seccion);
  if (copia.length === 0 || copia.some(([clave, valor]) =>
    !clave || typeof valor !== "string" || valor === ""))
    throw new TypeError("catálogo i18n de Dietas incompleto");
  return Object.freeze(Object.fromEntries(copia));
}

/** La composición llama a esta función al abrir Dietas, antes de crear vistas o mapa. */
export async function prepararTextosDietas({ reintentar = false, cargar = textosComunes.cargarTextos,
  releer = textosComunes.reintentarTextos } = {}) {
  const propia = ++secuencia;
  limpiar();
  if (typeof reintentar !== "boolean" || typeof cargar !== "function" ||
      (reintentar && typeof releer !== "function"))
    throw new TypeError("lector de textos de Dietas no disponible");
  try {
    const textos = await (reintentar ? releer("dietas") : cargar("dietas"));
    if (propia !== secuencia) throw new Error("preparación de textos de Dietas sustituida");
    if (typeof textos?.seccion !== "function" || typeof textos.idioma !== "string" ||
        typeof textos.localizacion !== "string")
      throw new TypeError("catálogo i18n de Dietas incompleto");
    const general = validarSeccion(textos.seccion("general"));
    const secciones = Object.fromEntries(Object.keys(PUBLICAR).map((nombre) =>
      [nombre, validarSeccion(textos.seccion(nombre))]));
    const nombres = [general, ...Object.values(secciones)].flatMap((seccion) => Object.keys(seccion));
    if (new Set(nombres).size !== nombres.length) throw new TypeError("catálogo i18n de Dietas incompleto");
    const mensajes = Object.freeze(Object.assign({}, general, secciones.borradores,
      secciones.revision, secciones.circuito, secciones.rectificacion_dietas,
      secciones.rectificacion_admin));
    for (const [nombre, publicar] of Object.entries(PUBLICAR)) publicar(secciones[nombre]);
    CLAVES = Object.freeze(Object.keys(mensajes));
    MENSAJES_DIETAS = mensajes;
    return Object.freeze({ idioma: textos.idioma, localizacion: textos.localizacion,
      incidenciaCatalogo: textos.incidenciaCatalogo ?? null,
      incidenciaIndice: textos.incidenciaIndice ?? null });
  } catch (error) {
    if (propia === secuencia) limpiar();
    throw error;
  }
}

export function crearTraductorDietas(catalogo = MENSAJES_DIETAS) {
  if (!CLAVES || !catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Dietas incompleto");
  }
  const preparadoEn = secuencia;
  return (clave, variables = {}) => {
    if (preparadoEn !== secuencia || !CLAVES) throw new Error("textos de Dietas sin preparar");
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Dietas desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
