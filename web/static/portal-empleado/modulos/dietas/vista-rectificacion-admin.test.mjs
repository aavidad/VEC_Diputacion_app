import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaRectificacionAdminDietas } from "./vista-rectificacion-admin.js";

const dato = (nombre) => nombre.slice(5).replace(/-([a-z])/gu, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = ""; this.value = ""; }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); }
  replaceChildren(...nodos) { this.children.forEach((n) => { n.parent = null; }); this.children = []; this.append(...nodos); }
  remove() { this.parent?.removeChild(this); }
  removeChild(n) { this.children = this.children.filter((x) => x !== n); n.parent = null; }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  matches(selector) { const m = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u); if (!m) return this.tagName === selector; const valor = m[1] === "name" ? this.name : this.dataset[dato(m[1])]; return valor !== undefined && (m[2] === undefined || valor === m[2]); }
  closest(selector) { const selectores = selector.split(/,\s*/u); for (let n = this; n; n = n.parent) if (selectores.some((s) => n.matches(s))) return n; return null; }
  querySelectorAll(selector) { const encontrados = []; const buscar = (n) => { if (n.matches(selector)) encontrados.push(n); n.children.forEach(buscar); }; buscar(this); return encontrados; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
}
function contenedor() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
const solicitudRef = `srd_${"a".repeat(32)}`;
const reciboRef = `rrd_${"b".repeat(32)}`;
const personaRef = `per_${"c".repeat(22)}`;
const empleadoRef = `emp_${"d".repeat(22)}`;
const relacionRef = `rel_${"e".repeat(22)}`;
const asignacionRef = `ads_${"f".repeat(22)}`;
const solicitud = { solicitud_ref: solicitudRef, estado: "pendiente", persona_ref: personaRef, empleado_ref: empleadoRef,
  relacion_ref: relacionRef, unidad_ref: "unidad-servidor", asignacion_ref: asignacionRef, version_origen: 4,
  fecha_referencia: "2026-09-24", campos_a_revisar: ["centro_ref"], motivo_revision: "Centro incorrecto",
  detalle_solicitado: "Revisar el centro", registrada_en: "2026-09-24T10:00:00.000000Z",
  asignacion_actual: { asignacion_ref: asignacionRef, version: 4, centro_ref: "centro-servidor",
    administrativo_persona_ref: `per_${"1".repeat(22)}`, responsable_persona_ref: `per_${"2".repeat(22)}`,
    grupo_dieta: 2, vigente_desde: "2026-09-01" } };
const lista = { cardinalidad: 1, solicitudes: [solicitud] };
const catalogo = { solicitud_ref: solicitudRef, asignacion_ref: asignacionRef, version: 4,
  centros: [{ ref: "centro-servidor", etiqueta: "Centro acreditado" }],
  administrativos: [{ persona_ref: solicitud.asignacion_actual.administrativo_persona_ref, etiqueta: "Persona administrativa acreditada" }],
  responsables: [{ persona_ref: solicitud.asignacion_actual.responsable_persona_ref, etiqueta: "Persona responsable acreditada" }] };
const espera = async () => { await Promise.resolve(); await Promise.resolve(); };
function abrirFicha(raiz) { const panel = raiz.querySelector("[data-dietas-rectificacion-admin]"); panel.listeners.click({ target: panel.querySelector("[data-dietas-ra-ver]") }); return panel; }
const textos = (n) => [n.textContent, ...n.children.flatMap((h) => textos(h))].join(" ");

