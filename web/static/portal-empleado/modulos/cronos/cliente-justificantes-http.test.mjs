import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteJustificantesCronosHTTP, ErrorClienteJustificantesCronos,
  RUTAS_JUSTIFICANTES_CRONOS } from "./cliente-justificantes-http.js";

const ref = (c) => `ref:${c.repeat(64)}`;
const entrada = { solicitud_ref: "permiso:cronos:solicitud:ensayo001", clave_operacion: ref("a"),
  documento_id: ref("b"), documento_version: 2, sha256: "c".repeat(64), version_esperada: 0 };
const recibo = { solicitud_ref: entrada.solicitud_ref, documento_id: entrada.documento_id,
  documento_version: 2, sha256: entrada.sha256, estado: "pendiente_revision", version: 1,
  recibo_ref: "recibo:cronos:ensayo001", instante_utc: "2026-10-14T10:00:00Z", replay: false };
const respuesta = (status, body) => new Response(JSON.stringify(body),
  { status, headers: { "content-type": "application/json; charset=utf-8" } });

test("aporte usa la ruta real y exige recibo ligado al documento y la huella", async () => {
  let llamada;
  const cliente = crearClienteJustificantesCronosHTTP({ fetchImpl: async (url, opciones) => {
    llamada = { url, opciones }; return respuesta(201, { recibo });
  } });
  assert.equal((await cliente.aportar(entrada)).recibo_ref, recibo.recibo_ref);
  assert.equal(llamada.url, RUTAS_JUSTIFICANTES_CRONOS.aportar);
  assert.equal(JSON.parse(llamada.opciones.body).documento_version, "2");
  assert.equal(llamada.opciones.redirect, "error");
  assert.equal(llamada.opciones.cache, "no-store");
  const alterado = crearClienteJustificantesCronosHTTP({ fetchImpl: async () => respuesta(201,
    { recibo: { ...recibo, sha256: "d".repeat(64) } }) });
  await assert.rejects(alterado.aportar(entrada), (e) => e instanceof ErrorClienteJustificantesCronos && e.codigo === "respuesta_incompatible");
});

test("enlace documental pendiente no se muestra como recibo", async () => {
  const cliente = crearClienteJustificantesCronosHTTP({ fetchImpl: async () => respuesta(202, { enlace_pendiente: true }) });
  assert.deepEqual(await cliente.aportar(entrada), { enlace_pendiente: true });
});
