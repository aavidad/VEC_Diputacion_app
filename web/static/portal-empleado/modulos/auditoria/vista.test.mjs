import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaAuditoria, renderizarVistaAuditoria, validarRespuestaAuditoria } from "./vista.js";

const registro = {
  id: "aud_1", modulo_id: "personal", accion: "relacion.actualizada", actor_ref: "per_1",
  ocurrido_en: "2026-09-28T08:00:00Z", resultado: "confirmado", expediente_ref: "exp_1",
  recibo_ref: "rec_1", antes_sha256: "a".repeat(64), despues_sha256: "b".repeat(64),
  motivo: "rectificacion", fuente: "Personal", datos_disponibles: true,
  antes: { unidad: "A" }, despues: { unidad: "B" },
};
const opciones = { finalidad_ref: "fin_1", motivo_ref: "catalogo:v1:revision", permiso_requerido: "auditoria.consultar", es_ejemplo: true };
const esperar = () => new Promise((resolver) => setImmediate(resolver));
function raizFalsa() {
  const eventos = {};
  return { eventos, raiz: {
    innerHTML: "", replaceChildren() { this.innerHTML = ""; }, querySelector() { return null; },
    addEventListener(k, fn) { eventos[k] = fn; }, removeEventListener(k) { delete eventos[k]; },
  } };
}

test("sin expediente de navegación la consulta permanece cerrada y no ofrece vínculo ni actor libres", () => {
  const html = renderizarVistaAuditoria();
  assert.match(html, /Abra Auditoría desde un expediente autorizado/u);
  assert.match(html, /id="auditoria-ayuda"[^>]+hidden/u);
  assert.match(html, /name="desde"[^>]+disabled/u);
  assert.doesNotMatch(html, /name="expediente_ref"|name="actor_ref"|<table/u);
});

test("solo muestra expediente exacto escapado y aviso de configuración de ejemplo", () => {
  const html = renderizarVistaAuditoria({ estado: "esperando", habilitada: true,
    expedienteRef: 'exp_1"><script>', ejemplo: true });
  assert.match(html, /exp_1&quot;&gt;&lt;script&gt;/u);
  assert.match(html, /Configuración de ejemplo pendiente de RRHH/u);
  assert.doesNotMatch(html, /<script>|name="expediente_ref"|name="actor_ref"/u);
});

test("registros autorizados muestran actor, instante, motivo y valores minimizados", () => {
  const respuesta = validarRespuestaAuditoria({ registros: [{ ...registro, antes: { unidad: "<script>" } }], siguiente_cursor: "" });
  const html = renderizarVistaAuditoria({ estado: "disponible", habilitada: true, registros: respuesta.registros });
  assert.match(html, /per_1|rectificacion|Huella SHA-256|datetime="2026-09-28T08:00:00Z"/u);
  assert.match(html, /&lt;script&gt;/u);
  assert.doesNotMatch(html, /<script>/u);
  assert.throws(() => validarRespuestaAuditoria({ registros: [{ ...registro, despues: { secreto: {} } }], siguiente_cursor: "" }), /proyección/u);
});

test("paginación se mantiene y denegación no presenta datos previos", () => {
  const pagina = renderizarVistaAuditoria({ estado: "vacio", habilitada: true, pagina: 2, puedeAnterior: true, siguienteCursor: "cursor_3" });
  assert.match(pagina, /Página 2|data-auditoria-siguiente/u);
  const denegado = renderizarVistaAuditoria({ estado: "denegado", habilitada: true, registros: [registro] });
  assert.match(denegado, /Acceso denegado/u);
  assert.doesNotMatch(denegado, /per_1|<table/u);
});

test("GET de opciones antecede al POST, que usa expediente de navegación y refs recibidas", async () => {
  const { eventos, raiz } = raizFalsa();
  let peticion;
  const fuente = {
    obtenerOpciones: async () => opciones,
    consultar: async (entrada) => { peticion = entrada; return { registros: [registro], siguiente_cursor: "" }; },
  };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(k) { return { desde: "2026-09-28T08:00", hasta: "2026-09-28T09:00" }[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1" });
    assert.match(raiz.innerHTML, /Comprobando opciones/u);
    await esperar();
    assert.match(raiz.innerHTML, /Configuración de ejemplo/u);
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    await esperar();
    assert.equal(peticion.expediente_ref, "exp_1");
    assert.equal(peticion.actor_ref, "");
    assert.equal(peticion.finalidad_ref, opciones.finalidad_ref);
    assert.equal(peticion.motivo_ref, opciones.motivo_ref);
    assert.match(raiz.innerHTML, /per_1/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});

test("sin expediente no pide opciones, y 403 en opciones no expone datos", async () => {
  const { raiz } = raizFalsa();
  let llamadas = 0;
  const fuente = { obtenerOpciones: async () => { ++llamadas; throw Object.assign(Error("secreto"), { codigo: "denegado" }); },
    consultar: async () => { throw Error("no debe llamarse"); } };
  const cerrada = montarVistaAuditoria({ raiz, fuente });
  assert.equal(llamadas, 0);
  cerrada.desmontar();
  const denegada = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1" });
  await esperar();
  assert.equal(llamadas, 1);
  assert.match(raiz.innerHTML, /Acceso denegado/u);
  assert.doesNotMatch(raiz.innerHTML, /secreto|<table/u);
  denegada.desmontar();
});

test("cambiar fecha cancela respuesta tardía y limpia el resultado", async () => {
  const { eventos, raiz } = raizFalsa();
  let resolver, signal;
  const fuente = { obtenerOpciones: async () => opciones,
    consultar: (_entrada, o) => { signal = o.signal; return new Promise((r) => { resolver = r; }); } };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(k) { return { desde: "2026-09-28T08:00", hasta: "2026-09-28T09:00" }[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1" });
    await esperar();
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    eventos.input({ target: { matches: () => true, name: "desde", value: "2026-09-28T08:30", selectionStart: null, selectionEnd: null } });
    assert.equal(signal.aborted, true);
    resolver({ registros: [registro], siguiente_cursor: "" });
    await esperar();
    assert.doesNotMatch(raiz.innerHTML, /per_1/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});
