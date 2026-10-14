import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteCorreccionesCronosHTTP, ErrorClienteCorreccionesCronos, RUTAS_CORRECCIONES_CRONOS } from "./cliente-correcciones-http.js";

const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const REF = `correccion:cronos:${CLAVE}`;
function respuesta(estado, cuerpo) {
  const bytes = new TextEncoder().encode(JSON.stringify(cuerpo));
  return { status: estado, redirected: false, headers: { get: (n) => n === "content-type" ? "application/json" : null }, body: { getReader() {
    let hecho = false; return { read: async () => hecho ? { done: true } : (hecho = true, { done: false, value: bytes }), releaseLock() {}, cancel: async () => {} };
  } } };
}
function bandeja() { return { paso: "responsable", pendientes: [{ solicitud_ref: REF, empleado_ref: "emp_0123456789abcdefghijkl", empleado_etiqueta: "Persona ficticia", marcaje_original_ref: "", hueco_declarado: true,
  movimiento: "entrada", fecha_civil: "2026-10-14", hora_pretendida: "08:00", motivo_codigo: "olvido_marcaje", estado: "pendiente_responsable", version: 1, solicitada_en: "2026-10-14T08:00:00Z" }] }; }
function historia() { return { solicitud_ref: REF, empleado_ref: "emp_0123456789abcdefghijkl", estado: "pendiente_responsable", version: 1, fecha_civil: "2026-10-14", hora_pretendida: "08:00", movimiento: "entrada", motivo_codigo: "olvido_marcaje",
  actuaciones: [{ paso: "solicitud", resultado: "", motivo: "", actor_ref: "", perfil_ref: "", estado: "pendiente_responsable", version: 1, recibo_ref: "recibo:cronos:inicial", instante_utc: "2026-10-14T08:00:00Z" }], asientos: [] }; }

test("GET de bandeja e historia propia usa rutas fijas y no envía identidad", async () => {
  const llamadas = [];
  const c = crearClienteCorreccionesCronosHTTP({ fetchImpl: async (url, o) => { llamadas.push([url, o]); return respuesta(200, llamadas.length === 1 ? bandeja() : historia()); } });
  assert.equal((await c.consultarBandeja("responsable")).pendientes.length, 1);
  assert.equal((await c.consultarHistoria(REF, { vista: "propia" })).actuaciones[0].recibo_ref, "recibo:cronos:inicial");
  assert.equal(llamadas[0][0], `${RUTAS_CORRECCIONES_CRONOS.bandeja}?paso=responsable`);
  assert.equal(llamadas[1][0], `${RUTAS_CORRECCIONES_CRONOS.historia}?solicitud_ref=${encodeURIComponent(REF)}&vista=propia`);
  for (const [, o] of llamadas) { assert.equal(o.method, "GET"); assert.equal(o.body, undefined); assert.equal(o.credentials, "same-origin"); assert.equal(o.cache, "no-store"); assert.equal(o.redirect, "error"); }
  await assert.rejects(c.consultarHistoria(REF, { vista: "propia", paso: "responsable" }), TypeError);
});

test("POST exige motivo, versión y recibo correspondiente; 403/503 quedan como errores", async () => {
  const entrada = { solicitud_ref: REF, clave_operacion: CLAVE, paso: "responsable", resultado: "favorable", motivo: "Olvido revisado", version_esperada: "1" };
  const recibo = { recibo: { solicitud_ref: REF, actuacion_ref: "correccion:actuacion:1", recibo_ref: "recibo:cronos:1", estado: "pendiente_rrhh", version: 2, instante_utc: "2026-10-14T09:00:00Z", replay: false } };
  let llamada;
  const c = crearClienteCorreccionesCronosHTTP({ fetchImpl: async (url, o) => { llamada = [url, o]; return respuesta(201, recibo); } });
  assert.equal((await c.registrarActuacion(entrada)).version, 2);
  assert.equal(llamada[0], RUTAS_CORRECCIONES_CRONOS.actuaciones);
  assert.deepEqual(JSON.parse(llamada[1].body), entrada);
  await assert.rejects(c.registrarActuacion({ ...entrada, motivo: " Olvido revisado" }), TypeError);
  await assert.rejects(c.registrarActuacion({ ...entrada, version_esperada: "2" }), TypeError);
  const malo = crearClienteCorreccionesCronosHTTP({ fetchImpl: async () => respuesta(201, { recibo: { ...recibo.recibo, estado: "aplicada" } }) });
  await assert.rejects(malo.registrarActuacion(entrada), (e) => e.codigo === "respuesta_incompatible");
  for (const estado of [403, 503]) {
    const fallido = crearClienteCorreccionesCronosHTTP({ fetchImpl: async () => respuesta(estado, { error: estado === 403 ? "no_competente" : "no_disponible" }) });
    await assert.rejects(fallido.registrarActuacion(entrada), (e) => e instanceof ErrorClienteCorreccionesCronos && e.estado === estado);
  }
});

test("RRHH y aplicación conservan pasos, versiones y resultados distintos", async () => {
  for (const [paso, resultado, previa, nueva, estado] of [
    ["rrhh", "desfavorable", "2", 3, "denegada_rrhh"], ["aplicar", "", "3", 4, "aplicada"],
  ]) {
    const entrada = { solicitud_ref: REF, clave_operacion: CLAVE, paso, resultado, motivo: "Actuación revisada", version_esperada: previa };
    const c = crearClienteCorreccionesCronosHTTP({ fetchImpl: async () => respuesta(201, { recibo: { solicitud_ref: REF, actuacion_ref: "correccion:actuacion:2", recibo_ref: "recibo:cronos:2", estado, version: nueva, instante_utc: "2026-10-14T09:00:00Z", replay: false } }) });
    assert.equal((await c.registrarActuacion(entrada)).estado, estado);
  }
});
