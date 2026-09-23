import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarVistaApariencia } from "./vista-apariencia.js";
import { crearTraductorAdministracion } from "./i18n.js";

class Elemento {
  constructor(documento, etiqueta) {
    this.ownerDocument = documento;
    this.etiqueta = etiqueta;
    this.children = [];
    this.atributos = new Map();
    this.eventos = new Map();
    this.dataset = new Proxy({}, {
      get: (_, clave) => this.getAttribute(`data-${clave}`) ?? undefined,
      set: (_, clave, valor) => { this.setAttribute(`data-${clave}`, String(valor)); return true; },
    });
  }
  append(...hijos) { this.children.push(...hijos); }
  replaceChildren(...hijos) { this.children = hijos; }
  setAttribute(clave, valor) { this.atributos.set(clave, String(valor)); this.ownerDocument.notificarAtributo?.(this, clave); }
  getAttribute(clave) { return this.atributos.get(clave) ?? null; }
  removeAttribute(clave) { this.atributos.delete(clave); this.ownerDocument.notificarAtributo?.(this, clave); }
  addEventListener(clave, fn) { this.eventos.set(clave, fn); }
  emitir(clave) { this.eventos.get(clave)?.({ preventDefault() {} }); }
  buscar(predicado) {
    if (predicado(this)) return this;
    for (const hijo of this.children) {
      const encontrado = hijo.buscar(predicado);
      if (encontrado) return encontrado;
    }
    return null;
  }
}

function documentoFalso() {
  const documento = { createElement(etiqueta) { return new Elemento(documento, etiqueta); } };
  const observadores = new Set();
  documento.defaultView = { MutationObserver: class {
    constructor(callback) { this.callback = callback; this.registros = []; }
    observe(objetivo) { this.objetivo = objetivo; observadores.add(this); }
    takeRecords() { return this.registros.splice(0); }
    disconnect() { observadores.delete(this); }
    anotar(objetivo, nombre) {
      if (objetivo !== this.objetivo || nombre !== "data-tema") return;
      this.registros.push({ type: "attributes", attributeName: nombre });
      queueMicrotask(() => { const registros = this.takeRecords(); if (registros.length) this.callback(registros); });
    }
  } };
  documento.notificarAtributo = (objetivo, nombre) => { for (const observador of observadores) observador.anotar(objetivo, nombre); };
  documento.documentElement = documento.createElement("html");
  documento.body = documento.createElement("body");
  return documento;
}

const cargar = () => import("../../../comun/tema-vec.js");
async function esperarEstado(raiz, estado) {
  for (let i = 0; i < 200; i += 1) {
    if (raiz.buscar((n) => n.dataset.aparienciaEstado === estado)) return;
    await new Promise((resolver) => setTimeout(resolver, 5));
  }
  assert.fail(`Apariencia no alcanzó el estado ${estado}`);
}

test("Apariencia previsualiza y recupera exactamente el tema previo al desmontar", async () => {
  const documento = documentoFalso();
  documento.documentElement.dataset.tema = "granate";
  const raiz = documento.createElement("div");
  const vista = montarVistaApariencia({ raiz, t: crearTraductorAdministracion(), cargarControlador: cargar });
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaEstado === "cargando")?.etiqueta, "section");
  await esperarEstado(raiz, "disponible");
  const radio = raiz.buscar((n) => n.etiqueta === "input" && n.value === "institucional");
  radio.checked = true;
  radio.emitir("change");
  raiz.buscar((n) => n.etiqueta === "form").emitir("submit");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "institucional");
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaPrevia === "true")?.etiqueta, "section");
  const publicar = raiz.buscar((n) => n.etiqueta === "button" && n.textContent?.startsWith("Publicar"));
  assert.equal(publicar.disabled, true);
  assert.match(raiz.buscar((n) => n.className === "administracion-apariencia-limite")?.textContent, /falta autoridad global/u);
  const ayuda = raiz.buscar((n) => n.className === "administracion-apariencia-ayuda");
  assert.equal(ayuda.getAttribute("aria-expanded"), "false");
  ayuda.emitir("click");
  assert.equal(ayuda.getAttribute("aria-expanded"), "true");
  const restablecer = raiz.buscar((n) => n.etiqueta === "button" && n.textContent === "Restablecer");
  restablecer.emitir("click");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");
  assert.equal(raiz.buscar((n) => n.etiqueta === "input" && n.value === "granate")?.checked, true);
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaPrevia === "false")?.etiqueta, "section");
  radio.checked = true;
  radio.emitir("change");
  raiz.buscar((n) => n.etiqueta === "form").emitir("submit");
  vista.desmontar();
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");
  assert.deepEqual(raiz.children, []);
});

test("Apariencia permite reintentar una carga fallida sin aplicar tema", async () => {
  const documento = documentoFalso();
  const raiz = documento.createElement("div");
  let intentos = 0;
  const vista = montarVistaApariencia({ raiz, t: crearTraductorAdministracion(), cargarControlador: () => {
    intentos += 1;
    return intentos === 1 ? Promise.reject(new Error("fallo sintético")) : cargar();
  } });
  await esperarEstado(raiz, "error");
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaEstado === "error")?.etiqueta, "section");
  raiz.buscar((n) => n.etiqueta === "button" && n.textContent === "Reintentar").emitir("click");
  await esperarEstado(raiz, "disponible");
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaEstado === "disponible")?.etiqueta, "section");
  assert.equal(documento.documentElement.getAttribute("data-tema"), null);
  vista.desmontar();
});

