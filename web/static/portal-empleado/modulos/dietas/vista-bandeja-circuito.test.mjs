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
const documentoLeido = {
  referencia, numero_documento: "VEC-D-2026-000012", fecha_apertura: "2026-09-20T08:00:00.000000Z", estado: "enviado_pendiente_revision", version: 3,
  fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", hora_inicio: "08:00", hora_fin: "15:30", motivo: "Reunión técnica", codigos_ruta: [], calculo: {},
  documento: { lineas: [
    { tipo: "dieta", concepto: "manutencion", fecha: "2026-09-20", importe_centimos: 2667 },
    { tipo: "kilometraje", ruta_indice: 1, kilometros: "42.5000", importe_centimos: 1105 },
    { tipo: "otro_gasto", concepto: "Aparcamiento", importe_centimos: 650, justificante_ref: "just:ticket-01", justificante_sha256: "a".repeat(64) },
    { tipo: "otro_medio", tipo_gasto: "taxi", catalogo_version: "provisional:otros-gastos:20260925", fecha: "2026-09-21", concepto: "Estación a sede", importe_centimos: 1250, justificante_ref: "ticket:taxi-02", justificante_sha256: "b".repeat(64) },
  ], manutencion_centimos: 2667, alojamiento_tope_centimos: 0, kilometraje_centimos: 1105, otros_centimos: 1900, total_orientativo_centimos: 5672 },
};
const recibo = { referencia: "rcd_1234567890123456789012", version: 4, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false };
function texto(nodo) { return [nodo.textContent, ...nodo.children.map(texto)].join(" "); }
const esperar = async () => { await Promise.resolve(); await Promise.resolve(); };

test("sin fuente de competencia lo dice en una línea y no ofrece acciones", async () => {
  const contenedor = raiz();
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: async () => ({ items: [], competencia: "sin_fuente" }), decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.match(panel.querySelector("[data-dietas-circuito-sin-fuente]").textContent, /validadores/u);
  assert.equal(panel.querySelector("[data-dietas-circuito-abrir]"), null);
  assert.equal(panel.querySelector("[data-dietas-circuito-decision]"), null);
  vista.desmontar();
});

test("abre el documento con sus líneas y total, exige motivo al devolver y muestra el recibo solo tras la respuesta", async () => {
  const contenedor = raiz(); const llamadas = []; const lecturas = [];
  const cliente = {
    listar: async () => ({ items: [fila], competencia: "acreditada" }),
    documento: async (ref, etapa) => { lecturas.push([ref, etapa]); return documentoLeido; },
    decidir: async (ref, entrada) => { llamadas.push([ref, entrada]); return { comision: { referencia, estado: "devuelta", version: 4 }, recibo }; },
  };
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente, generarClaveIdempotencia: () => "decision-circuito-0001" });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.equal(panel.querySelector("h2").textContent, "Pendientes de revisar");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  assert.deepEqual(lecturas, [[referencia, "revision"]]);
  const detalle = panel.querySelector("[data-dietas-circuito-detalle]");
  assert.equal(detalle.hidden, false);
  assert.equal(panel.querySelectorAll("[data-dietas-circuito-linea]").length, 4);
  assert.match(texto(detalle), /Justificante just:ticket-01/u);
  // La línea D5 muestra su tipo y fecha con rótulos de negocio, sin códigos.
  assert.match(texto(detalle), /Taxi · 21 sept 2026 · Estación a sede/u);
  assert.doesNotMatch(texto(detalle), /otros-gastos|tipo_gasto/u);
  assert.match(texto(panel.querySelector("[data-dietas-circuito-total]")), /56,72/u);
  assert.equal(panel.querySelector('[data-dietas-circuito-decision="aprobar"]').textContent, "Elevar al responsable");
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
  assert.equal(llamadas.length, 0); assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /motivo/u);
  panel.querySelector("[data-dietas-circuito-motivo]").value = "  Falta el ticket del aparcamiento ";
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
  assert.deepEqual(llamadas[0], [referencia, { etapa: "revision", decision: "devolver", motivo: "Falta el ticket del aparcamiento", clave_idempotencia: "decision-circuito-0001", version_esperada: 3 }]);
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /Recibo rcd_1234567890123456789012/u);
  assert.equal(panel.querySelector("[data-dietas-circuito-detalle]").hidden, true);
  assert.equal(panel.querySelector("[data-dietas-circuito-abrir]"), null);
  vista.desmontar();
});

