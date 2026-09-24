import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteRectificacionDietasHTTP } from "./cliente-rectificacion-http.js";

const solicitud = Object.freeze({ relacion_ref: `rel_${"a".repeat(22)}`, unidad_ref: "unidad-personal", asignacion_ref: `ads_${"b".repeat(22)}`, version_esperada: 3, fecha_referencia: "2026-09-24", clave_idempotencia: "rectificacion-dietas-0001", campos_a_revisar: ["unidad_ref", "centro_ref"], motivo_revision: "Centro incorrecto", detalle_solicitado: "Revise el dato vigente" });
const resultado = Object.freeze({ solicitud_ref: `srd_${"a".repeat(32)}`, recibo_ref: `rrd_${"b".repeat(32)}`, estado: "pendiente", registrada_en: "2026-09-24T10:00:00.000000Z", asignacion_ref: solicitud.asignacion_ref, version_origen: 3, decision_ref: "decision:rectificacion", efecto_ref: "efecto:rectificacion", consumo_huella_sha256: "c".repeat(64), auditoria_ad3_ref: "ad3:rectificacion" });
const json = (cuerpo, estado = 200) => new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } });

test("solicita la rectificación con el contrato exacto y ordena sus campos", async () => {
  let llamada; const cliente = crearClienteRectificacionDietasHTTP({ fetchImpl: async (...args) => { llamada = args; return json(resultado, 201); } });
  const recibido = await cliente.solicitar(solicitud);
  assert.equal(llamada[0], "/api/vec/personal/solicitudes-rectificacion-dietas"); assert.equal(llamada[1].method, "POST"); assert.equal(llamada[1].headers.Accept, "application/json"); assert.equal(llamada[1].headers["Content-Type"], "application/json; charset=utf-8");
  assert.deepEqual(JSON.parse(llamada[1].body).campos_a_revisar, ["centro_ref", "unidad_ref"]); assert.equal(recibido.recibo_ref, resultado.recibo_ref);
});

test("consulta solamente con los tres parámetros canónicos y distingue 404", async () => {
  const llamadas = []; const cliente = crearClienteRectificacionDietasHTTP({ fetchImpl: async (ruta) => { llamadas.push(ruta); return json({ error: "personal.error.no_encontrada" }, 404); } });
  await assert.rejects(() => cliente.consultar({ relacion_ref: solicitud.relacion_ref, unidad_ref: solicitud.unidad_ref, fecha_referencia: solicitud.fecha_referencia }), (error) => error.codigo === "no_encontrada");
  assert.equal(llamadas[0], `/api/vec/personal/solicitudes-rectificacion-dietas?relacion_ref=${encodeURIComponent(solicitud.relacion_ref)}&unidad_ref=unidad-personal&fecha_referencia=2026-09-24`);
});

test("conserva el resultado incierto para que la vista pueda reintentar el mismo material", async () => {
  const cliente = crearClienteRectificacionDietasHTTP({ fetchImpl: async () => json({ error: "personal.error.no_disponible" }, 503) });
  await assert.rejects(() => cliente.solicitar(solicitud), (error) => error.codigo === "no_disponible" && error.resultadoIndeterminado === true);
});

test("no acepta grupos ni una respuesta sin recibo con el formato real", async () => {
  const cliente = crearClienteRectificacionDietasHTTP({ fetchImpl: async () => json({ ...resultado, recibo_ref: "rrd_invalido" }, 201) });
  await assert.rejects(() => cliente.solicitar({ ...solicitud, campos_a_revisar: ["grupo_dieta"] }), /solicitud de rectificación no válida/u);
  await assert.rejects(() => cliente.solicitar(solicitud), (error) => error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado === true);
});
