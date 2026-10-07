import "./test-preparar-i18n.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaRectificacionDietas } from "./vista-rectificacion-dietas.js";

const dato = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.parent = null; this.listeners = {}; this.textContent = ""; this.value = ""; this.checked = false; this.hidden = false; this.disabled = false; }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { if (typeof nodo === "object") nodo.parent = this; }); }
  remove() { this.parent?.removeChild(this); } removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; } removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(nombre, valor) { if (nombre === "name") this.name = String(valor); }
  focus() { this.ownerDocument.activeElement = this; }
  matches(selector) { if (selector.includes(",")) return selector.split(",").some((parte) => this.matches(parte.trim())); const m = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u); if (!m) return this.tagName === selector; const clave = m[1] === "name" ? "name" : dato(m[1]); const valor = m[1] === "name" ? this.name : this.dataset[clave]; return valor !== undefined && (m[2] === undefined || valor === m[2]); }
  closest(selector) { for (let actual = this; actual; actual = actual.parent) if (actual.matches(selector)) return actual; return null; }
  querySelectorAll(selector) { const salida = []; const visitar = (nodo) => { if (typeof nodo === "object") { if (nodo.matches(selector)) salida.push(nodo); nodo.children.forEach(visitar); } }; visitar(this); return salida; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
const asignacion = { verificada: true, relacion_ref: `rel_${"a".repeat(22)}`, unidad_ref: "unidad-personal", asignacion_ref: `ads_${"b".repeat(22)}`, version: 2, fecha_referencia: "2026-09-24" };
const resultado = { solicitud_ref: `srd_${"c".repeat(32)}`, recibo_ref: `rrd_${"d".repeat(32)}`, estado: "pendiente", registrada_en: "2026-09-24T10:00:00.000000Z" };
const esperar = async () => { await Promise.resolve(); await Promise.resolve(); };

test("consulta 404, abre el formulario y confirma solo tras recibo válido", async () => {
  const contenedor = raiz(); const entradas = []; const vista = montarVistaRectificacionDietas(contenedor, { asignacion, generarClaveIdempotencia: () => "rectificacion-dietas-0001", cliente: { consultar: async () => { const error = new Error(); error.codigo = "no_encontrada"; throw error; }, solicitar: async (entrada) => { entradas.push(entrada); return resultado; } } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-rectificacion]"); assert.match(panel.querySelector("[data-dietas-rectificacion-estado]").textContent, /No hay/u);
  panel.listeners.click({ target: panel.querySelector("[data-dietas-rectificacion-abrir]") }); const formulario = panel.querySelector("[data-dietas-rectificacion-form]"); assert.equal(formulario.hidden, false);
  formulario.querySelectorAll('[name="campos_a_revisar"]')[0].checked = true; formulario.querySelector('[name="motivo_revision"]').value = "Centro incorrecto";
  formulario.listeners.submit({ preventDefault() {} }); await esperar(); assert.deepEqual(entradas[0].campos_a_revisar, ["centro_ref"]); assert.equal(entradas[0].clave_idempotencia, "rectificacion-dietas-0001");
  const mensaje = panel.querySelector("[data-dietas-rectificacion-estado]").textContent;
  assert.match(mensaje, /Solicitud registrada/u); assert.match(mensaje, /rrd_/u); assert.match(mensaje, /Pendiente de revisión/u);
  assert.doesNotMatch(mensaje, /2026-09-24T|Estado: pendiente/u); vista.desmontar();
});

test("GET muestra estado final y recibo de acceso sin atribuir la consulta al alta", async () => {
  for (const [estado, etiqueta] of [["confirmada", "Corrección confirmada"], ["rechazada", "Solicitud rechazada"]]) {
    const contenedor = raiz(); const acceso = { ...resultado, recibo_ref: `rrd_${"e".repeat(32)}`,
      estado, registrada_en: "2026-10-03T10:00:00.000000Z" };
    const vista = montarVistaRectificacionDietas(contenedor, { asignacion,
      cliente: { consultar: async () => acceso, solicitar: async () => resultado } });
    await esperar();
    const mensaje = contenedor.querySelector("[data-dietas-rectificacion-estado]").textContent;
    assert.match(mensaje, /Consulta realizada el 3 oct 2026/u);
    assert.match(mensaje, /Recibo de acceso rrd_e/u);
    assert.match(mensaje, new RegExp(etiqueta, "u"));
    assert.doesNotMatch(mensaje, /Solicitud registrada|Solicitud recuperada|2026-10-03T|Estado de la solicitud: (confirmada|rechazada)/u);
    vista.desmontar();
  }
});

test("un estado inesperado se presenta sin el código técnico", async () => {
  const contenedor = raiz(); const vista = montarVistaRectificacionDietas(contenedor, { asignacion,
    cliente: { consultar: async () => ({ ...resultado, estado: "transitoria" }), solicitar: async () => resultado } });
  await esperar();
  const mensaje = contenedor.querySelector("[data-dietas-rectificacion-estado]").textContent;
  assert.match(mensaje, /Estado de la solicitud: No disponible/u);
  assert.doesNotMatch(mensaje, /transitoria/u);
  vista.desmontar();
});

test("un 503 incierto bloquea la edición y un segundo envío recupera la misma solicitud", async () => {
  const contenedor = raiz(); const entradas = []; let intento = 0; let claves = 0; let signal; let terminarConsulta;
  const vista = montarVistaRectificacionDietas(contenedor, { asignacion,
    generarClaveIdempotencia: () => `rectificacion-dietas-000${++claves}`,
    cliente: { consultar: (_entrada, opciones) => { signal = opciones.signal; return new Promise((resolver) => { terminarConsulta = resolver; }); },
      solicitar: async (entrada) => { entradas.push(entrada); if (!intento++) { const error = new Error(); error.codigo = "no_disponible"; error.resultadoIndeterminado = true; throw error; } return { ...resultado, estado: "replay_confirmado" }; } } });
  const panel = contenedor.querySelector("[data-dietas-rectificacion]"); panel.listeners.click({ target: panel.querySelector("[data-dietas-rectificacion-abrir]") });
  const formulario = panel.querySelector("[data-dietas-rectificacion-form]"); const motivo = formulario.querySelector('[name="motivo_revision"]');
  formulario.querySelectorAll('[name="campos_a_revisar"]')[1].checked = true; motivo.value = "Unidad incorrecta";
  formulario.listeners.submit({ preventDefault() {} }); await esperar();
  assert.equal(formulario.querySelector("fieldset").disabled, true); assert.equal(motivo.disabled, true);
  assert.equal(formulario.querySelector("[data-dietas-rectificacion-enviar]").disabled, true);
  assert.ok(panel.querySelector("[data-dietas-rectificacion-reintentar]"));
  terminarConsulta(resultado); await esperar();
  assert.match(panel.querySelector("[data-dietas-rectificacion-estado]").textContent, /No se ha confirmado/u);
  motivo.value = "Otra petición"; formulario.listeners.submit({ preventDefault() {} }); await esperar();
  assert.equal(claves, 1); assert.deepEqual(entradas[1], entradas[0]);
  assert.equal(motivo.disabled, false); assert.doesNotMatch(panel.querySelector("[data-dietas-rectificacion-estado]").textContent, /replay_confirmado|2026-09-24T/u);
  assert.match(panel.querySelector("[data-dietas-rectificacion-estado]").textContent, /Pendiente de revisión/u);
  vista.desmontar(); assert.equal(signal.aborted, true);
});