test("consulta lista competente sin ámbito aportado por navegador y confirma con datos de Personal", async () => {
  const raiz = contenedor(); const llamadas = []; const consultas = [];
  const vista = montarVistaRectificacionAdminDietas(raiz, {
    cliente: { listar: async (opciones) => { consultas.push(opciones); return lista; }, decidir: async (ref, entrada) => { llamadas.push({ ref, entrada }); return { solicitud_ref: ref, recibo_ref: reciboRef, estado: "confirmada", registrada_en: "2026-09-24T11:00:00.000000Z" }; } },
    clienteCatalogoCompetente: { consultar: async () => catalogo },
    generarClaveIdempotencia: () => "decision-rectificacion-0001", confirmarAccion: () => true,
  });
  await espera(); const panel = abrirFicha(raiz); await espera();
  assert.deepEqual(Object.keys(consultas[0]), ["signal"]);
  const form = panel.querySelector("[data-dietas-ra-form]"); form.querySelector('[name="motivo_decision"]').value = "Corrección comprobada";
  assert.equal(form.querySelector('[name="ra_centro"]').tagName, "select");
  assert.equal(form.querySelector('[name="ra_administrativo"]').tagName, "select");
  assert.equal(form.querySelector('[name="ra_responsable"]').tagName, "select");
  assert.equal(form.querySelector('[data-dietas-ra-decision="confirmar"]').disabled, false);
  await panel.listeners.click({ target: form.querySelector('[data-dietas-ra-decision="confirmar"]') }); await espera();
  assert.equal(llamadas[0].ref, solicitudRef);
  assert.equal(llamadas[0].entrada.unidad_ref, "unidad-servidor");
  assert.equal(llamadas[0].entrada.persona_ref, personaRef);
  assert.equal(llamadas[0].entrada.correccion.administrativo_persona_ref, solicitud.asignacion_actual.administrativo_persona_ref);
  assert.equal(llamadas[0].entrada.clave_idempotencia, "decision-rectificacion-0001");
  assert.equal(vista.recibo.recibo_ref, reciboRef);
  assert.match(panel.querySelector("[data-dietas-ra-estado-ficha]").textContent, /Recibo/u);
  vista.desmontar();
});

test("sin catálogo autorizado no ofrece referencias libres ni permite confirmar", async () => {
  const raiz = contenedor(); const entradas = [];
  const vista = montarVistaRectificacionAdminDietas(raiz, {
    cliente: { listar: async () => lista, decidir: async (_ref, entrada) => { entradas.push(entrada); return { solicitud_ref: solicitudRef, recibo_ref: reciboRef, estado: "rechazada", registrada_en: "2026-09-24T11:00:00Z" }; } },
    generarClaveIdempotencia: () => "decision-rectificacion-0004", confirmarAccion: () => true,
  });
  await espera(); const panel = abrirFicha(raiz), form = panel.querySelector("[data-dietas-ra-form]");
  for (const nombre of ["ra_centro", "ra_administrativo", "ra_responsable"]) {
    assert.equal(form.querySelector(`[name="${nombre}"]`).tagName, "select");
    assert.equal(form.querySelector(`[name="${nombre}"]`).disabled, true);
  }
  assert.equal(form.querySelector('[data-dietas-ra-decision="confirmar"]').disabled, true);
  assert.match(form.querySelector("[data-dietas-ra-estado-catalogo]").textContent, /No hay opciones autorizadas/u);
  assert.doesNotMatch(textos(panel), /per_[A-Za-z0-9_-]{22,128}|centro-servidor/u);
  await panel.listeners.click({ target: form.querySelector('[data-dietas-ra-decision="confirmar"]') }); await espera();
  assert.equal(entradas.length, 0);
  form.querySelector('[name="motivo_decision"]').value = "No procede por cambio de unidad";
  await panel.listeners.click({ target: form.querySelector('[data-dietas-ra-decision="rechazar"]') }); await espera();
  assert.equal(entradas[0].decision, "rechazar"); vista.desmontar();
});

test("catálogo de otra asignación no habilita la confirmación", async () => {
  const raiz = contenedor(); let decisiones = 0;
  const vista = montarVistaRectificacionAdminDietas(raiz, {
    cliente: { listar: async () => lista, decidir: async () => { decisiones++; throw new Error("decisión inesperada"); } },
    clienteCatalogoCompetente: { consultar: async () => ({ ...catalogo, version: 9 }) },
    confirmarAccion: () => true,
  });
  await espera(); const panel = abrirFicha(raiz); await espera();
  assert.equal(panel.querySelector('[data-dietas-ra-decision="confirmar"]').disabled, true);
  assert.equal(panel.querySelector('[name="ra_administrativo"]').disabled, true);
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-ra-decision="confirmar"]') }); await espera();
  assert.equal(decisiones, 0); vista.desmontar();
});

