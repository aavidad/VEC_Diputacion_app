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
    querySelector(selector) { if (selector === "[data-meritos-filtro]" || selector.startsWith('[data-meritos-vista=') || selector.startsWith('[data-meritos-detalle=')) return { focus() {} }; return null; }
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
  assert.doesNotMatch(renderizarMeritos({ estado: "denegado", meritos: datos.meritos }), /Grado en Derecho/);
  assert.match(renderizarMeritos({ estado: "error" }), /La consulta falló/);
  assert.match(renderizarMeritos({ estado: "cargando" }), /Esperando una respuesta/);
});

test("el inventario resume hechos y el detalle contiene fuente, evidencia y vigencia", () => {
  const html = renderizarMeritos(datos);
  assert.match(html, /Grado en Derecho/);
  assert.match(html, /aria-expanded="false"/);
  assert.doesNotMatch(html, /Documento aportado/);
  assert.match(html, /Declarado/);
  assert.match(html, /Acreditados/);
  assert.match(html, /<button type="button" disabled aria-disabled="true"/);
  assert.doesNotMatch(html, /Bolsa A|1,5/);
  const detalle = renderizarMeritos(datos, { seleccionado: 0 });
  assert.match(detalle, /aria-expanded="true"/);
  assert.match(detalle, /Documento aportado/);
  assert.match(detalle, /Sin caducidad informada/);
  assert.match(detalle, /Persona/);
  assert.match(detalle, /colspan="3"/);
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
  const html = renderizarMeritos({ estado: "disponible", meritos: [{ nombre: "<script>alert(1)</script>", evidencia: "<img src=x>", estado: "acreditado" }], requisitos: [{ requisito: "Título", resultado: "pendiente" }], valoraciones: [{ puntos: 9 }] }, { seleccionado: 0 });
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /&lt;script&gt;/);
  assert.doesNotMatch(html, /<img/);
  assert.match(html, /&lt;img src=x&gt;/);
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
  const detalle = { padre: nodo, dataset: { meritosDetalle: "0" } };
  nodo.listeners.get("click")({ target: { closest: (selector) => selector === "[data-meritos-detalle]" ? detalle : null } });
  assert.match(nodo.innerHTML, /Documento aportado/);
  nodo.listeners.get("click")({ target: { closest: (selector) => selector === "[data-meritos-detalle]" ? detalle : null } });
  assert.doesNotMatch(nodo.innerHTML, /Documento aportado/);
  let prevenido = false;
  const boton = { padre: nodo, dataset: { meritosVista: "inventario" } };
  nodo.listeners.get("keydown")({ key: "ArrowRight", target: { closest: () => boton }, preventDefault() { prevenido = true; } });
  assert.ok(prevenido);
  assert.match(nodo.innerHTML, /Bolsa A/);
  montaje.actualizar({ estado: "denegado", meritos: datos.meritos });
  assert.match(nodo.innerHTML, /Acceso denegado/);
  assert.doesNotMatch(nodo.innerHTML, /Grado en Derecho|Documento aportado/);
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

test("la vista carga el catálogo i18n de la versión F2", async () => {
  const fuente = await readFile(new URL("vista.js", import.meta.url), "utf8");
  assert.match(fuente, /from "\.\/i18n\.js\?v=20260924-f2-web2"/);
  const modulo = await import("./vista.js?v=20260926-integracion-bolsa-ct-v1");
  assert.equal(typeof modulo.montarVistaMeritos, "function");
  assert.match(modulo.renderizarMeritos(), /Fuente de méritos no conectada/);
});
