import assert from "node:assert/strict";
import test from "node:test";

import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";

const expediente_ref = "expediente:ct:anotacion";
const clave_idempotencia = "8e2ee4bf-5c95-48da-a0ed-f1d8ca2c4a50";
const recibo = Object.freeze({
  operacion: "registrar_anotacion_administrativa", organizacion_ref: "organizacion:ct:1",
  expediente_ref, version_anterior: 8, version_resultante: 9,
  fase_resultante: "nombramiento", estado_resultante: "en_curso",
  seguimiento_original: { seguimiento_ref: "seguimiento:ct:1", version_seguimiento: 1, huella_raiz_seguimiento_sha256: "a".repeat(64) },
  recibo_ref: "recibo:anotacion:1", auditoria_ref: "auditoria:ct:1", evento_ref: "evento:ct:1",
  actor_ref: "actor:ct:1", registrada_en: "2026-09-12T10:00:00Z",
});
const respuesta = (data, status = 200) => new Response(JSON.stringify(data), {
  status, headers: { "content-type": "application/json; charset=utf-8" },
});

test("anotación: POST y recuperación conservan la intención sintética y el recibo", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return respuesta({ data: recibo }, opciones.method === "POST" ? 201 : 200);
  } });
  const solicitud = { expediente_ref, version_esperada: 8, clave_idempotencia, observaciones: "Anotación sintética de prueba." };
  assert.deepEqual(await cliente.anotacionAdministrativa.registrar(solicitud), recibo);
  assert.equal(llamadas[0][0], "/api/vec/contratacion-temporal/expedientes/anotaciones-administrativas");
  assert.equal(llamadas[0][1].method, "POST");
  assert.deepEqual(JSON.parse(llamadas[0][1].body), solicitud);
  assert.deepEqual(await cliente.anotacionAdministrativa.recuperar({ expediente_ref, clave_idempotencia }, { version_observada: 8 }), recibo);
  assert.equal(llamadas[1][0], "/api/vec/contratacion-temporal/expedientes/anotaciones-administrativas/recuperacion?expediente_ref=expediente%3Act%3Aanotacion&clave_idempotencia=8e2ee4bf-5c95-48da-a0ed-f1d8ca2c4a50");
  assert.equal(llamadas[1][1].method, "GET");
  assert.equal(llamadas[1][1].body, undefined);
});

test("anotación: un recibo ajeno no convierte una respuesta en confirmación", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data: { ...recibo, expediente_ref: "expediente:ct:ajeno" } }, 201) });
  await assert.rejects(cliente.anotacionAdministrativa.registrar({ expediente_ref, version_esperada: 8, clave_idempotencia, observaciones: "Anotación sintética de prueba." }));
});
