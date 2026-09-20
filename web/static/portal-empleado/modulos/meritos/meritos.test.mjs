import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { MENSAJES_MERITOS_ES, crearTraductorMeritos } from "./i18n.js";
import { montarVistaMeritos } from "./vista.js";

function raizDePrueba() {
  class Nodo {
    constructor(documento, etiqueta = "div") {
      this.ownerDocument = documento;
      this.tagName = etiqueta.toUpperCase();
      this.children = [];
      this.dataset = {};
      this.listeners = new Map();
      this.attributes = new Map();
      this.parentElement = null;
      this.textContent = "";
      this.disabled = false;
      this.value = "";
    }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parentElement = this; }); }
    remove() { this.parentElement && (this.parentElement.children = this.parentElement.children.filter((nodo) => nodo !== this)); }
    setAttribute(nombre, valor) { this.attributes.set(nombre, String(valor)); }
    getAttribute(nombre) { return this.attributes.get(nombre) ?? null; }
    removeAttribute(nombre) { this.attributes.delete(nombre); }
    addEventListener(tipo, oyente) { this.listeners.set(tipo, oyente); }
    removeEventListener(tipo, oyente) { if (this.listeners.get(tipo) === oyente) this.listeners.delete(tipo); }
  }
  const documento = { createElement(etiqueta) { return new Nodo(documento, etiqueta); } };
  const raiz = new Nodo(documento, "main");
  raiz.querySelector = (selector) => raiz.children.find((nodo) => selector === "[data-modulo-meritos]" && nodo.dataset.moduloMeritos !== undefined) ?? null;
  return raiz;
}

test("el catálogo de Méritos es español y el traductor queda cerrado", () => {
  const t = crearTraductorMeritos();
  assert.ok(Object.isFrozen(MENSAJES_MERITOS_ES));
  assert.equal(t("titulo"), "Méritos y formación");
  assert.match(t("sin_conexion"), /sin API|sin red/i);
  assert.throws(() => crearTraductorMeritos({ titulo: "incompleto" }), /incompleto/i);
  assert.throws(() => t("clave_inexistente"), /clave_inexistente/i);
});

test("Méritos presenta información sintética, fuentes y límites de conexión", () => {
  const raiz = raizDePrueba();
  montarVistaMeritos({ raiz });
  const vista = raiz.querySelector("[data-modulo-meritos]");
  assert.ok(vista);
  assert.match(vista.innerHTML, /Antonio López Fernández/);
  assert.match(vista.innerHTML, /datos sintéticos/i);
  assert.match(vista.innerHTML, /sin conexión|pendiente de conexión/i);
  assert.match(vista.innerHTML, /Formación|Méritos/i);
  assert.doesNotMatch(vista.innerHTML, /usuario de prueba|demo-user/i);
});

test("Méritos navega localmente entre vistas, filtra sin efectos y conserva acciones deshabilitadas", () => {
  const raiz = raizDePrueba();
  montarVistaMeritos({ raiz });
  const vista = raiz.querySelector("[data-modulo-meritos]");
  assert.match(vista.innerHTML, /data-meritos-vista="formacion"/);
  vista.listeners.get("click")({ target: { closest: () => ({ dataset: { meritosVista: "formacion" }, textContent: "Formación disponible" }) } });
  assert.match(vista.innerHTML, /Formación disponible/);
  assert.match(vista.innerHTML, /aria-selected="true"/);
  assert.match(vista.innerHTML, /data-meritos-filtro/);
  vista.listeners.get("change")({ target: { matches: () => true, value: "pendiente" } });
  assert.match(vista.innerHTML, /0 registros visibles/);
  assert.match(vista.innerHTML, /<button type="button" disabled aria-disabled="true">Inscribirme<\/button>/);
});

test("Méritos registra desmontaje, anuncia el contexto y no introduce red ni almacenamiento", async () => {
  const raiz = raizDePrueba();
  const anuncios = [];
  let desmontajeRegistrado;
  const vista = montarVistaMeritos({ raiz, anunciar: (mensaje) => anuncios.push(mensaje), registrarDesmontar: (desmontar) => { desmontajeRegistrado = desmontar; } });
  assert.equal(typeof vista.desmontar, "function");
  assert.equal(desmontajeRegistrado, vista.desmontar);
  raiz.querySelector("[data-modulo-meritos]").listeners.get("click")({ target: { closest: () => ({ dataset: { meritosVista: "formacion" }, textContent: "Formación disponible" }) } });
  assert.ok(anuncios.some((mensaje) => /m[eé]rito|formaci[oó]n/i.test(mensaje)));
  vista.desmontar();
  assert.equal(raiz.querySelector("[data-modulo-meritos]"), null);
  const fuente = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie/i);
});

test("la vista de Méritos consume el traductor cerrado", async () => {
  const fuente = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.match(fuente, /from "\.\/i18n\.js"/);
  assert.match(fuente, /crearTraductorMeritos/);
  assert.match(fuente, /\bt\(/);
});
