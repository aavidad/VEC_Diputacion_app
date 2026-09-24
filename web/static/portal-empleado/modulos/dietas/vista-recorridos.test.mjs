import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaRecorridosDietas } from "./vista-recorridos.js";

const claveDato = (atributo) => atributo.slice(5).replace(/-([a-z])/gu, (_todo, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
    this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = ""; this.disabled = false;
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  remove() { this.parent?.removeChild(this); }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  removeAttribute(nombre) { delete this.attrs[nombre]; }
  addEventListener(tipo, manejador) { this.listeners[tipo] = manejador; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  focus() { this.ownerDocument.activeElement = this; }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    if (!coincidencia) return false;
    const atributo = coincidencia[1];
    const valor = atributo.startsWith("data-") ? this.dataset[claveDato(atributo)] : this.attrs[atributo];
    return valor !== undefined && (coincidencia[2] === undefined || valor === coincidencia[2]);
  }
  closest(selector) { for (let actual = this; actual; actual = actual.parent) if (actual.matches(selector)) return actual; return null; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) { const salida = []; const visitar = (actual) => { if (actual.matches(selector)) salida.push(actual); actual.children.forEach(visitar); }; visitar(this); return salida; }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
function texto(nodo) { return [nodo.textContent, ...nodo.children.map(texto)].join(" "); }
const item = Object.freeze({
  comision: { referencia: "dco_1234567890123456789012", estado: "borrador", fecha_inicio: "2026-09-24",
    fecha_fin: "2026-09-25", motivo: "Visita al centro", codigos_ruta: [], relacion_ref: "rel_1234567890123456789012" },
  recibo: { referencia: "rcd_1234567890123456789012", version: 1, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false },
});
const clienteBorradores = () => ({ listar: async () => ({ items: [item] }), obtener: async () => item, crear: async () => item });

test("Dietas monta el recorrido interno y abre el formulario real sin itinerario de presentación", async () => {
  const contenedor = raiz();
  const vista = montarVistaRecorridosDietas(contenedor, { clienteBorradores: clienteBorradores() });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  const formulario = contenedor.querySelector("[data-dietas-borrador-form]");
  assert.ok(panel); assert.ok(formulario);
  assert.equal(formulario.hidden, true);
  assert.equal(contenedor.querySelector("[data-dietas-area-itinerario]"), null);
  assert.equal(contenedor.querySelector("[data-dietas-mapa-pendiente]"), null);
  assert.equal(panel.querySelectorAll("[data-dietas-cambiar-etapa]").length, 6);
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-abrir-nueva-comision]") });
  assert.equal(formulario.hidden, false);
  assert.equal(panel.querySelector("[data-dietas-abrir-nueva-comision]").attrs["aria-expanded"], "true");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-abrir-nueva-comision]") });
  assert.equal(formulario.hidden, true);
  vista.desmontar();
  assert.equal(contenedor.querySelector("[data-dietas-recorridos]"), null);
});

test("la bandeja propia consulta GET y conserva el recibo real en la ficha", async () => {
  const contenedor = raiz(); let consultas = 0;
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: { ...clienteBorradores(), obtener: async () => { consultas += 1; return item; } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  const boton = contenedor.querySelector("[data-dietas-borrador-detalle]");
  await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({ target: boton });
  assert.equal(consultas, 1);
  const recibo = contenedor.querySelector("[data-dietas-borrador-recibo]");
  assert.ok(recibo);
  assert.match(texto(recibo), /rcd_1234567890123456789012/u);
  assert.equal(panel.querySelector("[data-dietas-panel-etapa=\"circuito\"]").hidden, true);
  vista.desmontar();
});

test("al cambiar de bandeja o salir aborta lecturas pendientes y no conserva una etapa ajena", async () => {
  const contenedor = raiz(); const consultas = [];
  const clienteCircuito = {
    listar: (_consulta, { signal }) => new Promise((_resolver, rechazar) => {
      consultas.push(signal);
      signal.addEventListener("abort", () => rechazar(Object.assign(new Error("cancelada"), { codigo: "operacion_abortada" })), { once: true });
    }),
    decidir: async () => { throw new Error("no debe decidir"); },
  };
  const vista = montarVistaRecorridosDietas(contenedor, { clienteBorradores: clienteBorradores(), clienteCircuito });
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-cambiar-etapa="revision"]') });
  assert.equal(consultas.length, 1);
  assert.ok(panel.querySelector("[data-dietas-bandeja-circuito]"));
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-cambiar-etapa="autorizacion"]') });
  assert.equal(consultas[0].aborted, true);
  assert.equal(consultas.length, 2);
  vista.desmontar();
  assert.equal(consultas[1].aborted, true);
  assert.equal(contenedor.querySelector("[data-dietas-recorridos]"), null);
});