test("un resultado incierto conserva clave y decisión para repetir exactamente la misma acción", async () => {
  const contenedor = raiz(); const entradas = []; let intento = 0;
  const cliente = {
    listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
    decidir: async (_ref, entrada) => { entradas.push(entrada); if (!intento++) { const error = new Error(); error.resultadoIndeterminado = true; throw error; } return { comision: { referencia, estado: "pendiente_autorizacion", version: 4 }, recibo: { ...recibo, repeticion: true } }; },
  };
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente, generarClaveIdempotencia: () => "decision-circuito-0001" });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="aprobar"]') });
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /confirmado/u);
  assert.equal(panel.querySelector('[data-dietas-circuito-decision="aprobar"]').disabled, true);
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-reintento]") });
  assert.equal(entradas.length, 2); assert.deepEqual(entradas[1], entradas[0]);
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /Ya estaba registrado/u);
  vista.desmontar();
});

test("el control de documentos busca por etapa y fechas acreditadas", async () => {
  const contenedor = raiz(); const consultas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, etapas: ["revision", "liquidacion"], etapaInicial: "revision",
    cliente: { listar: async (consulta) => { consultas.push(consulta); return { items: [], competencia: "acreditada" }; }, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.equal(panel.querySelector("h2").textContent, "Control de documentos");
  const filtros = panel.querySelector("[data-dietas-circuito-filtros]");
  assert.equal(filtros.hidden, false);
  panel.querySelector("[data-dietas-circuito-etapa]").value = "liquidacion";
  panel.querySelector("[data-dietas-circuito-desde]").value = "2026-09-01";
  panel.querySelector("[data-dietas-circuito-hasta]").value = "2026-09-30";
  await filtros.listeners.submit({ preventDefault() {} });
  assert.deepEqual(consultas.at(-1), { etapa: "liquidacion", limit: 20, fecha_desde: "2026-09-01", fecha_hasta: "2026-09-30" });
  assert.match(texto(panel), /No hay documentos para estas fechas/u);
  assert.throws(() => montarVistaBandejaCircuitoDietas(raiz(), { cliente: { listar() {}, decidir() {} }, etapas: ["otra"] }), TypeError);
  vista.desmontar();
});

test("desmontar aborta la lectura y descarta su respuesta tardía", async () => {
  const contenedor = raiz(); let signal; let resolver;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: (_consulta, opciones) => { signal = opciones.signal; return new Promise((resolve) => { resolver = resolve; }); }, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); vista.desmontar(); assert.equal(signal.aborted, true); resolver({ items: [fila], competencia: "acreditada" }); await esperar(); assert.equal(contenedor.children.length, 0);
});

test("un reenvío muestra etapa, fecha y motivo de la devolución anterior, sin quién la hizo", async () => {
  const contenedor = raiz();
  const reenviado = { ...documentoLeido, version: 5, devolucion: { etapa: "autorizacion", motivo: "Falta el justificante del taxi", version: 3, devuelta_en: "2026-09-19T09:00:00.123456Z" } };
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: async () => ({ items: [fila], competencia: "acreditada" }),
    documento: async () => reenviado, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  const etapa = panel.querySelector('[data-dietas-circuito-reenvio="etapa"]');
  assert.match(texto(etapa), /Reenvío.*Devuelto en Autorización el 19 sept 2026/u);
  assert.match(texto(panel.querySelector('[data-dietas-circuito-reenvio="motivo"]')), /Motivo de la devolución.*Falta el justificante del taxi/u);
  assert.doesNotMatch(texto(panel), /per_|act_|actor/u);
  vista.desmontar();
  const sinReenvio = raiz();
  const otra = montarVistaBandejaCircuitoDietas(sinReenvio, { cliente: { listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panelDos = sinReenvio.querySelector("[data-dietas-bandeja-circuito]");
  await panelDos.listeners.click({ target: panelDos.querySelector("[data-dietas-circuito-abrir]") });
  assert.equal(panelDos.querySelector("[data-dietas-circuito-reenvio]"), null);
  otra.desmontar();
});

test("el motivo de devolución se recorta con el mismo conjunto de blancos que rechaza Go", async () => {
  const contenedor = raiz(); const llamadas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { generarClaveIdempotencia: () => "decision-circuito-0002", cliente: {
    listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
    decidir: async (ref, entrada) => { llamadas.push(entrada); return { comision: { referencia, estado: "devuelta", version: 4 }, recibo }; } } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  panel.querySelector("[data-dietas-circuito-motivo]").value = "\ufeff\u0085 Falta el ticket\u00a0\u0085";
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
  assert.equal(llamadas[0].motivo, "Falta el ticket");
  vista.desmontar();
});
