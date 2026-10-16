import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import * as rutasBolsa from "./portal-bolsas-ruta-filtros.js?v=20261008-bolsa-global-v2";
import { versionDe } from "./versiones-cache.test-helper.mjs";

const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = portal.indexOf("function aplicarRutaCandidatosBolsa()");
const fin = portal.indexOf("function actualizarVistaBolsa(", inicio);
assert.ok(inicio > 0 && fin > inicio, "el consumidor del enlace pertenece a la raíz");

function escenario() {
  const llamadas = [], avisos = [], navegaciones = [], globales = [];
  const location = { pathname: "/portal-empleado/", search: "?lang=es&bolsa_ref=bolsa%3A1&estado=disponible",
    hash: "#bolsa/bolsa-candidatos" };
  const datos = { bolsas: [{ bolsa_ref: "bolsa:1" }, { bolsa_ref: "bolsa:2" }] };
  const estado = { vista: "bolsa-candidatos", datosBolsas: { carga: "listo", datos },
    bolsaSeleccionada: "", filtrosBolsa: {}, datosCandidatos: { carga: "listo" } };
  const entorno = { estado, rutasBolsa, controladorBolsas: { cargarGlobalBolsa: (...argumentos) => { globales.push(argumentos); return Promise.resolve(); }, cargarCandidatosBolsa: (...argumentos) => {
    llamadas.push(argumentos); return Promise.resolve();
  } }, rutaCandidatosAplicada: null, window: { location },
  history: { replaceState(_estado, _titulo, ruta) {
    const url = new URL(ruta, "https://vec.example"); location.search = url.search; location.hash = url.hash;
  } }, navegar: (vista) => { estado.vista = vista; navegaciones.push(vista); },
  anunciar: (mensaje) => avisos.push(mensaje), traducirPortal: (clave) => clave };
  const aplicar = runInNewContext(`${portal.slice(inicio, fin)}; aplicarRutaCandidatosBolsa`, entorno);
  return { aplicar, estado, location, llamadas, navegaciones, avisos, globales };
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
  const inicioGuarda = portal.indexOf("function salidaGestionOfertasBloqueada()");
  const finGuarda = portal.indexOf("function anunciarSalidaGestionOfertas()", inicioGuarda);
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
  navegar, aplicarRutaCandidatosBolsa: () => { aplicadas += 1; }, rutaCandidatosAplicada: null,
  modoGestionOfertasBolsa: "", superficieOfertasBolsa: null, rutaOfertasSolicitadaRef: "",
  CLAVE_RUTA_GESTION_OFERTAS: "ofertas_bolsa_ref", URLSearchParams,
  actualizarVistaBolsa: () => {}, cerrarGestionOfertasBolsa: () => {} };
  runInNewContext(`${portal.slice(inicioGuarda, finGuarda)}\n${portal.slice(inicioInstalacion, finInstalacion)}; instalarEnlacesBolsa()`, contexto);
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

test("el enlace a otra bolsa conserva la selección y la operación incierta, incluso durante la carga del helper", async () => {
  const inicioInstalacion = portal.indexOf("function instalarEnlacesBolsa()");
  const finInstalacion = portal.indexOf("function instalarEventosAuditoriaBolsa()", inicioInstalacion);
  let escuchar, liberar, focos = 0, cargas = 0;
  const control = { tagName: "BUTTON", dataset: { bolsaRef: "bolsa:otra" }, classList: { contains: () => false },
    closest: () => null };
  const estado = { vista: "bolsa-candidatos", bolsaSeleccionada: "bolsa:original",
    datosBolsas: { carga: "listo", datos: { bolsas: [{ bolsa_ref: "bolsa:otra" }] } }, filtrosBolsa: {},
    modalFicha: { operacionesB8: { directa: { pendiente: { clave: "clave-original" } } }, resultadoInciertoCambioSituacion: true } };
  const contexto = { estado, document: { addEventListener(tipo, fn) { if (tipo === "click") escuchar = fn; },
    querySelector: () => ({ focus: () => { focos++; } }) }, window: { addEventListener() {} },
  porId: () => ({ contains: () => true }), prepararRutasBolsa: () => { cargas++; return new Promise((resolve) => { liberar = resolve; }); },
  controladorBolsas: {}, anunciar() {}, traducirPortal: (clave) => clave, URL, history: {}, navegar() {} };
  runInNewContext(`${portal.slice(inicioInstalacion, finInstalacion)}; instalarEnlacesBolsa()`, contexto);
  const evento = { target: { closest: (selector) => selector.includes('data-accion="ver-bolsa"') ? control : null },
    stopImmediatePropagation() {}, preventDefault() {} };
  await escuchar(evento);
  assert.equal(cargas, 0); assert.equal(estado.bolsaSeleccionada, "bolsa:original");
  assert.equal(estado.modalFicha.operacionesB8.directa.pendiente.clave, "clave-original");
  assert.equal(focos, 1);
  estado.modalFicha.resultadoInciertoCambioSituacion = false;
  const apertura = escuchar(evento);
  assert.equal(cargas, 1);
  estado.modalFicha.enviandoCambioSituacion = true;
  liberar({}); await apertura;
  assert.equal(estado.bolsaSeleccionada, "bolsa:original");
  assert.equal(estado.filtrosBolsa.estado, undefined);
});

