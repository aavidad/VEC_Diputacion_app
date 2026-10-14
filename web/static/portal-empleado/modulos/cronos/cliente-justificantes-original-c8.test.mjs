import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteOriginalC8HTTP, ErrorClienteOriginalC8, RUTAS_ORIGINAL_C8 } from "./cliente-justificantes-original-c8.js";

const SOLICITUD = "permiso:cronos:solicitud:abcdefgh";
const DOCUMENTO = `ref:${"a".repeat(64)}`;
const HUELLA = "b".repeat(64);
const CLAVE = `ref:${"c".repeat(64)}`;
const entrada = { solicitud_ref: SOLICITUD, clave_operacion: CLAVE, version_esperada: 0,
  documento_id: DOCUMENTO, documento_version: 1 };
const recibo = { recibo: { solicitud_ref: SOLICITUD, documento_id: DOCUMENTO, documento_version: 1,
  sha256: HUELLA, estado: "pendiente_revision", version: 1,
  recibo_ref: "recibo:cronos:00000000-0000-4000-8000-000000000001",
  instante_utc: "2026-10-14T18:00:00Z", replay: false } };

test("C8 envía solo el identificador y la versión del original admitido; coteja el recibo", async () => {
  const llamadas = [];
  const cliente = crearClienteOriginalC8HTTP({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return new Response(JSON.stringify(recibo), { status: 201, headers: { "content-type": "application/json" } });
  } });
  const r = await cliente.aportar(entrada);
  assert.equal(r.sha256, HUELLA);
  assert.equal(llamadas[0].ruta, RUTAS_ORIGINAL_C8.aportar);
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), {
    solicitud_ref: SOLICITUD, clave_operacion: CLAVE, version_esperada: "0",
    documento_id: DOCUMENTO, documento_version: "1",
  });
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  for (const malo of [{ ...entrada, sha256: HUELLA }, { ...entrada, persona_ref: "per_ajena" },
    { ...entrada, documento_version: 0 }, { ...entrada, solicitud_ref: "permiso:cronos:solicitud:x" }]) {
    assert.throws(() => cliente.aportar(malo), TypeError);
  }
});

test("C8 falla cerrado ante recibo de otro documento y conserva conflicto nominal", async () => {
  const ajeno = structuredClone(recibo); ajeno.recibo.documento_id = `ref:${"d".repeat(64)}`;
  const cliente = crearClienteOriginalC8HTTP({ fetchImpl: async () => new Response(JSON.stringify(ajeno),
    { status: 201, headers: { "content-type": "application/json" } }) });
  await assert.rejects(cliente.aportar(entrada), (e) => e instanceof ErrorClienteOriginalC8 && e.codigo === "respuesta_incompatible");
  const conflicto = crearClienteOriginalC8HTTP({ fetchImpl: async () => new Response(JSON.stringify({ error: "estado_cambiado" }),
    { status: 409, headers: { "content-type": "application/json" } }) });
  await assert.rejects(conflicto.aportar(entrada), (e) => e instanceof ErrorClienteOriginalC8 && e.codigo === "estado_cambiado");
});

test("C8 recupera recibos históricos solo por GET autorizado y rechaza filas extra", async () => {
  const detalle = { solicitud_ref: SOLICITUD, estado_permiso: "concedido", version_permiso: 3,
    pendiente_justificar: true, historia: [{ version: 1, estado: "pendiente_revision", motivo_ref: "",
      documento_id: DOCUMENTO, documento_version: 1, sha256: HUELLA, recibo_documento_ref: "recibo:documentos:abc",
      custodio_id: "almacen.local", recibo_ref: recibo.recibo.recibo_ref, instante_utc: recibo.recibo.instante_utc }] };
  const bandeja = { pendientes: [{ solicitud_ref: SOLICITUD, empleado_etiqueta: "Elena Ruiz Martín",
    documento_id: DOCUMENTO, documento_version: 1, sha256: HUELLA, estado: "pendiente_revision",
    version: 1, registrada_en: recibo.recibo.instante_utc }] };
  const llamadas = [];
  const cliente = crearClienteOriginalC8HTTP({ fetchImpl: async (url, opciones) => {
    llamadas.push({ url, opciones });
    return new Response(JSON.stringify(url === RUTAS_ORIGINAL_C8.bandeja ? bandeja : detalle),
      { status: 200, headers: { "content-type": "application/json" } });
  } });
  assert.equal((await cliente.consultarPropia(SOLICITUD)).historia[0].recibo_ref, recibo.recibo.recibo_ref);
  assert.equal((await cliente.consultarDetalleRRHH(SOLICITUD)).historia.length, 1);
  assert.equal((await cliente.consultarBandeja()).pendientes.length, 1);
  assert.deepEqual(llamadas.map((c) => c.opciones.method), ["GET", "GET", "GET"]);
  assert.ok(llamadas.every((c) => c.opciones.credentials === "same-origin" && c.opciones.redirect === "error"));
  const ajeno = structuredClone(detalle); ajeno.historia[0].persona_ref = "per_ajena";
  const fallido = crearClienteOriginalC8HTTP({ fetchImpl: async () => new Response(JSON.stringify(ajeno),
    { status: 200, headers: { "content-type": "application/json" } }) });
  await assert.rejects(fallido.consultarPropia(SOLICITUD), (e) => e instanceof ErrorClienteOriginalC8 && e.codigo === "respuesta_incompatible");
});
