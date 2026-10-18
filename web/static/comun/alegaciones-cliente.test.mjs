import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteAlegaciones } from "./alegaciones-cliente.js";

const objeto = { tipo: "decision_inscripcion", solicitud_ref: `solicitud_inscripcion_${"a".repeat(64)}`, acto_ref: "decision:uno", version_acto: 2 };
const orden = { objeto, fundamento: "Solicito revisar la decisión sobre mi solicitud.", clave_idempotencia: "12345678-1234-4234-8234-123456789012" };
const recibo = { esquema: "vec.bolsa.alegacion.presentada.v1", alegacion_ref: "alegacion:uno", objeto,
  fundamento: orden.fundamento, estado: "presentada", presentada_en: "2026-10-13T01:00:00Z", recibo_ref: "recibo:uno", repetida: true };
const respuesta = (data) => new Response(JSON.stringify({ data }), { status: 200, headers: { "Content-Type": "application/json" } });

test("una respuesta incierta no reenvía sola y el reintento conserva la orden", async () => {
  const peticiones = [];
  const cliente = crearClienteAlegaciones({ fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, cuerpo: JSON.parse(opciones.body) });
    if (peticiones.length === 1) throw new TypeError("red");
    return respuesta(recibo);
  } });
  await assert.rejects(cliente.presentar(orden), (e) => e.codigo === "red");
  assert.equal(peticiones.length, 1);
  assert.equal((await cliente.presentar(orden)).recibo_ref, "recibo:uno");
  assert.deepEqual(peticiones[0], peticiones[1]);
});

test("no confirma un recibo correspondiente a otra decisión", async () => {
  const cliente = crearClienteAlegaciones({ fetchImpl: async () => respuesta({ ...recibo, objeto: { ...objeto, acto_ref: "decision:otra" } }) });
  await assert.rejects(cliente.presentar(orden), (e) => e.codigo === "respuesta_no_valida");
});

test("las etiquetas y la identidad no pasan al cuerpo del acto", async () => {
  let cuerpo;
  const cliente = crearClienteAlegaciones({ fetchImpl: async (_ruta, opciones) => {
    cuerpo = JSON.parse(opciones.body); return respuesta(recibo);
  } });
  await cliente.presentar({ ...orden, objeto: { ...objeto, titulo: "Auxiliar administrativo", persona_ref: "persona:ajena" } });
  assert.deepEqual(cuerpo, orden);
});