test("rechazo no envía corrección y reintento incierto repite exactamente la misma decisión", async () => {
  const raiz = contenedor(); const entradas = []; let intento = 0;
  const vista = montarVistaRectificacionAdminDietas(raiz, {
    cliente: { listar: async () => lista, decidir: async (_ref, entrada) => { entradas.push(entrada); if (!intento++) { const e = new Error(); e.resultadoIndeterminado = true; throw e; } return { solicitud_ref: solicitudRef, recibo_ref: reciboRef, estado: "replay_confirmado", registrada_en: "2026-09-24T11:00:00.000000Z" }; } },
    generarClaveIdempotencia: () => "decision-rectificacion-0002", confirmarAccion: () => true,
  });
  await espera(); const panel = abrirFicha(raiz); panel.querySelector('[name="motivo_decision"]').value = "Datos acreditados no coinciden";
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-ra-decision="rechazar"]') }); await espera();
  assert.ok(panel.querySelector("[data-dietas-ra-reintentar]"));
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-ra-decision="confirmar"]') }); await espera();
  assert.equal(entradas.length, 1);
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-ra-reintentar]") }); await espera();
  assert.equal(entradas.length, 2); assert.deepEqual(entradas[1], entradas[0]);
  assert.equal(entradas[0].asignacion_ref, ""); assert.equal(entradas[0].version_esperada, 0);
  assert.equal(Object.hasOwn(entradas[0], "correccion"), false);
  assert.equal(vista.recibo.recibo_ref, reciboRef); vista.desmontar();
});

test("denegación no muestra solicitudes y desmontar cancela la lectura", async () => {
  const raiz = contenedor(); let signal;
  const vista = montarVistaRectificacionAdminDietas(raiz, { cliente: { listar: (_opciones) => new Promise((_resolve, reject) => { signal = _opciones.signal; reject(Object.assign(new Error(), { codigo: "acceso_denegado" })); }), decidir: async () => assert.fail("decisión inesperada") } });
  await espera(); const panel = raiz.querySelector("[data-dietas-rectificacion-admin]");
  assert.match(panel.querySelector("[data-dietas-ra-estado-lista]").textContent, /autorización/u);
  assert.equal(panel.querySelector("[data-dietas-ra-ver]"), null);
  vista.desmontar(); assert.equal(signal.aborted, true); assert.equal(raiz.children.length, 0);
});

test("una asignación posterior bloquea confirmar y conserva el rechazo motivado", async () => {
  const raiz = contenedor(); const entradas = []; const posterior = { ...solicitud, asignacion_actual: { ...solicitud.asignacion_actual, version: 5 } };
  const vista = montarVistaRectificacionAdminDietas(raiz, {
    cliente: { listar: async () => ({ cardinalidad: 1, solicitudes: [posterior] }), decidir: async (_ref, entrada) => { entradas.push(entrada); return { solicitud_ref: solicitudRef, recibo_ref: reciboRef, estado: "rechazada", registrada_en: "2026-09-24T11:00:00Z" }; } },
    generarClaveIdempotencia: () => "decision-rectificacion-0003", confirmarAccion: () => true,
  });
  await espera(); const panel = abrirFicha(raiz);
  assert.equal(panel.querySelector('[data-dietas-ra-decision="confirmar"]').disabled, true);
  panel.querySelector('[name="motivo_decision"]').value = "Asignación posterior acreditada";
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-ra-decision="rechazar"]') }); await espera();
  assert.equal(entradas[0].decision, "rechazar"); assert.equal(entradas[0].version_esperada, 0);
  vista.desmontar();
});
