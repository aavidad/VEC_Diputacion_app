import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

const portal = await readFile(new URL("./portal.js", import.meta.url), "utf8");
const esperar = async () => {
  for (let i = 0; i < 4; i += 1) await new Promise((resolver) => setImmediate(resolver));
};
function diferido() {
  let resolver;
  let rechazar;
  const promesa = new Promise((si, no) => { resolver = si; rechazar = no; });
  return { promesa, resolver, rechazar };
}

function montaje(cargarVista) {
  const inicial = portal.slice(portal.indexOf("let vistaInscripcionesBolsa ="),
    portal.indexOf("const clientePoliticaCese ="));
  const funcion = portal.slice(portal.indexOf("function montarVistaBolsa("),
    portal.indexOf("function aplicarRutaCandidatosBolsa("));
  // Sustituimos únicamente el transporte del import; se ejecuta el montaje
  // publicado para comprobar las carreras de navegación del shell.
  const referencia = 'import("./modulos/bolsa/inscripcion-rrhh-vista.js?v=20261010-b1-inscripcion-b99-v1")';
  assert.equal(funcion.split(referencia).length, 2);
  const estado = { vista: "solicitudes", solicitudes: ["anterior"] };
  let reintentar;
  const raiz = {
    innerHTML: "",
    querySelector: () => ({ addEventListener: (_tipo, fn) => { reintentar = fn; } }),
  };
  const contenedor = { innerHTML: "", querySelector: () => raiz };
  const errores = [];
  const api = runInNewContext(`${inicial}\n${funcion.replace(referencia, "cargarVista()")}
    ({ montar: () => montarVistaBolsa("solicitudes", contenedor), desmontar: desmontarInscripcionesBolsa })`, {
    AbortController, estado, contenedor, cargarVista,
    textoPortal: (clave) => clave,
    console: { error: (...args) => errores.push(args) },
  });
  return { ...api, estado, raiz, errores, reintentar: () => reintentar() };
}

test("abrir Solicitudes monta una vez y no consulta el cuadro de bolsas", async () => {
  let cargas = 0;
  let montajes = 0;
  const vista = montaje(async () => {
    cargas += 1;
    return { montarInscripcionesRRHH: async () => { montajes += 1; return { desmontar() {} }; } };
  });
  vista.montar();
  vista.montar();
  await esperar();
  assert.equal(cargas, 1);
  assert.equal(montajes, 1);
  // El VM carece de cliente de bolsas: una consulta previa fallaría la prueba.
});

test("salir antes de descargar la vista impide montar una bandeja tardía", async () => {
  const carga = diferido();
  let montajes = 0;
  const vista = montaje(() => carga.promesa);
  vista.montar();
  vista.estado.vista = "portal";
  vista.desmontar();
  carga.resolver({ montarInscripcionesRRHH: () => { montajes += 1; } });
  await esperar();
  assert.equal(montajes, 0);
});

test("salir mientras se prepara la bandeja aborta y retira su montaje tardío", async () => {
  const parte = diferido();
  let signal;
  let retiradas = 0;
  const vista = montaje(async () => ({ montarInscripcionesRRHH: (opciones) => {
    signal = opciones.signal;
    return parte.promesa;
  } }));
  vista.montar();
  await esperar();
  vista.estado.vista = "portal";
  vista.desmontar();
  assert.equal(signal.aborted, true);
  parte.resolver({ desmontar: () => { retiradas += 1; } });
  await esperar();
  assert.equal(retiradas, 1);
});

test("un fallo de descarga deja reintentar la misma pantalla sin duplicar el montaje", async () => {
  let cargas = 0;
  let montajes = 0;
  const vista = montaje(async () => {
    cargas += 1;
    if (cargas === 1) throw new TypeError("fallo de transporte");
    return { montarInscripcionesRRHH: async () => { montajes += 1; return { desmontar() {} }; } };
  });
  vista.montar();
  await esperar();
  assert.match(vista.raiz.innerHTML, /role="alert"/u);
  assert.match(vista.raiz.innerHTML, /data-inscripciones-montaje-reintentar/u);
  assert.equal(vista.errores.length, 1);
  assert.equal(vista.errores[0][1].tipo, "TypeError");
  vista.reintentar();
  await esperar();
  assert.equal(cargas, 2);
  assert.equal(montajes, 1);
});

test("el menú y el shell usan la misma URI nueva y empaquetan sólo el cliente RRHH interno", async () => {
  const [html, cache, interno, produccion, publico] = await Promise.all([
    readFile(new URL("./index.html", import.meta.url), "utf8"),
    readFile(new URL("./cache-publica-v1.json", import.meta.url), "utf8"),
    readFile(new URL("../../interno.manifest", import.meta.url), "utf8"),
    readFile(new URL("../../produccion.manifest", import.meta.url), "utf8"),
    readFile(new URL("../../publico.manifest", import.meta.url), "utf8"),
  ]);
  const recurso = /portal-menu-bolsa\.js\?v=([\w-]+)/u;
  assert.equal(portal.match(recurso)[1], html.match(recurso)[1]);
  assert.equal(html.match(/portal\.js\?v=([\w-]+)/u)[1], cache.match(/portal\.js\?v=([\w-]+)/u)[1]);
  for (const ruta of [
    "static/portal-empleado/modulos/bolsa/inscripcion-rrhh-cliente.js",
    "static/portal-empleado/modulos/bolsa/inscripcion-rrhh-vista.js",
    "static/textos/es/bolsa-inscripcion-rrhh.json",
    "static/textos/en/bolsa-inscripcion-rrhh.json",
  ]) {
    assert.ok(interno.split("\n").includes(ruta));
    assert.ok(produccion.split("\n").includes(ruta));
    assert.ok(!publico.split("\n").includes(ruta));
  }
});
