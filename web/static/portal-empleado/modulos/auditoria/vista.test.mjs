import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaAuditoria, renderizarVistaAuditoria, validarRespuestaAuditoria } from "./vista.js";

const registro = Object.freeze({
  id: "aud_1", modulo_id: "personal", accion: "relacion.actualizada", actor_ref: "per_1",
  ocurrido_en: "2026-09-28T08:00:00Z", resultado: "confirmado", expediente_ref: "exp_1",
  recibo_ref: "rec_1", antes_sha256: "a".repeat(64), despues_sha256: "b".repeat(64),
  motivo: "rectificacion", fuente: "Personal", datos_disponibles: true,
  antes: { unidad: "A" }, despues: { unidad: "B" },
});

test("la pantalla empieza cerrada, sin muestra local ni ayuda abierta", () => {
  const html = renderizarVistaAuditoria();
  assert.match(html, /Consulta no disponible/u);
  assert.match(html, /data-auditoria-ayuda[^>]+aria-expanded="false"/u);
  assert.match(html, /id="auditoria-ayuda"[^>]+hidden/u);
  assert.match(html, /name="expediente_ref"[^>]+disabled/u);
  assert.doesNotMatch(html, /<table|aud_1|datos-presentacion/u);
});

test("registros autorizados muestran actor, instante, motivo y antes/después sin inyección", () => {
  const respuesta = validarRespuestaAuditoria({ registros: [{ ...registro, antes: { unidad: "<script>" } }], siguiente_cursor: "" });
  const html = renderizarVistaAuditoria({ estado: "disponible", habilitada: true, registros: respuesta.registros });
  assert.match(html, /per_1/u);
  assert.match(html, /rectificacion/u);
  assert.match(html, /Antes|Después/u);
  assert.match(html, /&lt;script&gt;/u);
  assert.doesNotMatch(html, /<script>/u);
  assert.match(html, /Huella SHA-256/u);
  assert.match(html, /datetime="2026-09-28T08:00:00Z"/u);
});

test("la respuesta incompatible se rechaza entera y la denegación no filtra el resultado anterior", () => {
  assert.throws(() => validarRespuestaAuditoria({ registros: [registro, { ...registro, id: "aud_2", despues: { secreto: {} } }], siguiente_cursor: "" }), /proyección/u);
  assert.throws(() => validarRespuestaAuditoria({ registros: [{ ...registro, datos_disponibles: false }], siguiente_cursor: "" }), /contradictoria/u);
  const html = renderizarVistaAuditoria({ estado: "denegado", habilitada: true, registros: [registro] });
  assert.match(html, /Acceso denegado/u);
  assert.doesNotMatch(html, /per_1|rectificacion|<table/u);
});

test("paginación muestra controles y los filtros exactos quedan escapados", () => {
  const html = renderizarVistaAuditoria({ estado: "vacio", habilitada: true,
    filtros: { expediente_ref: 'exp_1" autofocus' }, pagina: 2, puedeAnterior: true, siguienteCursor: "cursor_3" });
  assert.match(html, /Página 2/u);
  assert.match(html, /data-auditoria-siguiente/u);
  assert.match(html, /exp_1&quot; autofocus/u);
  assert.doesNotMatch(html, /value="exp_1" autofocus"/u);
});

test("sin contexto positivo la vista no consulta; tras desmontar ignora una respuesta tardía", async () => {
  const manejadores = {};
  const raiz = {
    innerHTML: "", replaceChildren() { this.innerHTML = ""; },
    addEventListener(nombre, fn) { manejadores[nombre] = fn; },
    removeEventListener(nombre) { delete manejadores[nombre]; },
  };
  let llamadas = 0;
  const fuente = { consultar() { ++llamadas; return new Promise(() => {}); } };
  const cerrada = montarVistaAuditoria({ raiz, fuente });
  assert.match(raiz.innerHTML, /Consulta no disponible/u);
  assert.equal(llamadas, 0);
  cerrada.desmontar();
  assert.equal(raiz.innerHTML, "");
});

test("denegación posterior elimina cualquier resultado y no expone el error del servidor", async () => {
  const manejadores = {};
  const raiz = {
    innerHTML: "", replaceChildren() { this.innerHTML = ""; },
    addEventListener(nombre, fn) { manejadores[nombre] = fn; },
    removeEventListener(nombre) { delete manejadores[nombre]; },
  };
  let resolver; let rechazar;
  const fuente = { consultar() { return new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; }); } };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(clave) {
    return { expediente_ref: "exp_1", desde: "2026-09-28T08:00", hasta: "2026-09-28T09:00", actor_ref: "" }[clave];
  } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, contextoConsulta: { finalidadRef: "fin_1", motivoRef: "mot_1" } });
    manejadores.submit({ target: { matches: () => true }, preventDefault() {} });
    assert.match(raiz.innerHTML, /Consultando auditoría/u);
    resolver({ registros: [registro], siguiente_cursor: "" });
    await new Promise((resolve) => setImmediate(resolve));
    assert.match(raiz.innerHTML, /per_1/u);
    manejadores.submit({ target: { matches: () => true }, preventDefault() {} });
    rechazar(Object.assign(new Error("dato protegido"), { codigo: "denegado" }));
    await new Promise((resolve) => setImmediate(resolve));
    assert.match(raiz.innerHTML, /Acceso denegado/u);
    assert.doesNotMatch(raiz.innerHTML, /per_1|dato protegido/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});

test("al cambiar expediente se cancela la lectura y se ignora su respuesta tardía", async () => {
  const eventos = {};
  const raiz = {
    innerHTML: "", replaceChildren() { this.innerHTML = ""; }, querySelector() { return null; },
    addEventListener(k, fn) { eventos[k] = fn; }, removeEventListener(k) { delete eventos[k]; },
  };
  let resolver, signal;
  const fuente = { consultar(_entrada, opciones) { signal = opciones.signal; return new Promise((r) => { resolver = r; }); } };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(k) { return {
    expediente_ref: "exp_1", desde: "2026-09-28T08:00", hasta: "2026-09-28T09:00", actor_ref: "",
  }[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, contextoConsulta: { finalidadRef: "fin_1", motivoRef: "mot_1" } });
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    eventos.input({ target: { matches: () => true, name: "expediente_ref", value: "exp_2", selectionStart: 5, selectionEnd: 5 } });
    assert.equal(signal.aborted, true);
    resolver({ registros: [registro], siguiente_cursor: "" });
    await new Promise((resolve) => setImmediate(resolve));
    assert.match(raiz.innerHTML, /Indique los filtros/u);
    assert.doesNotMatch(raiz.innerHTML, /per_1/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});
