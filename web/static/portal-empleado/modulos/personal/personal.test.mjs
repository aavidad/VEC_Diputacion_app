import assert from "node:assert/strict";
import test from "node:test";
import { CAPACIDAD_CONSULTAR_PUESTO } from "./contrato.js";
import { montarModuloPersonal } from "./vista.js";

function raizFalsa() {
  class Nodo {
    constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
    remove() { this.parent?.removeChild(this); }
    addEventListener(tipo, manejador) { this.listeners.set(tipo, manejador); }
    setAttribute(clave, valor) { this.atributos.set(clave, valor); }
    matches(selector) { const clave = selector.match(/^\[data-([a-z-]+)\]$/u)?.[1]?.replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase()); return clave ? this.dataset[clave] !== undefined : false; }
    querySelector(selector) { if (this.matches(selector)) return this; for (const hijo of this.children) { const encontrado = hijo.querySelector(selector); if (encontrado) return encontrado; } return null; }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root");
}
function pagina(items = []) { return Object.freeze({ items: Object.freeze(items), total: items.length, limit: 25, offset: 0, catalogo: Object.freeze({ catalogo_id: "categorias-profesionales", catalogo_version: 1, catalogo_huella_sha256: "a".repeat(64) }), fuente: Object.freeze({ revision: "demo-v1", actualizada_en: "2026-09-20T08:00:00Z", demostracion: true, aviso: "DEMOSTRACIÓN pendiente de validación RRHH." }) }); }
const categoria = Object.freeze({ name: "Administrativo", area_etiqueta: "Administración general", state: "Demostración pendiente de validación RRHH" });

test("Personal muestra solo el catálogo RRHH autorizado, su demostración y su límite RPT", async () => {
  const raiz = raizFalsa(); const llamadas = [];
  const modulo = await montarModuloPersonal({ raiz, cliente: { async listarCategorias(consulta, { signal }) { llamadas.push({ consulta, signal }); return pagina([categoria]); } } });
  assert.equal(modulo.capacidad, CAPACIDAD_CONSULTAR_PUESTO); assert.equal(llamadas.length, 1); assert.deepEqual(llamadas[0].consulta, { q: "", area: "", limit: 25, offset: 0 });
  const texto = raiz.querySelector("[data-personal-categorias]").children.map((n) => n.textContent).join(" ");
  assert.match(texto, /demostracion:true/); assert.match(texto, /no es una RPT aprobada/); assert.doesNotMatch(texto, /nómina|servicios/i); modulo.desmontar(); assert.equal(raiz.querySelector("[data-personal-categorias]"), null);
});

test("el catálogo usa tabla semántica y rótulos localizados", async () => {
  const raiz = raizFalsa(); await montarModuloPersonal({ raiz, cliente: { async listarCategorias() { return pagina([categoria]); } } });
  const tabla = raiz.querySelector("[data-personal-categorias]").children.find((n) => n.tagName === "table");
  assert.equal(tabla.children[0].tagName, "caption"); assert.equal(tabla.children[1].tagName, "thead"); assert.equal(tabla.children[1].children[0].children[0].tagName, "th"); assert.equal(tabla.children[1].children[0].children[0].atributos.get("scope"), "col");
});

test("Personal cierra la pantalla sin conservar filas tras un error", async () => {
  const raiz = raizFalsa(); const avisos = [];
  await montarModuloPersonal({ raiz, anunciar: (...argumentos) => avisos.push(argumentos), cliente: { async listarCategorias() { throw new Error("503"); } } });
  const texto = raiz.querySelector("[data-personal-categorias]").children.map((n) => n.textContent).join(" ");
  assert.match(texto, /No se pudo consultar/); assert.equal(avisos.length, 1); assert.doesNotMatch(texto, /Administrativo/); assert.ok(raiz.querySelector("[data-personal-categorias]").children.some((n) => n.tagName === "form"));
});

test("desmontar aborta la consulta y no deja DOM tardío", async () => {
  const raiz = raizFalsa(); let resolver; let signal; let desmontar; const pendiente = new Promise((resolve) => { resolver = resolve; });
  const montaje = montarModuloPersonal({ raiz, registrarDesmontar: (limpiar) => { desmontar = limpiar; }, cliente: { listarCategorias(_consulta, opciones) { signal = opciones.signal; return pendiente; } } });
  await Promise.resolve(); const contenedor = raiz.querySelector("[data-personal-categorias]");
  // La limpieza se registra antes de esperar la respuesta y puede cancelar la navegación.
  assert.ok(contenedor); desmontar(); resolver(pagina([categoria])); const modulo = await montaje; modulo.desmontar(); assert.equal(signal.aborted, true); assert.equal(raiz.querySelector("[data-personal-categorias]"), null);
});
