import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import * as rutasBolsa from "./portal-bolsas-ruta-filtros.js";
import { versionDe } from "./versiones-cache.test-helper.mjs";

const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = portal.indexOf("function aplicarRutaCandidatosBolsa()");
const fin = portal.indexOf("function actualizarVistaBolsa(", inicio);
assert.ok(inicio > 0 && fin > inicio, "el consumidor del enlace pertenece a la raíz");

function escenario() {
  const llamadas = [], avisos = [], navegaciones = [];
  const location = { pathname: "/portal-empleado/", search: "?lang=es&bolsa_ref=bolsa%3A1&estado=disponible",
    hash: "#bolsa/bolsa-candidatos" };
  const datos = { bolsas: [{ bolsa_ref: "bolsa:1" }, { bolsa_ref: "bolsa:2" }] };
  const estado = { vista: "bolsa-candidatos", datosBolsas: { carga: "listo", datos },
    bolsaSeleccionada: "", filtrosBolsa: {}, datosCandidatos: { carga: "listo" } };
  const entorno = { estado, rutasBolsa, controladorBolsas: { cargarCandidatosBolsa: (...argumentos) => {
    llamadas.push(argumentos); return Promise.resolve();
  } }, rutaCandidatosAplicada: null, window: { location },
  history: { replaceState(_estado, _titulo, ruta) {
    const url = new URL(ruta, "https://vec.example"); location.search = url.search; location.hash = url.hash;
  } }, navegar: (vista) => { estado.vista = vista; navegaciones.push(vista); },
  anunciar: (mensaje) => avisos.push(mensaje), traducirPortal: (clave) => clave };
  const aplicar = runInNewContext(`${portal.slice(inicio, fin)}; aplicarRutaCandidatosBolsa`, entorno);
  return { aplicar, estado, location, llamadas, navegaciones, avisos };
}

test("la raíz consulta el predicado de la URL solo tras cotejar su bolsa con el GET autorizado", () => {
  const caso = escenario();
  assert.equal(caso.aplicar(), true);
  assert.equal(caso.estado.bolsaSeleccionada, "bolsa:1");
  assert.equal(caso.estado.filtrosBolsa.estado, "disponible");
  assert.equal(caso.estado.filtrosBolsa.texto, "");
  assert.deepEqual(caso.llamadas.map(([ref]) => ref), ["bolsa:1"]);
  caso.aplicar();
  assert.equal(caso.llamadas.length, 1, "repintar no repite la consulta");
  caso.location.search = "?lang=es&bolsa_ref=bolsa%3A1&estado=no_disponible";
  caso.aplicar();
  assert.equal(caso.estado.filtrosBolsa.estado, "no_disponible");
  assert.equal(caso.llamadas.length, 2, "Atrás o un filtro distinto releen su página exacta");
});

test("referencia, estado o cursor ajenos no inician GET de candidaturas", () => {
  for (const search of [
    "?lang=es&bolsa_ref=bolsa%3Aotro&estado=disponible",
    "?lang=es&bolsa_ref=bolsa%3A1&estado=desconocido",
    "?lang=es&bolsa_ref=bolsa%3A1&cursor=opaco",
    "?lang=es&bolsa_ref=bolsa%3A1&bolsa_ref=bolsa%3A2",
  ]) {
    const caso = escenario(); caso.location.search = search;
    assert.equal(caso.aplicar(), true, search);
    assert.deepEqual(caso.llamadas, [], search);
    assert.deepEqual(caso.navegaciones, ["resumen"], search);
    assert.equal(caso.location.search, "?lang=es", "solo se retiran filtros de Bolsa");
    assert.equal(caso.location.hash, "#bolsa/resumen");
    assert.equal(caso.avisos.length, 1);
  }
});

test("popstate cambia la vista al volver y hashchange no duplica el montaje de candidaturas", () => {
  const inicioInstalacion = portal.indexOf("function instalarEnlacesBolsa()");
  const finInstalacion = portal.indexOf("function instalarEventosAuditoriaBolsa()", inicioInstalacion);
  assert.ok(inicioInstalacion > 0 && finInstalacion > inicioInstalacion);
  const oyentes = new Map(), navegaciones = [];
  let aplicadas = 0;
  const location = { hash: "#bolsa/bolsa-candidatos" };
  const estado = { vista: "bolsa-candidatos" };
  const vistaDesdeHash = () => location.hash === "#bolsa/resumen" ? "resumen"
    : location.hash === "#bolsa/bolsa-candidatos" ? "bolsa-candidatos" : "portal";
  const navegar = (vista) => { estado.vista = vista; navegaciones.push(vista); };
  const contexto = { document: { addEventListener() {} }, window: { location,
    addEventListener(tipo, escucha) { oyentes.set(tipo, escucha); } }, estado, vistaDesdeHash,
  navegar, aplicarRutaCandidatosBolsa: () => { aplicadas += 1; }, rutaCandidatosAplicada: null };
  runInNewContext(`${portal.slice(inicioInstalacion, finInstalacion)}; instalarEnlacesBolsa()`, contexto);
  const hashchange = () => { const vista = vistaDesdeHash(); if (vista !== estado.vista) navegar(vista); };

  location.hash = "#bolsa/resumen"; oyentes.get("popstate")(); hashchange();
  assert.deepEqual(navegaciones, ["resumen"], "Atrás monta el cuadro, no deja Candidatos visible");
  assert.equal(aplicadas, 0);
  location.hash = "#bolsa/bolsa-candidatos"; oyentes.get("popstate")(); hashchange();
  assert.deepEqual(navegaciones, ["resumen", "bolsa-candidatos"]);
  assert.equal(aplicadas, 1, "Adelante aplica el filtro una vez");
  oyentes.get("popstate")();
  assert.equal(aplicadas, 2, "un cambio de query en Candidatos relee el filtro sin remontar la vista");
  assert.equal(navegaciones.length, 2);
  location.hash = "#portal"; oyentes.get("popstate")(); hashchange();
  assert.equal(navegaciones.at(-1), "portal");
  assert.equal(aplicadas, 2);
});

test("la cohorte de CSS, entrada y helper coincide con las URL servidas", async () => {
  const [html, cache, interno, produccion] = await Promise.all([
    "index.html", "cache-publica-v1.json", "../../interno.manifest", "../../produccion.manifest",
  ].map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  const versionRaiz = versionDe(html, "/portal-empleado/portal.js");
  assert.equal(versionRaiz, "20261008-w-fichas-capacidades-v2");
  assert.equal(versionDe(cache, "/portal-empleado/portal.js"), versionRaiz);
  for (const css of ["portal-componentes.css", "portal-capacidades.css"])
    assert.equal(versionDe(html, `/portal-empleado/${css}`), "20261008-bolsa-enlaces-v1");
  assert.equal(versionDe(cache, "/portal-empleado/portal-componentes.css"), "20261008-bolsa-enlaces-v1");
  for (const manifiesto of [interno, produccion]) {
    assert.equal(manifiesto.split("static/portal-empleado/portal-bolsas-ruta-filtros.js").length - 1, 1);
  }
});
