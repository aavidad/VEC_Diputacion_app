import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { webcrypto } from "node:crypto";
import {
  calcularHuellaDocumentoCronos, crearClienteNotificacionesCronosHTTP, ErrorClienteNotificacionesCronos, RUTAS_NOTIFICACIONES_CRONOS,
  textoNotificacionValido,
} from "./cliente-notificaciones-http.js";

function respuestaJSON(estado, cuerpo) {
  const bytes = new TextEncoder().encode(JSON.stringify(cuerpo));
  return { status: estado, ok: estado >= 200 && estado < 300, redirected: false,
    headers: { get: (n) => (n === "content-type" ? "application/json; charset=utf-8" : null) },
    body: { getReader() { let hecho = false; return { read: async () => (hecho ? { done: true } : (hecho = true, { done: false, value: bytes })), releaseLock() {}, cancel: async () => {} }; } } };
}
const REF = "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001";
const HUELLA = "a".repeat(64);
const PROPIAS = {
  tipos: [{ tipo_version_ref: "notificacion:cronos:tipo:otra-comunicacion:sintetico-1", tipo_ref: "notificacion:cronos:tipo:otra-comunicacion", nombre: "Otra comunicación a RRHH" }],
  notificaciones: [{ notificacion_ref: REF, tipo_ref: "notificacion:cronos:tipo:otra-comunicacion", tipo_nombre: "Otra comunicación a RRHH", fecha_referida: "2026-09-24",
    texto: "No pude fichar.\nLo comunico.", adjunto_ref: "registro:sintetico:0001", adjunto_sha256: HUELLA, registrada_en: "2026-09-24T08:00:00.123456Z",
    estado: "atendida", atendida_en: "2026-09-25T08:00:00Z" }],
};
const BANDEJA = { notificaciones: [{ notificacion_ref: REF, empleado_ref: "emp_AAAAAAAAAAAAAAAAAAAAAA", empleado_etiqueta: "Persona sintética A",
  tipo_ref: "notificacion:cronos:tipo:otra-comunicacion", tipo_nombre: "Otra", fecha_referida: "2026-09-24", texto: "Texto", registrada_en: "2026-09-24T08:00:00Z", atendida: false }] };
const ENTRADA = { clave_operacion: "not-a-00001", tipo_version_ref: "notificacion:cronos:tipo:otra-comunicacion:sintetico-1", fecha_referida: "2026-09-24",
  texto: "No pude fichar", adjunto_ref: "registro:sintetico:0001", adjunto_sha256: HUELLA };

test("las consultas van al mismo origen, sin identidad del cliente, y validan la respuesta completa", async () => {
  const llamadas = [];
  const cliente = crearClienteNotificacionesCronosHTTP({ fetchImpl: async (url, o) => { llamadas.push({ url, o }); return respuestaJSON(200, url === RUTAS_NOTIFICACIONES_CRONOS.propio ? PROPIAS : BANDEJA); } });
  assert.equal((await cliente.consultarPropias()).notificaciones[0].estado, "atendida");
  assert.equal((await cliente.consultarBandeja()).notificaciones.length, 1);
  assert.deepEqual(llamadas.map((l) => l.url), [RUTAS_NOTIFICACIONES_CRONOS.propio, RUTAS_NOTIFICACIONES_CRONOS.bandeja]);
  for (const { o } of llamadas) {
    assert.equal(o.credentials, "same-origin"); assert.equal(o.redirect, "error"); assert.equal(o.referrerPolicy, "no-referrer"); assert.equal(o.cache, "no-store");
  }
  for (const [nombre, alterar] of [
    ["atendida sin fecha", (v) => { delete v.notificaciones[0].atendida_en; }],
    ["adjunto a medias", (v) => { delete v.notificaciones[0].adjunto_sha256; }],
    ["estado inventado", (v) => { v.notificaciones[0].estado = "resuelta"; }],
    ["campo extra", (v) => { v.notificaciones[0].empleado_ref = "emp_x"; }],
    ["texto largo", (v) => { v.notificaciones[0].texto = "x".repeat(513); }],
    ["tipo sin versión", (v) => { v.tipos[0].tipo_version_ref = "notificacion:cronos:tipo:x"; }],
  ]) {
    const malo = structuredClone(PROPIAS); alterar(malo);
    await assert.rejects(crearClienteNotificacionesCronosHTTP({ fetchImpl: async () => respuestaJSON(200, malo) }).consultarPropias(),
      (e) => e instanceof ErrorClienteNotificacionesCronos && e.codigo === "respuesta_incompatible", nombre);
  }
  const malaBandeja = structuredClone(BANDEJA); malaBandeja.notificaciones[0].atendida = true;
  await assert.rejects(crearClienteNotificacionesCronosHTTP({ fetchImpl: async () => respuestaJSON(200, malaBandeja) }).consultarBandeja(), (e) => e.codigo === "respuesta_incompatible");
});

