import * as lectorComun from "../../../comun/textos.js";
import { instalarMENSAJES_CRONOS } from "./i18n.js?v=20260929-i18n-textos-v1";
import { instalarMENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";
import { instalarMENSAJES_CRONOS_PERMISOS, instalarMENSAJES_JUSTIFICACION_CRONOS } from "./i18n-permisos.js?v=20261001-cronos-grafo-bandeja-v5";
import { instalarMENSAJES_CRONOS_RESOLUCION, instalarMENSAJES_BANDEJA } from "./i18n-resolucion.js?v=20261001-cronos-grafo-bandeja-v5";
import { instalarMENSAJES_CRONOS_NOTIFICACIONES } from "./i18n-notificaciones.js";
import { instalarMENSAJES_CONSULTA_CRONOS } from "./i18n-consulta.js?v=20261001-cronos-grafo-bandeja-v5";
import { instalarMENSAJES_FICHAJE_CRONOS } from "./i18n-fichaje.js?v=20261001-cronos-grafo-bandeja-v5";
import { instalarMENSAJES_CRONOS_INCIDENCIAS } from "./i18n-incidencias.js?v=20261001-cronos-grafo-bandeja-v5";
import { instalarMENSAJES_HISTORIAL_CRONOS } from "./i18n-historial.js?v=20261001-cronos-historial-v1";
import { instalarMENSAJES_CONSULTA_PERMISOS_CRONOS } from "./i18n-permisos-consulta.js?v=20261001-cronos-c7-consulta-v2";
import { instalarMENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS } from "./i18n-notificaciones-historial.js?v=20261001-cronos-c9-historial-v2";
import { instalarMENSAJES_BANDEJA_NOTIFICACIONES_CRONOS } from "./i18n-bandeja-notificaciones.js?v=20261001-cronos-c9-recuperacion-v3";

const FUENTES = Object.freeze({
  jornada: ["cronos", "cronos-consulta", "cronos-fichaje", "cronos-incidencias"],
  permisos: ["cronos", "cronos-historial", "cronos-permisos", "cronos-permisos-consulta"],
  resolucion: ["cronos", "cronos-resolucion"],
  notificaciones: ["cronos", "cronos-notificaciones-historial", "cronos-bandeja-notificaciones"],
});
const SECCIONES_BASE = Object.freeze({
  jornada: ["general", "solicitudes"],
  permisos: ["general", "solicitudes", "permisos"],
  resolucion: ["general", "solicitudes", "resolucion"],
  notificaciones: ["general", "solicitudes", "notificaciones"],
});

const generaciones = new Map();

function seccion(textos, nombre) {
  const mensajes = textos.seccion(nombre);
  if (!mensajes || !Object.keys(mensajes).length
    || Object.values(mensajes).some((valor) => typeof valor !== "string" || !valor.trim())) {
    throw new TypeError("catálogo de Cronos incompleto");
  }
  return mensajes;
}

function validarCatalogo(textos) {
  if (textos.incidenciaIndice || textos.incidenciaCatalogo || textos.faltantes?.length) {
    throw new Error("textos de Cronos pendientes de recuperación");
  }
  return textos;
}

function publicar(pantalla, catalogos) {
  const base = catalogos.cronos;
  if (base) {
    instalarMENSAJES_CRONOS(seccion(base, "general"));
    instalarMENSAJES_CRONOS_SOLICITUDES(seccion(base, "solicitudes"));
    if (pantalla === "permisos") instalarMENSAJES_CRONOS_PERMISOS(seccion(base, "permisos"));
    if (pantalla === "resolucion") instalarMENSAJES_CRONOS_RESOLUCION(seccion(base, "resolucion"));
    if (pantalla === "notificaciones") instalarMENSAJES_CRONOS_NOTIFICACIONES(seccion(base, "notificaciones"));
  }
  if (catalogos["cronos-consulta"]) instalarMENSAJES_CONSULTA_CRONOS(seccion(catalogos["cronos-consulta"], "consulta"));
  if (catalogos["cronos-fichaje"]) instalarMENSAJES_FICHAJE_CRONOS(seccion(catalogos["cronos-fichaje"], "general"));
  if (catalogos["cronos-incidencias"]) instalarMENSAJES_CRONOS_INCIDENCIAS(seccion(catalogos["cronos-incidencias"], "incidencias"));
  if (catalogos["cronos-historial"]) instalarMENSAJES_HISTORIAL_CRONOS(seccion(catalogos["cronos-historial"], "historial"));
  if (catalogos["cronos-permisos"]) instalarMENSAJES_JUSTIFICACION_CRONOS(seccion(catalogos["cronos-permisos"], "justificacion"));
  if (catalogos["cronos-permisos-consulta"]) instalarMENSAJES_CONSULTA_PERMISOS_CRONOS(seccion(catalogos["cronos-permisos-consulta"], "consulta"));
  if (catalogos["cronos-resolucion"]) instalarMENSAJES_BANDEJA(seccion(catalogos["cronos-resolucion"], "bandeja"));
  if (catalogos["cronos-notificaciones-historial"]) instalarMENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS(seccion(catalogos["cronos-notificaciones-historial"], "historial"));
  if (catalogos["cronos-bandeja-notificaciones"]) instalarMENSAJES_BANDEJA_NOTIFICACIONES_CRONOS(seccion(catalogos["cronos-bandeja-notificaciones"], "bandeja"));
}

/** El shell espera este resultado antes de construir la pantalla indicada. */
export function prepararTextosCronos({ pantalla = "jornada", reintentar = false, lector = lectorComun } = {}) {
  const fuentes = FUENTES[pantalla];
  if (!fuentes) throw new TypeError("pantalla de Cronos desconocida");
  const generacion = (generaciones.get(pantalla) ?? 0) + 1;
  generaciones.set(pantalla, generacion);
  const cargar = reintentar ? lector.reintentarTextos : lector.cargarTextos;
  if (typeof cargar !== "function") throw new TypeError("lector común de textos no disponible");
  const intento = Promise.all(fuentes.map(async (fuente) => [fuente, validarCatalogo(await cargar(fuente))]))
    .then((pares) => {
      if (generaciones.get(pantalla) !== generacion) throw new Error("preparación de Cronos superada");
      const catalogos = Object.fromEntries(pares);
      // Validar todas las secciones antes de publicar cualquiera de ellas.
      for (const [fuente, textos] of pares) {
        const nombres = {
          cronos: SECCIONES_BASE[pantalla],
          "cronos-fichaje": ["general"], "cronos-permisos": ["justificacion"], "cronos-resolucion": ["bandeja"],
          "cronos-bandeja-notificaciones": ["bandeja"], "cronos-historial": ["historial"],
          "cronos-notificaciones-historial": ["historial"],
          "cronos-consulta": ["consulta"], "cronos-incidencias": ["incidencias"], "cronos-permisos-consulta": ["consulta"],
        }[fuente];
        for (const nombre of nombres) seccion(textos, nombre);
      }
      publicar(pantalla, catalogos);
      return Object.freeze({ pantalla, idioma: catalogos.cronos.idioma, localizacion: catalogos.cronos.localizacion });
    });
  return intento;
}
