import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import test from "node:test";

const fuente = await readFile(new URL("portal-arranque-aviso.js", import.meta.url), "utf8");
const textosES = JSON.parse(await readFile(new URL("../textos/es/portal-arranque.json", import.meta.url), "utf8"));
const textosEN = JSON.parse(await readFile(new URL("../textos/en/portal-arranque.json", import.meta.url), "utf8"));

function escenario({ url = "http://127.0.0.1/portal-empleado/", fallar = "", colgar = "", excesivo = "" } = {}) {
  const listeners = new Map();
  const peticiones = [];
  const errores = [];
  let recargas = 0;
  let siguienteTiempo = 0;
  let resolverPendiente;
  let avisarMutacion;
  const temporizadores = new Map();
  const crearElemento = (tagName) => ({
    tagName: tagName.toUpperCase(), dataset: {}, hidden: false, textContent: "",
    setAttribute() {},
    addEventListener(tipo, accion) { this[tipo] = accion; },
    append(...hijos) { this.children = hijos; },
  });
  const raiz = {
    childElementCount: 0,
    append(elemento) { this.aviso = elemento; },
    contains(elemento) { return this.aviso === elemento; },
  };
  const foco = {};
  const documento = { documentElement: { lang: "es" }, activeElement: foco,
    getElementById: () => raiz, createElement: crearElemento };
  const localizacion = { href: url, reload() { recargas += 1; } };
  const datos = { "/textos/idiomas.json": { por_defecto: "es", seguir_navegador: true,
    idiomas: [{ codigo: "es" }, { codigo: "en" }] },
  "/textos/es/portal-arranque.json": textosES, "/textos/en/portal-arranque.json": textosEN };
  const ventana = { addEventListener(tipo, accion) { listeners.set(tipo, accion); } };
  const entorno = {
    document: documento, location: localizacion, navigator: { languages: ["es-ES"] }, URL,
    AbortController, TextEncoder, TextDecoder, Uint8Array,
    MutationObserver: class { constructor(accion) { avisarMutacion = accion; } observe() {} disconnect() {} },
    window: ventana,
    console: { error: (...argumentos) => errores.push(argumentos) },
    setTimeout(accion, milisegundos) {
      const id = ++siguienteTiempo;
      temporizadores.set(id, { accion, milisegundos });
      return id;
    },
    clearTimeout(id) { temporizadores.delete(id); },
    async fetch(ruta, opciones) {
      peticiones.push({ ruta, opciones });
      if (ruta === colgar) return new Promise((resolver) => { resolverPendiente = resolver; });
      if (ruta === fallar) return { ok: false };
      if (ruta === excesivo) return { ok: true, async text() { return "x".repeat(128 * 1024 + 1); } };
      return { ok: true, async text() { return JSON.stringify(datos[ruta]); } };
    },
  };
  runInNewContext(fuente, entorno, { filename: "portal-arranque-aviso.js" });
  return {
    raiz, documento, peticiones, errores, listeners, ventana,
    expirar(milisegundos = 12_000) {
      const encontrado = [...temporizadores].find(([, valor]) => valor.milisegundos === milisegundos);
      assert.ok(encontrado, `temporizador de ${milisegundos} ms disponible`);
      temporizadores.delete(encontrado[0]);
      encontrado[1].accion();
    },
    resolverColgada(valor) { resolverPendiente?.({ ok: true, async text() { return JSON.stringify(valor); } }); },
    retirar() { raiz.aviso = null; avisarMutacion(); },
    foco,
    recargas: () => recargas,
    async terminarCarga() { for (let i = 0; i < 10; i += 1) await new Promise((seguir) => setImmediate(seguir)); },
  };
}

function campos(prueba) {
  const [cabecera, cuerpo] = prueba.raiz.aviso.children;
  return [cabecera.children[0], ...cuerpo.children];
}

test("el aviso autónomo sobrevive a un fallo permanente de portal.json y al import común", async () => {
  assert.doesNotMatch(fuente, /^import\s+.*from\s/u, "el aviso no importa el grafo común al evaluarse");
  const prueba = escenario({ fallar: "/textos/es/portal.json" });
  const [titulo, detalle, boton] = campos(prueba);
  assert.equal(prueba.listeners.has("error"), true, "registra el fallo antes de consultar catálogos");
  prueba.listeners.get("error")({ target: {
    tagName: "SCRIPT", type: "module", src: "http://127.0.0.1/portal-empleado/portal.js",
  } });
  await prueba.terminarCarga();
  assert.equal(titulo.textContent, textosES.titulo_error);
  assert.equal(detalle.textContent, textosES.error);
  assert.equal(boton.textContent, textosES.reintentar);
  assert.equal(boton.hidden, false);
  assert.deepEqual(prueba.errores.map(([, datos]) => datos.codigo), ["importacion_portal_no_disponible"]);
  assert.ok(prueba.peticiones.every(({ ruta, opciones }) => ruta.startsWith("/textos/")
    && opciones.mode === "same-origin" && opciones.redirect === "error" && opciones.referrerPolicy === "no-referrer"));
  assert.ok(!prueba.peticiones.some(({ ruta }) => ruta.endsWith("/portal.json")));
  boton.click();
  assert.equal(prueba.recargas(), 1);
});

