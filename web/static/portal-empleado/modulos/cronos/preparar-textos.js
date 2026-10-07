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
let siguienteIntento = 0;
let ultimoEstado = { orden: 0, idioma: null, valido: false };

function seccion(textos, nombre) {
  const mensajes = textos.seccion(nombre);
  if (!mensajes || !Object.keys(mensajes).length
    || Object.values(mensajes).some((valor) => typeof valor !== "string" || !valor.trim())) {
    throw new TypeError("catálogo de Cronos incompleto");
  }
  return mensajes;
}

function validarCatalogo(textos) {
  if (textos.incidenciaIndice || textos.faltantes?.length
    || (textos.incidenciaCatalogo && textos.incidenciaCatalogo.respaldo !== textos.idioma)) {
    throw new Error("textos de Cronos pendientes de recuperación");
  }
  return textos;
}

function invalidarTodo() {
  for (const instalar of [
    instalarMENSAJES_CRONOS, instalarMENSAJES_CRONOS_SOLICITUDES,
    instalarMENSAJES_CRONOS_PERMISOS, instalarMENSAJES_JUSTIFICACION_CRONOS,
    instalarMENSAJES_CRONOS_RESOLUCION, instalarMENSAJES_BANDEJA,
    instalarMENSAJES_CRONOS_NOTIFICACIONES, instalarMENSAJES_CONSULTA_CRONOS,
    instalarMENSAJES_FICHAJE_CRONOS, instalarMENSAJES_CRONOS_INCIDENCIAS,
    instalarMENSAJES_HISTORIAL_CRONOS, instalarMENSAJES_CONSULTA_PERMISOS_CRONOS,
    instalarMENSAJES_NOTIFICACIONES_HISTORIAL_CRONOS,
    instalarMENSAJES_BANDEJA_NOTIFICACIONES_CRONOS,
  ]) instalar(undefined);
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
  const orden = ++siguienteIntento;
  const generacion = (generaciones.get(pantalla) ?? 0) + (reintentar ? 1 : 0);
  generaciones.set(pantalla, generacion);
  const cargar = reintentar ? lector.reintentarTextos : lector.cargarTextos;
  if (typeof cargar !== "function") throw new TypeError("lector común de textos no disponible");
  const intento = Promise.all(fuentes.map(async (fuente) => [fuente, validarCatalogo(await cargar(fuente))]))
    .then(async (pares) => {
      if (generaciones.get(pantalla) !== generacion) throw new Error("preparación de Cronos superada");
      // Si una fuente usa el respaldo, toda la pantalla usa ese mismo idioma.
      const respaldo = pares.find(([, textos]) => textos.incidenciaCatalogo)?.[1].idioma;
      if (respaldo) {
        pares = await Promise.all(fuentes.map(async (fuente) => [fuente, validarCatalogo(await lector.cargarTextos(
          fuente, { idioma: respaldo, porDefecto: respaldo },
        ))]));
      }
      const catalogos = Object.fromEntries(pares);
      const idioma = catalogos.cronos.idioma;
      if (pares.some(([, textos]) => textos.idioma !== idioma || textos.incidenciaCatalogo)) {
        throw new Error("idiomas de Cronos incompatibles");
      }
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
      if (generaciones.get(pantalla) !== generacion
        || (ultimoEstado.orden > orden && (!ultimoEstado.valido || ultimoEstado.idioma !== idioma))) {
        throw new Error("preparación de Cronos superada");
      }
      if (ultimoEstado.valido && ultimoEstado.idioma !== idioma) invalidarTodo();
      publicar(pantalla, catalogos);
      if (orden >= ultimoEstado.orden) ultimoEstado = { orden, idioma, valido: true };
      return Object.freeze({ pantalla, idioma, localizacion: catalogos.cronos.localizacion });
    }).catch((error) => {
      if (generaciones.get(pantalla) === generacion && orden >= ultimoEstado.orden) {
        invalidarTodo();
        ultimoEstado = { orden, idioma: null, valido: false };
      }
      throw error;
    });
  return intento;
}
