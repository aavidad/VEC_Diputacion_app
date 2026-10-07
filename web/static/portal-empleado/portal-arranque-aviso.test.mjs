import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import test from "node:test";

const fuente = await readFile(new URL("portal-arranque-aviso.js", import.meta.url), "utf8");
const textosES = JSON.parse(await readFile(new URL("../textos/es/portal-arranque.json", import.meta.url), "utf8"));
const textosEN = JSON.parse(await readFile(new URL("../textos/en/portal-arranque.json", import.meta.url), "utf8"));

function escenario({ url = "http://127.0.0.1/portal-empleado/", fallar = "" } = {}) {
  const listeners = new Map();
  const peticiones = [];
  const errores = [];
  let recargas = 0;
  let tiempo;
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
  const documento = { documentElement: { lang: "es" }, getElementById: () => raiz, createElement: crearElemento };
  const localizacion = { href: url, reload() { recargas += 1; } };
  const datos = { "/textos/idiomas.json": { por_defecto: "es", seguir_navegador: true,
    idiomas: [{ codigo: "es" }, { codigo: "en" }] },
  "/textos/es/portal-arranque.json": textosES, "/textos/en/portal-arranque.json": textosEN };
  const ventana = { addEventListener(tipo, accion) { listeners.set(tipo, accion); } };
  const entorno = {
    document: documento, location: localizacion, navigator: { languages: ["es-ES"] }, URL,
    MutationObserver: class { observe() {} disconnect() {} },
    window: ventana,
    console: { error: (...argumentos) => errores.push(argumentos) },
    setTimeout(accion) { tiempo = accion; return 1; }, clearTimeout() {},
    async fetch(ruta, opciones) {
      peticiones.push({ ruta, opciones });
      if (ruta === fallar) return { ok: false };
      return { ok: true, async json() { return datos[ruta]; } };
    },
  };
  runInNewContext(fuente, entorno, { filename: "portal-arranque-aviso.js" });
  return {
    raiz, documento, peticiones, errores, listeners, ventana, expirar: () => tiempo(),
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
