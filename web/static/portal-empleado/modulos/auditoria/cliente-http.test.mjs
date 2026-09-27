import assert from "node:assert/strict";
import test from "node:test";
import { crearFuenteAuditoriaHTTP } from "./cliente-http.js";

const consulta = {
  expediente_ref: "exp_1", desde: "2026-09-01T00:00:00Z", hasta: "2026-09-10T00:00:00Z",
  actor_ref: "per_2", finalidad: "fin_control", motivo: "mot_revision",
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
  assert.deepEqual(JSON.parse(llamada.opciones.body), { ...consulta, limite: 50 });
});

test("consulta fuera de ámbito temporal o sin finalidad positiva no usa red", async () => {
  let llamadas = 0;
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async () => { ++llamadas; throw Error("red"); } });
  await assert.rejects(fuente.consultar({ ...consulta, finalidad: "" }), { codigo: "consulta_invalida" });
  await assert.rejects(fuente.consultar({ ...consulta, hasta: "2026-11-01T00:00:00Z" }), { codigo: "consulta_invalida" });
  await assert.rejects(fuente.consultar({ ...consulta, expediente_ref: "*" }), { codigo: "consulta_invalida" });
  assert.equal(llamadas, 0);
});

test("403 deniega sin leer cuerpo ni propagar contenido del servidor", async () => {
  const fuente = crearFuenteAuditoriaHTTP({ fetchImpl: async () => new Response("dato protegido", { status: 403 }) });
  await assert.rejects(fuente.consultar(consulta), { codigo: "denegado", estado: 403 });
});
