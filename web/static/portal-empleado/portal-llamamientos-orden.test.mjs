import test from "node:test";
import assert from "node:assert/strict";
import { prepararTextosPortal } from "./portal-i18n.js?v=VEC_WEB_VERSION";
await prepararTextosPortal("bolsa");
const { emitirLlamamiento, mensajeFueraDeOrden } = await import("./portal-llamamientos-operaciones-api.js?v=VEC_WEB_VERSION");

test("F1-110 el 409 por orden dice quién va antes sin confundirlo con la clave repetida", async () => {
  const payload = { bolsa_ref: "bolsa:01", participaciones: ["participacion:03"], configuracion: { referencia: "NEC-01" }, clave_idempotencia: "b7-emision-0003" };
  const responder = (error) => ({ fetchImpl: async () => ({ ok: false, status: 409, json: async () => ({ error }) }) });
  const salida = await emitirLlamamiento(payload, responder({ codigo: "llamamiento_fuera_de_orden", motivos_salto: [], participacion_ref: "participacion:02", orden_vigente: 2 }));
  assert.equal(salida.codigo, "llamamiento_fuera_de_orden");
  assert.deepEqual(salida.antes, { participacion_ref: "participacion:02", orden_vigente: 2 });
  assert.match(salida.mensaje, /número de orden 2\b/);
  const conNombre = mensajeFueraDeOrden(salida.antes, [{ participacion_ref: "participacion:02", nombre_visible: "Lucía Ortega Ruiz" }]);
  assert.match(conNombre, /Antes va Lucía Ortega Ruiz, con el número de orden 2,/);
  const sinDatos = await emitirLlamamiento(payload, responder({ codigo: "llamamiento_fuera_de_orden", motivos_salto: [] }));
  assert.equal(sinDatos.antes, null);
  assert.doesNotMatch(sinDatos.mensaje, /\{|undefined|null/);
  const clave = await emitirLlamamiento(payload, responder({ codigo: "conflicto" }));
  assert.notEqual(clave.codigo, "llamamiento_fuera_de_orden");
  assert.notEqual(clave.mensaje, salida.mensaje);
});
