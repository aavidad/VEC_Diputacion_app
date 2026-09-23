import assert from "node:assert/strict";
import test from "node:test";
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
  setAttribute(clave, valor) { this.atributos.set(clave, String(valor)); }
  getAttribute(clave) { return this.atributos.get(clave) ?? null; }
  removeAttribute(clave) { this.atributos.delete(clave); }
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
  documento.documentElement = documento.createElement("html");
  documento.body = documento.createElement("body");
  return documento;
}

const cargar = () => import("../../../comun/tema-vec.js");
async function esperarEstado(raiz, estado) {
  for (let i = 0; i < 30; i += 1) {
    if (raiz.buscar((n) => n.dataset.aparienciaEstado === estado)) return;
    await new Promise((resolver) => setImmediate(resolver));
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
