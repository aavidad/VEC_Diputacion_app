import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteRectificacionAdminDietasHTTP } from "./cliente-rectificacion-admin-http.js";

const solicitudRef = `srd_${"a".repeat(32)}`, reciboRef = `rrd_${"b".repeat(32)}`;
const asignacionRef = `ads_${"c".repeat(22)}`;
const solicitud = { solicitud_ref: solicitudRef, estado: "pendiente", persona_ref: `per_${"d".repeat(22)}`,
  empleado_ref: `emp_${"e".repeat(22)}`, relacion_ref: `rel_${"f".repeat(22)}`, unidad_ref: "unidad-servidor",
  asignacion_ref: asignacionRef, version_origen: 2, fecha_referencia: "2026-09-24", campos_a_revisar: ["centro_ref"],
  motivo_revision: "Centro incorrecto", detalle_solicitado: "Revisar centro", registrada_en: "2026-09-24T10:00:00Z",
  asignacion_actual: { centro_ref: "centro-servidor", administrativo_persona_ref: `per_${"1".repeat(22)}`,
    responsable_persona_ref: `per_${"2".repeat(22)}`, grupo_dieta: 2, vigente_desde: "2026-09-01", version: 2, asignacion_ref: asignacionRef } };
const lista = { recibo_ref: reciboRef, decision_ref: "decision", efecto_ref: "efecto", consumo_huella_sha256: "0".repeat(64),
  auditoria_ad3_ref: "auditoria", consultada_en: "2026-09-24T10:00:00Z", cardinalidad: 1, solicitudes: [solicitud] };
const entrada = { decision: "confirmar", persona_ref: solicitud.persona_ref, empleado_ref: solicitud.empleado_ref,
  relacion_ref: solicitud.relacion_ref, unidad_ref: solicitud.unidad_ref, asignacion_ref: asignacionRef, version_esperada: 2,
  fecha_referencia: "2026-09-24", clave_idempotencia: "decision-rectificacion-0001", motivo_revision: "Centro comprobado",
  correccion: { centro_ref: "centro-servidor", administrativo_persona_ref: solicitud.asignacion_actual.administrativo_persona_ref,
    responsable_persona_ref: solicitud.asignacion_actual.responsable_persona_ref, grupo_dieta: 2, vigente_desde: "2026-09-01" } };
const recibo = { solicitud_ref: solicitudRef, recibo_ref: reciboRef, estado: "confirmada", registrada_en: "2026-09-24T11:00:00Z",
  asignacion_ref: asignacionRef, version_origen: 2, decision_ref: "decision", efecto_ref: "efecto",
  consumo_huella_sha256: "0".repeat(64), auditoria_ad3_ref: "auditoria" };
const json = (cuerpo, estado = 200) => new Response(JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });

test("GET competente no aporta ámbito ni solicitud y omite cookies", async () => {
  const llamadas = []; const cliente = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async (...args) => { llamadas.push(args); return json(lista); } });
  const r = await cliente.listar();
  assert.equal(llamadas[0][0], "/api/vec/personal/solicitudes-rectificacion-dietas/competentes");
  assert.equal(llamadas[0][1].method, "GET"); assert.equal(llamadas[0][1].credentials, "omit");
  assert.equal(Object.hasOwn(llamadas[0][1], "body"), false);
  assert.deepEqual(Object.keys(llamadas[0][1].headers), ["Accept"]);
  assert.equal(r.solicitudes[0].asignacion_actual.centro_ref, "centro-servidor");
});

test("GET falla cerrado si cambia cardinalidad o acceso V3; expone cambio de asignación para rechazo", async () => {
  const clienteMala = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async () => json({ ...lista, cardinalidad: 0 }) });
  await assert.rejects(clienteMala.listar(), { codigo: "respuesta_incompatible" });
  const clienteAjena = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async () => json({ ...lista, solicitudes: [{ ...solicitud, asignacion_actual: { ...solicitud.asignacion_actual, version: 3 } }] }) });
  assert.equal((await clienteAjena.listar()).solicitudes[0].asignacion_actual.version, 3);
  const clienteDenegada = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async () => json({ error: "personal.error.acceso_denegado" }, 403) });
  await assert.rejects(clienteDenegada.listar(), { codigo: "acceso_denegado", estado: 403 });
});

test("PUT confirma con contrato exacto y conserva recibo validado", async () => {
  const llamadas = []; const cliente = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async (...args) => { llamadas.push(args); return json(recibo); } });
  const r = await cliente.decidir(solicitudRef, entrada);
  assert.equal(llamadas[0][0], `/api/vec/personal/solicitudes-rectificacion-dietas/${solicitudRef}`);
  assert.equal(llamadas[0][1].method, "PUT"); assert.equal(llamadas[0][1].credentials, "omit");
  assert.deepEqual(JSON.parse(llamadas[0][1].body), entrada);
  assert.equal(r.recibo_ref, reciboRef);
});

test("PUT 503 o respuesta 200 incompatible es incierto; 409 no lo es", async () => {
  const fallo503 = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async () => json({ error: "personal.error.no_disponible" }, 503) });
  await assert.rejects(fallo503.decidir(solicitudRef, entrada), { codigo: "no_disponible", resultadoIndeterminado: true });
  const incompatible = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async () => json({ ...recibo, recibo_ref: "otro" }) });
  await assert.rejects(incompatible.decidir(solicitudRef, entrada), { codigo: "respuesta_incompatible", resultadoIndeterminado: true });
  const conflicto = crearClienteRectificacionAdminDietasHTTP({ fetchImpl: async () => json({ error: "personal.error.conflicto" }, 409) });
  await assert.rejects(conflicto.decidir(solicitudRef, entrada), { codigo: "conflicto", resultadoIndeterminado: false });
});