test("el envío manda sólo tipo, fecha, texto, referencia y huella y exige su recibo", async () => {
  let enviado;
  const recibo = { recibo: { notificacion_ref: REF, recibo_ref: "recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f", instante_utc: "2026-09-25T08:00:00Z", replay: false } };
  const cliente = crearClienteNotificacionesCronosHTTP({ fetchImpl: async (url, o) => { enviado = { url, ...o }; return respuestaJSON(201, recibo); } });
  assert.equal((await cliente.enviar(ENTRADA)).notificacion_ref, REF);
  assert.equal(enviado.url, RUTAS_NOTIFICACIONES_CRONOS.envios);
  assert.deepEqual(JSON.parse(enviado.body), ENTRADA);
  const { adjunto_ref: _r, adjunto_sha256: _h, ...sinAdjunto } = ENTRADA;
  await cliente.enviar(sinAdjunto);
  assert.deepEqual(Object.keys(JSON.parse(enviado.body)).sort(), ["clave_operacion", "fecha_referida", "texto", "tipo_version_ref"]);
  for (const malo of [{ ...ENTRADA, adjunto_sha256: "" }, { ...ENTRADA, adjunto_sha256: "0".repeat(64) }, { ...ENTRADA, texto: "  " }, { ...ENTRADA, texto: "a\u0007" },
    { ...ENTRADA, texto: "x".repeat(513) }, { ...ENTRADA, fecha_referida: "2026-02-30" }, { ...ENTRADA, empleado_ref: "emp_x" }, { ...ENTRADA, clave_operacion: "x" }]) {
    await assert.rejects(cliente.enviar(malo), TypeError);
  }
  assert.ok(textoNotificacionValido("ñ".repeat(512)));
  assert.ok(textoNotificacionValido("dos\nlíneas\tcon tabulador"));
});

test("los rechazos nominales llegan con su código", async () => {
  for (const [estado, cuerpo, codigo] of [[409, { error: "tipo_no_vigente" }, "tipo_no_vigente"], [409, { error: "conflicto" }, "conflicto"],
    [409, { error: "estado_cambiado" }, "estado_cambiado"], [403, { error: "no_competente" }, "no_competente"], [403, { error: "sin_empleado" }, "sin_empleado"],
    [400, { error: "peticion_invalida" }, "peticion_invalida"], [503, { error: "no_disponible" }, "servicio_no_disponible"]]) {
    const cliente = crearClienteNotificacionesCronosHTTP({ fetchImpl: async () => respuestaJSON(estado, cuerpo) });
    await assert.rejects(cliente.enviar(ENTRADA), (e) => e instanceof ErrorClienteNotificacionesCronos && e.codigo === codigo, `${estado} ${cuerpo.error}`);
  }
  const grande = crearClienteNotificacionesCronosHTTP({ fetchImpl: async () => respuestaJSON(409, { error: "bandeja_demasiado_grande" }) });
  await assert.rejects(grande.consultarBandeja(), (e) => e instanceof ErrorClienteNotificacionesCronos && e.codigo === "bandeja_demasiado_grande");
  const atencion = { recibo: { atencion_ref: "notificacion:cronos:atencion:0f0e0d0c-0b0a-4000-8000-000000000002", notificacion_ref: REF,
    recibo_ref: "recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f", instante_utc: "2026-09-25T08:00:00Z", replay: true } };
  const cliente = crearClienteNotificacionesCronosHTTP({ fetchImpl: async () => respuestaJSON(200, atencion) });
  assert.equal((await cliente.atender({ clave_operacion: "ate-r-00001", notificacion_ref: REF })).replay, true);
  await assert.rejects(cliente.atender({ clave_operacion: "ate-r-00001", notificacion_ref: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000009" }),
    (e) => e.codigo === "respuesta_incompatible", "un recibo de otra notificación no vale");
});

test("la huella del documento se calcula en el navegador y el contenido no se conserva", async () => {
  const bytes = new TextEncoder().encode("documento sintético");
  let leido;
  const archivo = { size: bytes.byteLength, arrayBuffer: async () => { leido = new Uint8Array(bytes).buffer; return leido; } };
  const huella = await calcularHuellaDocumentoCronos(archivo, webcrypto);
  assert.match(huella, /^[0-9a-f]{64}$/u);
  assert.ok(new Uint8Array(leido).every((b) => b === 0), "los bytes leídos se borran");
  await assert.rejects(calcularHuellaDocumentoCronos({ size: 0, arrayBuffer: async () => new ArrayBuffer(0) }, webcrypto), TypeError);
  await assert.rejects(calcularHuellaDocumentoCronos({ size: 11 * 1024 * 1024, arrayBuffer: async () => new ArrayBuffer(1) }, webcrypto), TypeError);
});

test("el cliente no guarda nada en el navegador ni sube el documento", async () => {
  const fuente = await readFile(new URL("./cliente-notificaciones-http.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|credentials:\s*"include"|FormData/u);
});
