import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaBandejaCircuitoDietas } from "./vista-bandeja-circuito.js";

const dato = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = ""; this.value = ""; this.disabled = false; }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  remove() { this.parent?.removeChild(this); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; } removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); if (nombre === "name") this.name = String(valor); }
  focus() { this.ownerDocument.activeElement = this; }
  matches(selector) { const m = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u); if (!m) return this.tagName === selector; const clave = m[1] === "name" ? "name" : dato(m[1]); const valor = m[1] === "name" ? this.name : this.dataset[clave]; return valor !== undefined && (m[2] === undefined || valor === m[2]); }
  closest(selector) { for (let actual = this; actual; actual = actual.parent) if (actual.matches(selector)) return actual; return null; }
  querySelectorAll(selector) { const salida = []; const visitar = (actual) => { if (actual.matches(selector)) salida.push(actual); actual.children.forEach(visitar); }; visitar(this); return salida; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
const referencia = "dco_1234567890123456789012";
const fila = { referencia, estado: "enviado_pendiente_revision", version: 3, fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21" };
const esperar = async () => { await Promise.resolve(); await Promise.resolve(); };

test("carga la bandeja conectada, muestra las dos decisiones y conserva el recibo sólo tras POST válido", async () => {
  const contenedor = raiz(); const llamadas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: async () => ({ items: [fila] }), decidir: async (_ref, entrada) => { llamadas.push(entrada); return { comision: { referencia, estado: "pendiente_autorizacion", version: 4 }, recibo: { referencia: "rcd_1234567890123456789012", version: 4, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false } }; } }, generarClaveIdempotencia: () => "decision-circuito-0001" });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]"); const aprobar = panel.querySelector('[data-dietas-circuito-decision="aprobar"]');
  await panel.listeners.click({ target: aprobar });
  assert.equal(llamadas[0].clave_idempotencia, "decision-circuito-0001"); assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /Recibo/u); vista.desmontar();
});

test("un resultado incierto conserva clave y decisión para el reintento exacto", async () => {
  const contenedor = raiz(); const entradas = []; let intento = 0;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: async () => ({ items: [fila] }), decidir: async (_ref, entrada) => { entradas.push(entrada); if (!intento++) { const error = new Error(); error.resultadoIndeterminado = true; throw error; } return { comision: { referencia, estado: "pendiente_autorizacion", version: 4 }, recibo: { referencia: "rcd_1234567890123456789012", version: 4, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: true } }; } }, generarClaveIdempotencia: () => "decision-circuito-0001" });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]"); await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="aprobar"]') }); await esperar(); assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /confirmado/u); const reintento = panel.querySelector("[data-dietas-circuito-reintento]"); assert.ok(reintento); await panel.listeners.click({ target: reintento });
  assert.equal(entradas.length, 2); assert.deepEqual(entradas[1], entradas[0]); vista.desmontar();
});

test("desmontar aborta la lectura y descarta su respuesta tardía", async () => {
  const contenedor = raiz(); let signal; let resolver;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: (_consulta, opciones) => { signal = opciones.signal; return new Promise((resolve) => { resolver = resolve; }); }, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); vista.desmontar(); assert.equal(signal.aborted, true); resolver({ items: [fila] }); await esperar(); assert.equal(contenedor.children.length, 0);
});
