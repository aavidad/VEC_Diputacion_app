import assert from "node:assert/strict";
import test from "node:test";
import { createHash, webcrypto } from "node:crypto";
import { IDIOMA_ACTUAL, IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";
import { crearClienteExportacionServicios, MIME_EXPORTACION_SERVICIOS, RUTA_EXPORTACION_SERVICIOS, MAXIMO_EXPORTACION_SERVICIOS } from "./cliente-http-exportacion-servicios.js";

const reciboRef = "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100";
const corte = Object.freeze({ vigente_en: "2026-10-01", conocido_en: "2026-10-03T08:59:59.123456Z" });
const bytes = new TextEncoder().encode('\uFEFF"Inicio";"Fin"\r\n"2020-01-01";"2020-12-31"\r\n');
const hash = (datos) => createHash("sha256").update(datos).digest("hex");
function respuesta(datos = bytes, cambios = {}) {
  return new Response(datos, { headers: { "Content-Type": MIME_EXPORTACION_SERVICIOS, "Content-Length": String(datos.byteLength), "Content-Disposition": 'attachment; filename="servicios.csv"', "X-Content-SHA256": hash(datos), "X-Recibo-Ref": reciboRef, ...cambios } });
}
function fallo(estado, codigo) {
  const cuerpo = JSON.stringify({ error: codigo });
  return new Response(cuerpo, { status: estado, headers: { "Content-Type": "application/json; charset=utf-8", "Content-Length": String(new TextEncoder().encode(cuerpo).byteLength) } });
}

test("exporta el recibo y corte exactos por POST mismo origen, sin otra consulta ni filas", async () => {
  const llamadas = [];
  const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async (...args) => { llamadas.push(args); return respuesta(); } });
  const archivo = await cliente.exportar({ reciboRef, corte, persona: "persona_oculta", items: ["fila_oculta"] });
  assert.equal(llamadas.length, 1);
  const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, RUTA_EXPORTACION_SERVICIOS);
  assert.deepEqual([opciones.method, opciones.credentials, opciones.mode, opciones.cache, opciones.redirect, opciones.referrerPolicy], ["POST", "same-origin", "same-origin", "no-store", "error", "no-referrer"]);
  assert.deepEqual(JSON.parse(opciones.body), { recibo_ref: reciboRef, corte, idioma: IDIOMA_ACTUAL });
  assert.deepEqual(archivo.bytes, bytes); assert.equal(archivo.huella, hash(bytes)); assert.equal(archivo.reciboRef, reciboRef);
  assert.equal(archivo.nombre, "servicios.csv"); assert.equal(archivo.mime, MIME_EXPORTACION_SERVICIOS);
  assert.ok(Object.isFrozen(archivo));
});

test("cada idioma procede del índice común, sin cambiar el corte ni renovar GET", async () => {
  const idiomas = [];
  const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async (_ruta, { body }) => { idiomas.push(JSON.parse(body).idioma); return respuesta(); } });
  for (const { codigo } of IDIOMAS_DISPONIBLES) await cliente.exportar({ reciboRef, corte, idioma: codigo });
  assert.deepEqual(idiomas, IDIOMAS_DISPONIBLES.map(({ codigo }) => codigo));
});

test("metadatos ausentes o inválidos se rechazan antes del POST", async () => {
  let llamadas = 0;
  const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async () => { llamadas += 1; return respuesta(); } });
  for (const entrada of [{}, { reciboRef, corte: null }, { reciboRef, corte: { ...corte, vigente_en: "2025-02-29" } }, { reciboRef, corte: { ...corte, empleado: "emp_x" } }, { reciboRef, corte, idioma: "idioma_inventado" }]) {
    await assert.rejects(cliente.exportar(entrada), (e) => e.codigo === "sin_consulta");
  }
  assert.equal(llamadas, 0);
});

test("tipo, archivo, recibo, longitud y huella se comprueban antes de entregar bytes", async () => {
  for (const cambios of [
    { "Content-Type": "text/html" }, { "Content-Disposition": 'attachment; filename="../privado.csv"' },
    { "X-Recibo-Ref": "fichapropia:ffffffff-ffff-ffff-ffff-ffffffffffff" }, { "X-Content-SHA256": "a".repeat(64) },
    { "Content-Length": "0" }, { "Content-Length": String(MAXIMO_EXPORTACION_SERVICIOS + 1) }, { "Content-Length": String(bytes.length + 1) },
  ]) {
    const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async () => respuesta(bytes, cambios) });
    await assert.rejects(cliente.exportar({ reciboRef, corte }), (e) => e.codigo === "respuesta_no_valida");
  }
  const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async () => {
    const r = respuesta(); r.headers.delete("X-Content-SHA256"); return r;
  } });
  await assert.rejects(cliente.exportar({ reciboRef, corte }), (e) => e.codigo === "respuesta_no_valida");
});

test("errores cerrados no se descargan ni lanzan GET; cuerpo ajeno queda opaco", async () => {
  for (const [estado, codigo, esperado] of [[400, "peticion_invalida", "sin_consulta"], [403, "acceso_denegado", "denegado"], [503, "no_disponible", "no_disponible"], [503, "secreto_privado", "respuesta_no_valida"]]) {
    const llamadas = [];
    const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async (ruta) => { llamadas.push(ruta); return fallo(estado, codigo); } });
    await assert.rejects(cliente.exportar({ reciboRef, corte }), (e) => e.codigo === esperado && !e.message.includes("secreto"));
    assert.deepEqual(llamadas, [RUTA_EXPORTACION_SERVICIOS]);
  }
});

test("red, plazo y cancelación quedan separados; cancelación temprana no hace POST", async () => {
  const red = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async () => { throw new Error("detalle_red"); } });
  await assert.rejects(red.exportar({ reciboRef, corte }), (e) => e.codigo === "no_disponible");
  const controlador = new AbortController(); let llamadas = 0;
  const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: (_ruta, { signal }) => {
    llamadas += 1; return new Promise((_resolver, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("aborto"))));
  } });
  const pendiente = cliente.exportar({ reciboRef, corte, signal: controlador.signal }); controlador.abort();
  await assert.rejects(pendiente, (e) => e.codigo === "operacion_abortada");
  await assert.rejects(cliente.exportar({ reciboRef, corte, signal: controlador.signal }), (e) => e.codigo === "operacion_abortada");
  assert.equal(llamadas, 1);
  const plazo = crearClienteExportacionServicios({ cryptoImpl: webcrypto, plazoMs: 5, fetchImpl: (_ruta, { signal }) => new Promise((_resolver, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("plazo")))) });
  await assert.rejects(plazo.exportar({ reciboRef, corte }), (e) => e.codigo === "no_disponible");
});

test("un cuerpo mayor de lo declarado cancela la lectura sin entregar bytes", async () => {
  let cancelado = false;
  const stream = new ReadableStream({ start(controller) { controller.enqueue(new Uint8Array(16)); }, cancel() { cancelado = true; } });
  const cliente = crearClienteExportacionServicios({ cryptoImpl: webcrypto, fetchImpl: async () => new Response(stream, { headers: { "Content-Type": MIME_EXPORTACION_SERVICIOS, "Content-Disposition": 'attachment; filename="servicios.csv"', "Content-Length": "8", "X-Content-SHA256": "a".repeat(64), "X-Recibo-Ref": reciboRef } }) });
  await assert.rejects(cliente.exportar({ reciboRef, corte }), (e) => e.codigo === "respuesta_no_valida");
  assert.equal(cancelado, true);
});
