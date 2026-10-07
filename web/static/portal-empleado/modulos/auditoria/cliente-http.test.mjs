import assert from "node:assert/strict";
import test from "node:test";
import { crearFuenteAuditoriaHTTP } from "./cliente-http.js";

const consulta = {
  fuente: "ct", expediente_ref: "exp_1", desde: "2026-09-01T00:00:00Z", hasta: "2026-09-10T00:00:00Z",
  actor_ref: "", finalidad_ref: "fin_control", motivo_ref: "catalogo:v1:mot_revision",
};

test("POST mismo origen con referencias exactas y sin credenciales en URL", async () => {
  let llamada;
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async (ruta, opciones) => {
    llamada = { ruta, opciones };
    return new Response(JSON.stringify({ registros: [], siguiente_cursor: "" }), {
      status: 200, headers: { "Content-Type": "application/json" },
    });
  } });
  const resultado = await fuente.consultar(consulta);
  assert.deepEqual(resultado, { registros: [], siguiente_cursor: "" });
  assert.equal(llamada.ruta, "/api/vec/auditoria/consultas");
  assert.equal(llamada.opciones.method, "POST");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.cache, "no-store");
  assert.equal(llamada.opciones.redirect, "error");
  assert.deepEqual(JSON.parse(llamada.opciones.body), { ...consulta, limite: 50, cursor: "" });
});

test("consulta fuera de ámbito temporal o sin finalidad positiva no usa red", async () => {
  let llamadas = 0;
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async () => { ++llamadas; throw Error("red"); } });
  await assert.rejects(fuente.consultar({ ...consulta, finalidad_ref: "" }), { codigo: "consulta_invalida" });
  await assert.rejects(fuente.consultar({ ...consulta, hasta: "2026-11-01T00:00:00Z" }), { codigo: "consulta_invalida" });
  await assert.rejects(fuente.consultar({ ...consulta, expediente_ref: "*" }), { codigo: "consulta_invalida" });
  await assert.rejects(fuente.consultar({ ...consulta, fuente: "libre" }), { codigo: "consulta_invalida" });
  assert.equal(llamadas, 0);
});

test("GET opciones autenticadas entrega únicamente refs validadas", async () => {
  let llamada;
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async (ruta, opciones) => {
    llamada = { ruta, opciones };
    return new Response(JSON.stringify({ finalidad_ref: "fin_1", motivo_ref: "catalogo:v1:revision", fuentes: ["ct", "bolsa"],
      permiso_requerido: "auditoria.consultar", es_ejemplo: true }), {
      status: 200, headers: { "Content-Type": "application/json" },
    });
  } });
  const opciones = await fuente.obtenerOpciones();
  assert.equal(opciones.motivo_ref, "catalogo:v1:revision");
  assert.deepEqual(opciones.fuentes, ["ct", "bolsa"]);
  assert.equal(llamada.ruta, "/api/vec/auditoria/opciones");
  assert.equal(llamada.opciones.method, "GET");
  assert.equal(llamada.opciones.body, undefined);
  assert.equal(llamada.opciones.credentials, "same-origin");
});

test("opciones 403 o mal formadas cierran consulta sin usar refs inventadas", async () => {
  const denegada = crearFuenteAuditoriaHTTP({ fetchImpl: async () => new Response("secreto", { status: 403 }) });
  await assert.rejects(denegada.obtenerOpciones(), { codigo: "denegado", estado: 403 });
  const malformada = crearFuenteAuditoriaHTTP({ fetchImpl: async () => new Response(
    JSON.stringify({ finalidad_ref: "fin_1", motivo_ref: "*", permiso_requerido: "auditoria.consultar", es_ejemplo: true }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  ) });
  await assert.rejects(malformada.obtenerOpciones(), { codigo: "respuesta_invalida" });
  const sinFuente = crearFuenteAuditoriaHTTP({ fetchImpl: async () => new Response(
    JSON.stringify({ finalidad_ref: "fin_1", motivo_ref: "catalogo:v1:revision", permiso_requerido: "auditoria.consultar", es_ejemplo: true }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  ) });
  await assert.rejects(sinFuente.obtenerOpciones(), { codigo: "respuesta_invalida" });
});

test("403 deniega sin leer cuerpo ni propagar contenido del servidor", async () => {
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async () => new Response("dato protegido", { status: 403 }) });
  await assert.rejects(fuente.consultar(consulta), { codigo: "denegado", estado: 403 });
});

test("404 indica capacidad no disponible sin leer el cuerpo ni convertirlo en denegación", async () => {
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async () => new Response("dato protegido", { status: 404 }) });
  await assert.rejects(fuente.obtenerOpciones(), { codigo: "no_disponible", estado: 404 });
  await assert.rejects(fuente.consultar(consulta), { codigo: "no_disponible", estado: 404 });
});
