import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { montarModuloEstructuraOrganizativaPublica } from "./vista-estructura-organizativa-publica.js";

test("estructura carga el i18n actualizado del corte F2", () => {
  const codigo = readFileSync(new URL("./vista-estructura-organizativa-publica.js", import.meta.url), "utf8");
  assert.match(codigo, /from "\.\/i18n\.js\?v=20260924-f2-web2";/u);
});

function raiz() {
  class Nodo {
    constructor(documento, etiqueta = "div") {
      this.ownerDocument = documento;
      this.tagName = etiqueta;
      this.children = [];
      this.dataset = {};
      this.parent = null;
      this.textContent = "";
      this.atributos = new Map();
      this.listeners = {};
    }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    removeChild(n) { this.children = this.children.filter((h) => h !== n); n.parent = null; }
    remove() { this.parent?.removeChild(this); }
    setAttribute(k, v) { this.atributos.set(k, String(v)); }
    addEventListener(tipo, listener) { this.listeners[tipo] = listener; }
    matches(selector) {
      const clave = selector.match(/^\[data-([a-z-]+)\]$/u)?.[1]?.replace(/-([a-z])/gu, (_m, l) => l.toUpperCase());
      return clave ? this.dataset[clave] !== undefined : this.tagName === selector;
    }
    querySelector(selector) {
      if (this.matches(selector)) return this;
      for (const hijo of this.children) {
        const encontrado = hijo.querySelector(selector);
        if (encontrado) return encontrado;
      }
      return null;
    }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) };
  return new Nodo(documento, "root");
}
function textoVisible(nodo) {
  return nodo.hidden ? "" : [nodo.textContent, ...nodo.children.map(textoVisible)].join(" ");
}
function estructura() {
  const unidades = [];
  for (let i = 0; i < 14; i += 1) unidades.push({ clave: `delegacion-${i}`, etiqueta: "Delegación", tipo: "delegacion" });
  for (let i = 0; i < 41; i += 1) unidades.push({ clave: `centro-${i}`, etiqueta: "Centro", tipo: "centro", adscripcion_clave: "delegacion-0" });
  for (let i = 0; i < 11; i += 1) unidades.push({ clave: `puesto-${i}`, etiqueta: "Jefatura", tipo: "puesto_responsabilidad", adscripcion_clave: "centro-0" });
  return {
    unidades,
    fuente: {
      revision: "demo-v1", actualizada_en: "2026-09-06T00:00:00Z",
      demostracion: true, aviso: "Demo sin vigencia",
      huella_sha256: "0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c",
    },
  };
}

test("la ayuda queda detrás de ? y la fuente DEMO sigue visible", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  const boton = vista.querySelector("[data-personal-estructura-ayuda]");
  const contenido = vista.querySelector("[data-personal-estructura-ayuda-contenido]");
  assert.equal(boton.tagName, "button");
  assert.equal(boton.textContent, "?");
  assert.match(boton.atributos.get("aria-label"), /Consulta pública/u);
  assert.equal(boton.atributos.get("aria-expanded"), "false");
  assert.equal(contenido.hidden, true);
  assert.match(textoVisible(vista), /Fuente DEMO.*no acredita vigencia administrativa/u);
  assert.match(textoVisible(vista), /14 delegaciones/u);
  assert.doesNotMatch(textoVisible(vista), /Huella SHA-256|demo-v1|2026-09-06T00:00:00Z/u);
  boton.listeners.click();
  assert.equal(contenido.hidden, false);
  assert.equal(boton.atributos.get("aria-expanded"), "true");
  assert.match(textoVisible(vista), /demo-v1/u);
  assert.match(textoVisible(vista), /0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c/u);
  assert.doesNotMatch(textoVisible(vista), /2026-09-06T00:00:00Z/u);
  boton.listeners.click();
  assert.equal(contenido.hidden, true);
  modulo.desmontar();
});

test("la tabla prioriza nombre y adscripción legible, conserva clave y scroll interno", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  const tabla = vista.children.find((n) => n.className === "tabla-contenedor");
  assert.equal(tabla.atributos.get("role"), "region");
  assert.equal(tabla.atributos.get("tabindex"), "0");
  const cuerpo = tabla.querySelector("tbody");
  assert.equal(cuerpo.children.length, 66);
  const centro = cuerpo.children[14];
  assert.equal(centro.children[0].tagName, "th");
  assert.equal(centro.children[0].textContent, "Centro");
  assert.equal(centro.children[0].atributos.get("scope"), "row");
  assert.equal(centro.children[2].textContent, "Delegación");
  assert.equal(textoVisible(centro.children.at(-1)), " centro-0");
  modulo.desmontar();
});

test("error y desmontaje abortan sin pintar respuesta tardía", async () => {
  const r = raiz();
  const avisos = [];
  await montarModuloEstructuraOrganizativaPublica({ raiz: r, anunciar: (...aviso) => avisos.push(aviso), cliente: { async obtener() { throw new Error("503"); } } });
  assert.equal(avisos.length, 1);
  let resolver;
  let signal;
  let limpiar;
  const otraRaiz = raiz();
  const montaje = montarModuloEstructuraOrganizativaPublica({ raiz: otraRaiz, registrarDesmontar: (fn) => { limpiar = fn; }, cliente: { obtener({ signal: s }) { signal = s; return new Promise((resolve) => { resolver = resolve; }); } } });
  await Promise.resolve();
  limpiar();
  resolver(estructura());
  const modulo = await montaje;
  modulo.desmontar();
  assert.equal(signal.aborted, true);
  assert.equal(otraRaiz.querySelector("[data-personal-estructura-organizativa-publica]"), null);
});