test("un error de evaluación común muestra causa y reintento sin esperar al temporizador", async () => {
  const prueba = escenario();
  prueba.listeners.get("error")({ target: prueba.ventana });
  await prueba.terminarCarga();
  const [titulo, detalle, boton] = campos(prueba);
  assert.equal(titulo.textContent, textosES.titulo_error);
  assert.equal(detalle.textContent, textosES.error);
  assert.equal(boton.hidden, false);
  assert.deepEqual(prueba.errores.map(([, datos]) => datos.codigo), ["evaluacion_portal_no_disponible"]);
});

test("el aviso usa la URL y cae al idioma por defecto si falla su catálogo", async () => {
  const prueba = escenario({ url: "http://127.0.0.1/portal-empleado/?lang=en",
    fallar: "/textos/en/portal-arranque.json" });
  await prueba.terminarCarga();
  prueba.expirar();
  const [titulo, detalle, boton] = campos(prueba);
  assert.equal(prueba.documento.documentElement.lang, "es");
  assert.equal(titulo.textContent, textosES.titulo_error);
  assert.equal(detalle.textContent, textosES.error);
  assert.equal(boton.textContent, textosES.reintentar);
});

test("índice colgado vence antes del aviso y el catálogo mínimo del DOM pinta el reintento", async () => {
  const prueba = escenario({ colgar: "/textos/idiomas.json" });
  prueba.listeners.get("error")({ target: prueba.ventana });
  assert.equal(campos(prueba)[0].textContent, "");
  prueba.expirar(3_000);
  await prueba.terminarCarga();
  const [titulo, detalle, boton] = campos(prueba);
  assert.equal(prueba.documento.documentElement.lang, "es");
  assert.equal(titulo.textContent, textosES.titulo_error);
  assert.equal(detalle.textContent, textosES.error);
  assert.equal(boton.textContent, textosES.reintentar);
  assert.equal(boton.hidden, false);
  assert.deepEqual(prueba.peticiones.map(({ ruta }) => ruta),
    ["/textos/idiomas.json", "/textos/es/portal-arranque.json"]);
  assert.equal(prueba.peticiones[0].opciones.signal.aborted, true);
});

test("un catálogo superior a 128 KiB se descarta y se lee el respaldo", async () => {
  const prueba = escenario({ url: "http://127.0.0.1/portal-empleado/?lang=en",
    excesivo: "/textos/en/portal-arranque.json" });
  await prueba.terminarCarga();
  assert.equal(prueba.documento.documentElement.lang, "es");
  assert.equal(campos(prueba)[0].textContent, textosES.titulo);
});

test("una respuesta tardía no cambia idioma ni foco tras montar el portal", async () => {
  const prueba = escenario({ url: "http://127.0.0.1/portal-empleado/?lang=en",
    colgar: "/textos/idiomas.json" });
  const [titulo] = campos(prueba);
  prueba.retirar();
  prueba.resolverColgada({ por_defecto: "es", seguir_navegador: true,
    idiomas: [{ codigo: "es" }, { codigo: "en" }] });
  await prueba.terminarCarga();
  assert.equal(prueba.documento.documentElement.lang, "es");
  assert.equal(prueba.documento.activeElement, prueba.foco);
  assert.equal(titulo.textContent, "");
  assert.deepEqual(prueba.peticiones.map(({ ruta }) => ruta), ["/textos/idiomas.json"]);
});

test("un catálogo mínimo tardío tampoco reescribe el idioma del portal montado", async () => {
  const prueba = escenario({ url: "http://127.0.0.1/portal-empleado/?lang=en",
    colgar: "/textos/en/portal-arranque.json" });
  await prueba.terminarCarga();
  const [titulo] = campos(prueba);
  prueba.retirar();
  prueba.resolverColgada(textosEN);
  await prueba.terminarCarga();
  assert.equal(prueba.documento.documentElement.lang, "es");
  assert.equal(prueba.documento.activeElement, prueba.foco);
  assert.equal(titulo.textContent, "");
});