test("un cambio externo de tema prevalece al desmontar sin alterar foco ni contraste", async () => {
  const documento = documentoFalso();
  documento.body.dataset.contraste = "true";
  const foco = documento.createElement("button");
  documento.activeElement = foco;
  const raiz = documento.createElement("div");
  const vista = montarVistaApariencia({ raiz, t: crearTraductorAdministracion(), cargarControlador: cargar });
  await esperarEstado(raiz, "disponible");
  const granate = raiz.buscar((n) => n.etiqueta === "input" && n.value === "granate");
  granate.checked = true;
  granate.emitir("change");
  raiz.buscar((n) => n.etiqueta === "form").emitir("submit");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");
  documento.documentElement.dataset.tema = "institucional";
  vista.desmontar();
  assert.equal(documento.documentElement.getAttribute("data-tema"), "institucional");
  assert.equal(documento.body.dataset.contraste, "true");
  assert.equal(documento.activeElement, foco);
});

test("una escritura externa del mismo tema desactiva la preview y una nueva parte del estado visible", async () => {
  const documento = documentoFalso();
  const raiz = documento.createElement("div");
  const vista = montarVistaApariencia({ raiz, t: crearTraductorAdministracion(), cargarControlador: cargar });
  await esperarEstado(raiz, "disponible");
  const granate = raiz.buscar((n) => n.etiqueta === "input" && n.value === "granate");
  granate.checked = true;
  granate.emitir("change");
  const form = raiz.buscar((n) => n.etiqueta === "form");
  form.emitir("submit");
  documento.documentElement.dataset.tema = "granate";
  await Promise.resolve();
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaPrevia === "false")?.etiqueta, "section");
  vista.desmontar();
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");

  const otraRaiz = documento.createElement("div");
  const otraVista = montarVistaApariencia({ raiz: otraRaiz, t: crearTraductorAdministracion(), cargarControlador: cargar });
  await esperarEstado(otraRaiz, "disponible");
  const institucional = otraRaiz.buscar((n) => n.etiqueta === "input" && n.value === "institucional");
  institucional.checked = true;
  institucional.emitir("change");
  otraRaiz.buscar((n) => n.etiqueta === "form").emitir("submit");
  otraRaiz.buscar((n) => n.etiqueta === "button" && n.textContent === "Restablecer").emitir("click");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");
  otraVista.desmontar();
});

test("tras una sustitución externa, otra preview toma como base el tema nuevo", async () => {
  const documento = documentoFalso();
  const raiz = documento.createElement("div");
  const vista = montarVistaApariencia({ raiz, t: crearTraductorAdministracion(), cargarControlador: cargar });
  await esperarEstado(raiz, "disponible");
  const granate = raiz.buscar((n) => n.etiqueta === "input" && n.value === "granate");
  granate.checked = true;
  granate.emitir("change");
  const form = raiz.buscar((n) => n.etiqueta === "form");
  form.emitir("submit");
  documento.documentElement.dataset.tema = "institucional";
  await Promise.resolve();
  assert.equal(raiz.buscar((n) => n.dataset.aparienciaPrevia === "false")?.etiqueta, "section");
  form.emitir("submit");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");
  raiz.buscar((n) => n.etiqueta === "button" && n.textContent === "Restablecer").emitir("click");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "institucional");
  vista.desmontar();
});

test("el import versionado es distinto y la carga real permite previsualizar", async () => {
  const sinVersion = new URL("../../../comun/tema-vec.js", import.meta.url);
  const versionada = new URL("../../../comun/tema-vec.js?v=20260924-f2-web2", import.meta.url);
  assert.notEqual(versionada.href, sinVersion.href);
  const rutaNavegador = new URL("../../../comun/tema-vec.js?v=20260924-f2-web2", "https://vec.example/portal-empleado/modulos/administracion/vista-apariencia.js");
  assert.equal(rutaNavegador.pathname, "/comun/tema-vec.js");
  assert.equal(rutaNavegador.search, "?v=20260924-f2-web2");
  const fuente = await readFile(new URL("./vista-apariencia.js", import.meta.url), "utf8");
  assert.match(fuente, /import\("\.\.\/\.\.\/\.\.\/comun\/tema-vec\.js\?v=20260924-f2-web2"\)/u);

  const documento = documentoFalso();
  const raiz = documento.createElement("div");
  const vista = montarVistaApariencia({ raiz, t: crearTraductorAdministracion() });
  await esperarEstado(raiz, "disponible");
  const granate = raiz.buscar((n) => n.etiqueta === "input" && n.value === "granate");
  granate.checked = true;
  granate.emitir("change");
  raiz.buscar((n) => n.etiqueta === "form").emitir("submit");
  assert.equal(documento.documentElement.getAttribute("data-tema"), "granate");
  vista.desmontar();
  assert.equal(documento.documentElement.getAttribute("data-tema"), null);
});
