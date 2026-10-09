import { aplicarPreferenciasInicialesAplazadas, exigirParametrosConocidos, iniciarAreaPersonal } from "./aplicacion.js?v=20261009-nombre-propio-v2";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { cargarPreferenciasIniciales, crearClientePreferencias } from "./cliente-http.js?v=20261009-nombre-propio-v2";
import { cargarVistasDisponibles } from "./vistas-disponibles.js?v=20261005-b4b-v1";
import * as temaComun from "../comun/tema-vec.js?v=20260930-codexf-temas-v2";
import { INDICE_IDIOMAS, prepararIdiomas } from "../comun/idioma.js";
import { peticionesEnSerie } from "../comun/imagen-propia.js?v=20261007-p7-http-v1";

// Una sola cola para las consultas de Usuarios de esta página. La consulta a
// Bolsa acaba antes de arrancar imagen o preferencias.
const fetchUsuarios = peticionesEnSerie(globalThis.fetch);
const clientePreferencias = crearClientePreferencias({ fetchImpl: fetchUsuarios });
try { await prepararIdiomas(); } catch { /* i18n conserva el índice del documento. */ }
const idiomaURL = new URLSearchParams(window.location.search).get("lang");
const idiomaExplicito = INDICE_IDIOMAS.idiomas.some(({ codigo }) => codigo === idiomaURL);
let preferencias = null;
let errorPreferencias = null;
if (!idiomaExplicito) {
  try { preferencias = await cargarPreferenciasIniciales(clientePreferencias); }
  catch (error) { errorPreferencias = error; }
}
await iniciarI18nAreaPersonal(document, { idiomaPreferido: preferencias?.estado.valores.idioma });
const controladorVisual = preferencias && typeof temaComun.aplicarPreferenciasVisuales === "function"
  ? temaComun.aplicarPreferenciasVisuales(preferencias.estado.valores, { documento: document, ventana: window })
  : typeof temaComun.crearControladorPreferenciasVisuales === "function"
    ? temaComun.crearControladorPreferenciasVisuales({ documento: document, ventana: window }) : null;

async function resolverCliente() {
  exigirParametrosConocidos(new URLSearchParams(window.location.search));
  const { crearClienteHTTPAreaPersonal } = await import("./cliente-http.js?v=20261009-nombre-propio-v2");
  return { cliente: crearClienteHTTPAreaPersonal(), vistasDisponibles: await cargarVistasDisponibles() };
}

try {
  const dependencias = await resolverCliente();
  const estado = await iniciarAreaPersonal({ ...dependencias, clientePreferencias, preferencias,
    errorPreferencias, controladorVisual, fetchUsuarios, preferenciasAplazadas: idiomaExplicito });
  if (idiomaExplicito) {
    let lectura = null; let errorPreferencia = null;
    try { lectura = await cargarPreferenciasIniciales(clientePreferencias); }
    catch (error) { errorPreferencia = error; }
    aplicarPreferenciasInicialesAplazadas(estado, lectura, errorPreferencia);
  }
} catch {
  const carga = document.getElementById("estado-carga");
  if (carga) {
    carga.className = "estado-error";
    const titulo = document.createElement("h2");
    titulo.textContent = traducir("areaPersonal.estado.error.titulo");
    const detalle = document.createElement("p");
    detalle.textContent = traducir("areaPersonal.estado.error.detalle");
    const garantia = document.createElement("p");
    garantia.textContent = traducir("areaPersonal.estado.error.carga.garantia");
    carga.replaceChildren(titulo, detalle, garantia);
  }
}
