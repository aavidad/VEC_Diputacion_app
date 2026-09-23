import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { MENSAJES_MERITOS_ES, crearTraductorMeritos } from "./i18n.js";
import { montarVistaMeritos, renderizarMeritos } from "./vista.js";

function raizDePrueba() {
  class Nodo {
    constructor(documento) { this.ownerDocument = documento; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parentElement = null; this.innerHTML = ""; }
    append(nodo) { this.children.push(nodo); nodo.parentElement = this; }
    remove() { if (this.parentElement) this.parentElement.children = this.parentElement.children.filter((x) => x !== this); this.parentElement = null; }
    addEventListener(tipo, oyente) { this.listeners.set(tipo, oyente); }
    removeEventListener(tipo, oyente) { if (this.listeners.get(tipo) === oyente) this.listeners.delete(tipo); }
    contains(nodo) { return nodo?.padre === this; }
    querySelector(selector) { if (selector === "[data-meritos-filtro]" || selector.startsWith('[data-meritos-vista=')) return { focus() {} }; return null; }
  }
  const documento = { createElement() { return new Nodo(documento); } };
  const raiz = new Nodo(documento);
  raiz.querySelector = () => raiz.children[0] ?? null;
  return raiz;
}

const datos = {
  estado: "disponible",
  meritos: [
    { nombre: "Grado en Derecho", tipo: "Titulación", fuente: "Persona", evidencia: "Documento aportado", vigencia: "Sin caducidad informada", estado: "declarado" },
    { nombre: "Curso de gestión", tipo: "Formación", fuente: "Formación", evidencia: "Certificado", vigencia: "2026", estado: "acreditado" },
  ],
  requisitos: [
    { requisito: "Titulación", convocatoria: "Bolsa A", versionBases: "v2", resultado: "pendiente", motivo: "Evidencia en revisión", procedencia: "Bases publicadas", hito: "Fin de plazo" },
  ],
  valoraciones: [
    { merito: "Curso de gestión", convocatoria: "Proceso B", versionBases: "v4", criterio: "Formación acreditada", puntos: 1.5 },
  ],
};

test("el catálogo español es cerrado", () => {
  const t = crearTraductorMeritos();
  assert.ok(Object.isFrozen(MENSAJES_MERITOS_ES));
  assert.equal(t("titulo"), "Méritos");
  assert.throws(() => crearTraductorMeritos({ titulo: "incompleto" }), /incompleto/);
  assert.throws(() => t("desconocida"), /desconocida/);
});

test("sin conector no se cargan datos sintéticos ni se ofrece una aportación efectiva", () => {
  const html = renderizarMeritos();
  assert.match(html, /Fuente de méritos no conectada/);
  assert.match(html, /No se han cargado datos de ejemplo/);
  assert.doesNotMatch(html, /Antonio López|Grado en Derecho|registros visibles|puntos obtenidos/);
  assert.doesNotMatch(html, /<table/);
  assert.match(renderizarMeritos({ estado: "denegado" }), /Acceso denegado/);
  assert.match(renderizarMeritos({ estado: "error" }), /La consulta falló/);
  assert.match(renderizarMeritos({ estado: "cargando" }), /Esperando una respuesta/);
});

test("el inventario enseña fuente, evidencia, vigencia y estado declarado sin confundirlo con acreditación", () => {
  const html = renderizarMeritos(datos);
  assert.match(html, /Grado en Derecho/);
  assert.match(html, /Documento aportado/);
  assert.match(html, /Sin caducidad informada/);
  assert.match(html, /Declarado/);
  assert.match(html, /Acreditados/);
  assert.match(html, /<button type="button" disabled aria-disabled="true"/);
  assert.doesNotMatch(html, /Bolsa A|1,5/);
});

test("requisitos y valoraciones mantienen convocatoria y versión independientes", () => {
  const requisitos = renderizarMeritos(datos, { pestana: "requisitos" });
  assert.match(requisitos, /Bolsa A/);
  assert.match(requisitos, /Evidencia en revisión/);
  assert.match(requisitos, /Fin de plazo/);
  assert.doesNotMatch(requisitos, /1,5/);
  const valoraciones = renderizarMeritos(datos, { pestana: "valoraciones" });
  assert.match(valoraciones, /Proceso B/);
  assert.match(valoraciones, /v4/);
  assert.match(valoraciones, /1,5/);
  assert.doesNotMatch(valoraciones, /Bolsa A/);
});

test("escapa datos recibidos y no deriva el cumplimiento de los puntos", () => {
  const html = renderizarMeritos({ estado: "disponible", meritos: [{ nombre: "<script>alert(1)</script>", estado: "acreditado" }], requisitos: [{ requisito: "Título", resultado: "pendiente" }], valoraciones: [{ puntos: 9 }] });
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /&lt;script&gt;/);
  assert.match(renderizarMeritos({ estado: "disponible", requisitos: datos.requisitos, valoraciones: datos.valoraciones }, { pestana: "requisitos" }), /Pendiente/);
  assert.throws(() => renderizarMeritos({ estado: "disponible", meritos: {} }), /inválida/);
});

test("montaje, pestañas por teclado, actualización y desmontaje conservan el contrato", () => {
  const raiz = raizDePrueba(); const anuncios = []; let registrado;
  const montaje = montarVistaMeritos({ raiz, anunciar: (mensaje) => anuncios.push(mensaje), registrarDesmontar: (fn) => { registrado = fn; } });
  const nodo = raiz.querySelector();
  assert.match(nodo.innerHTML, /Fuente de méritos no conectada/);
  montaje.actualizar(datos);
  assert.match(nodo.innerHTML, /Grado en Derecho/);
  let prevenido = false;
  const boton = { padre: nodo, dataset: { meritosVista: "inventario" } };
  nodo.listeners.get("keydown")({ key: "ArrowRight", target: { closest: () => boton }, preventDefault() { prevenido = true; } });
  assert.ok(prevenido);
  assert.match(nodo.innerHTML, /Bolsa A/);
  assert.ok(anuncios.length >= 2);
  assert.equal(registrado, montaje.desmontar);
  montaje.desmontar();
  assert.equal(raiz.querySelector(), null);
  montaje.actualizar(datos);
});

test("la vista no abre red ni almacenamiento web", async () => {
  const fuente = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie/i);
  assert.doesNotMatch(fuente, /datos-presentacion|datos-sinteticos/i);
});