test("Atrás compensa la ruta mientras la aceptación sigue pendiente", () => {
  const inicioCompensar = portal.indexOf("function compensarHistorialGestionOfertas(");
  const finCompensar = portal.indexOf("function actualizarSalidaGestionOfertasBolsa()", inicioCompensar);
  const inicioInstalacion = portal.indexOf("function instalarEnlacesBolsa()");
  const finInstalacion = portal.indexOf("function instalarEventosAuditoriaBolsa()", inicioInstalacion);
  const rutaFicha = "/portal-empleado/?bolsa_ref=bolsa%3Auno#bolsa/bolsa-candidatos";
  const location = { pathname: "/portal-empleado/", search: "", hash: "#portal" };
  const pendiente = { clave: "clave-original", cuerpo: { motivo: "Aceptación" } };
  const estado = { vista: "bolsa-candidatos", bolsaSeleccionada: "bolsa:uno", filtrosBolsa: {},
    modalFicha: { rutaApertura: rutaFicha, resultadoInciertoCambioSituacion: true,
      operacionesB8: { directa: { pendiente } } } };
  let popstate, focos = 0, cambiosVista = 0, compensaciones = 0;
  const contexto = { estado, window: { location, addEventListener(tipo, fn) { if (tipo === "popstate") popstate = fn; } },
    document: { addEventListener() {}, querySelector: () => ({ focus: () => { focos++; } }) },
    history: { pushState(_datos, _titulo, ruta) { compensaciones++; const url = new URL(ruta, "https://vec.example");
      location.pathname = url.pathname; location.search = url.search; location.hash = url.hash; } },
    rutaVisibleGestionOfertasBolsa: "", compensandoHistorialGestionOfertas: false,
    navegar: () => { cambiosVista++; }, rutaCandidatosAplicada: null,
    salidaGestionOfertasBloqueada: () => false, vistaDesdeHash: () => "portal" };
  runInNewContext(`${portal.slice(inicioCompensar, finCompensar)}\n${portal.slice(inicioInstalacion, finInstalacion)}; instalarEnlacesBolsa()`, contexto);
  popstate();
  assert.equal(`${location.pathname}${location.search}${location.hash}`, rutaFicha);
  assert.equal(compensaciones, 1); assert.equal(cambiosVista, 0); assert.equal(focos, 1);
  assert.equal(estado.modalFicha.operacionesB8.directa.pendiente, pendiente);
});

test("la cohorte de CSS, entrada y helper coincide con las URL servidas", async () => {
  const [html, cache, interno, produccion] = await Promise.all([
    "index.html", "cache-publica-v1.json", "../../interno.manifest", "../../produccion.manifest",
  ].map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  const versionRaiz = versionDe(html, "/portal-empleado/portal.js");
  assert.notEqual(versionRaiz, "20261008-documentos-ficha-v2");
  assert.notEqual(versionRaiz, "20261008-analisis-confirmado-v2");
  assert.notEqual(versionRaiz, "20261008-bolsa-global-v2");
  assert.notEqual(versionRaiz, "20261008-alta-analisis-bolsa-v4");
  assert.notEqual(versionRaiz, "20261008-r-traza-idioma-v1");
  assert.notEqual(versionRaiz, "20261008-r-personas-fichas-v3");
  assert.equal(versionDe(cache, "/portal-empleado/portal.js"), versionRaiz);
  assert.equal(versionDe(html, "/portal-empleado/portal-capacidades.css"), "20261015-t-tabla-situaciones-v1");
  assert.equal(versionDe(html, "/portal-empleado/portal-componentes.css"), "20261015-s-cambio-situacion-v1");
  assert.equal(versionDe(cache, "/portal-empleado/portal-componentes.css"), "20261015-s-cambio-situacion-v1");
  for (const manifiesto of [interno, produccion]) {
    assert.equal(manifiesto.split("static/portal-empleado/portal-bolsas-ruta-filtros.js").length - 1, 1);
  }
});


test("el total global abre una sola página del corte y no consulta ninguna bolsa individual", () => {
  const caso = escenario();
  caso.location.search = `?lang=es&bolsa_global=renuncia&corte_bolsa=${"a".repeat(64)}`;
  assert.equal(caso.aplicar(), true);
  assert.equal(caso.aplicar(), true);
  assert.deepEqual(caso.llamadas, []);
  assert.equal(caso.globales.length, 1);
  assert.equal(caso.globales[0][0], "renuncia");
  assert.equal(caso.globales[0][1].corte, "a".repeat(64));
  caso.location.search = `?lang=es&bolsa_global=disponible&corte_bolsa=${"a".repeat(64)}`;
  caso.aplicar();
  assert.equal(caso.globales.length, 2, "Atrás o filtro nuevo piden solo su página");
});


test("la navegación común mantiene visible la aceptación pendiente, incluida Auditoría", () => {
  const inicioNavegar = portal.indexOf("function navegar(vista, opciones = {})");
  const finNavegar = portal.indexOf("\nfunction ", inicioNavegar + 1);
  assert.ok(inicioNavegar > 0 && finNavegar > inicioNavegar);
  for (const destino of ["auditoria", "portal", "contratacion-temporal"]) {
    let focos = 0, restaurada = "";
    const estado = { vista: "bolsa-candidatos", modalFicha: { resultadoInciertoCambioSituacion: true,
      operacionesB8: { directa: { paso: "incierto" } }, rutaApertura: "/portal-empleado/#bolsa/bolsa-candidatos" } };
    runInNewContext(`${portal.slice(inicioNavegar, finNavegar)}; navegar(destino, { desdeRuta: true })`, {
      estado, destino, document: { querySelector: () => ({ focus: () => { focos++; } }) },
      compensarHistorialGestionOfertas: (ruta) => { restaurada = ruta; },
    });
    assert.equal(estado.vista, "bolsa-candidatos");
    assert.equal(restaurada, estado.modalFicha.rutaApertura);
    assert.equal(focos, 1);
  }
});
