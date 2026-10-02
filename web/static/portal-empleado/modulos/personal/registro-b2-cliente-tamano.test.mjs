import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteRegistroB2 } from "./registro-b2-cliente.js";

const corte = { vigenteEn: "2026-09-20", conocidoEn: "2026-09-20T10:00:00.000000Z" };
const consulta = { ...corte, empleadoRef: "emp_" + "e".repeat(24) };
function respuesta(trozos, { declarada, status = 200 } = {}) {
  const estado = { lecturas: 0, cancelaciones: 0, liberaciones: 0 };
  let indice = 0;
  const lector = {
    async read() { estado.lecturas++; return indice < trozos.length ? { done: false, value: trozos[indice++] } : { done: true }; },
    async cancel() { estado.cancelaciones++; },
    releaseLock() { estado.liberaciones++; },
  };
  return {
    estado,
    status, ok: status === 200, redirected: false,
    headers: { get(nombre) { return nombre === "content-type" ? "application/json; charset=utf-8" : nombre === "content-length" && declarada !== undefined ? String(declarada) : null; } },
    body: { getReader() { return lector; }, cancel: lector.cancel },
  };
}
const clientePara = (r) => crearClienteRegistroB2({ fetchImpl: async () => r });
const excesiva = (e) => e.codigo === "respuesta_excesiva";

test("ficha rechaza y cancela Content-Length superior a 4 MiB sin leer", async () => {
  const r = respuesta([], { declarada: 4 * 1024 * 1024 + 1 });
  await assert.rejects(() => clientePara(r).consultarFicha(consulta), excesiva);
  assert.equal(r.estado.cancelaciones, 1);
  assert.equal(r.estado.lecturas, 0);
});

test("ficha comprueba los bytes reales antes de copiar, aunque la longitud declare menos", async () => {
  for (const declarada of [undefined, 1]) {
    const r = respuesta([new Uint8Array(4 * 1024 * 1024), new Uint8Array(1)], { declarada });
    await assert.rejects(() => clientePara(r).consultarFicha(consulta), excesiva);
    assert.equal(r.estado.cancelaciones, 1);
    assert.equal(r.estado.lecturas, 2);
    assert.equal(r.estado.liberaciones, 1);
  }
});

test("listas conservan 512 KiB y 256 fragmentos", async () => {
  for (const operacion of ["listarVacantes", "listarEmpleados"]) {
    for (const r of [
      respuesta([], { declarada: 512 * 1024 + 1 }),
      respuesta([new Uint8Array(512 * 1024 + 1)]),
      respuesta(Array.from({ length: 257 }, () => new Uint8Array([32]))),
    ]) {
      await assert.rejects(() => clientePara(r)[operacion](corte), excesiva);
      assert.equal(r.estado.cancelaciones, 1);
    }
  }
});

test("el error 503 de ficha conserva el lector de 512 KiB", async () => {
  const r = respuesta([], { status: 503, declarada: 600_077 });
  await assert.rejects(() => clientePara(r).consultarFicha(consulta), (e) => e.codigo === "estado_no_valido" && e.estado === 503);
  assert.equal(r.estado.cancelaciones, 1);
  assert.equal(r.estado.lecturas, 0);
});

test("POST conserva 512 KiB y 256 fragmentos y cancela el resultado incierto", async () => {
  const alta = {
    persona_ref: "per_" + "p".repeat(24), organismo_ref: "organismo:sintetico", unidad_ref: "unidad:sintetica",
    regimen: { ref: "regimen:sintetico", version: 1 }, modalidad: { ref: "modalidad:sintetica", version: 1 },
    vigente_desde: "2026-09-20", acto_ref: "acto:sintetico", fuente_ref: "fuente:sintetica", fuente_version: 1, fuente_huella_sha256: "a".repeat(64),
  };
  for (const r of [
    respuesta([], { status: 201, declarada: 512 * 1024 + 1 }),
    respuesta([new Uint8Array(512 * 1024 + 1)], { status: 201 }),
    respuesta(Array.from({ length: 257 }, () => new Uint8Array([32])), { status: 201 }),
  ]) {
    await assert.rejects(() => clientePara(r).registrarAlta(alta, { claveIdempotencia: "11111111-1111-4111-8111-111111111111" }), (e) => e.codigo === "resultado_incierto");
    assert.equal(r.estado.cancelaciones, 1);
  }
});
