import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import * as rutas from "./portal-bolsas-ruta-filtros.js";

const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = portal.indexOf("function instalarEnlacesBolsa()");
const fin = portal.indexOf("function instalarEventosAuditoriaBolsa()", inicio);

function escenario({ autorizada = true, cambiarFicha = false } = {}) {
  const bolsaRef = "bolsa:administrativo", llamamientoRef = "llamamiento:emitido";
  const location = { origin: "https://vec.example", pathname: "/portal-empleado/",
    search: "?lang=en", hash: "#contratacion-temporal" };
  Object.defineProperty(location, "href", { get: () => `${location.origin}${location.pathname}${location.search}${location.hash}` });
  const control = { tagName: "BUTTON", dataset: { bolsaRef, seguimiento: llamamientoRef },
    classList: { contains: () => true }, closest: (selector) => selector === "[data-ct-resultado-bolsa]" ? {} : null };
  const estado = { vista: "contratacion-temporal", filtrosBolsa: {}, datosBolsas: null };
  const avisos = [], navegaciones = [], rutasAplicadas = [];
  let click;
  const entorno = { estado, contextoBolsaCT: { expedienteRef: "expediente:actual", carga: "listo",
    datos: { bolsas: autorizada ? [{ bolsa_ref: bolsaRef }] : [] } }, URL, URLSearchParams,
    window: { location, addEventListener() {} },
    document: { addEventListener(tipo, fn) { if (tipo === "click") click = fn; } },
    porId: () => ({ contains: (elemento) => elemento === control }),
    prepararRutasBolsa: async () => { if (cambiarFicha) entorno.contextoBolsaCT = null; return rutas; },
    history: { pushState(_a, _b, ruta) { const url = new URL(ruta, location.href); location.search = url.search; location.hash = url.hash; } },
    navegar: (vista) => navegaciones.push(vista),
    aplicarRutaCandidatosBolsa: () => rutasAplicadas.push(location.search),
    anunciar: (mensaje) => avisos.push(mensaje), traducirPortal: (clave) => clave,
    controladorBolsas: {}, rutaCandidatosAplicada: null };
  runInNewContext(`${portal.slice(inicio, fin)}; instalarEnlacesBolsa()`, entorno);
  return { location, estado, avisos, navegaciones, rutasAplicadas,
    pulsar: () => click({ target: { closest: () => control }, stopImmediatePropagation() {}, preventDefault() {} }) };
}

test("desde la ficha abre el seguimiento exacto con su lectura autorizada aunque Inicio no se haya cargado", async () => {
  const caso = escenario();
  await caso.pulsar();
  assert.deepEqual(caso.avisos, []);
  assert.deepEqual(caso.navegaciones, ["bolsa-candidatos"]);
  assert.equal(caso.location.hash, "#bolsa/bolsa-candidatos");
  const parametros = new URLSearchParams(caso.location.search);
  assert.equal(parametros.get("bolsa_ref"), "bolsa:administrativo");
  assert.equal(parametros.get("seguimiento"), "llamamiento:emitido");
  assert.equal(parametros.get("lang"), "en");
  assert.equal(caso.rutasAplicadas.length, 1);
});

test("una bolsa fuera del ámbito o una ficha cambiada durante la carga no navegan", async () => {
  for (const opciones of [{ autorizada: false }, { cambiarFicha: true }]) {
    const caso = escenario(opciones);
    await caso.pulsar();
    assert.deepEqual(caso.navegaciones, []);
    assert.deepEqual(caso.rutasAplicadas, []);
    assert.equal(caso.location.hash, "#contratacion-temporal");
    assert.equal(caso.avisos.length, opciones.autorizada === false ? 1 : 0);
  }
});
