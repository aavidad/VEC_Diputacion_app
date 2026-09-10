import assert from "node:assert/strict";
import test from "node:test";
import {
  validarSolicitudIncorporacionEjercicio, validarReciboIncorporacionEjercicio,
  validarPreparacionIncorporacionEjercicio,
} from "./contrato-incorporacion-ejercicio.js";
import { crearIncorporacionEjercicioClienteHTTP, RUTA_INCORPORACION_EJERCICIO } from "./cliente-http-incorporacion-ejercicio.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";

const solicitud = { expediente_ref: "expediente:ct:001", solicitud_personal_ref: "solicitud:personal:001",
  version_actual_expediente_observada: 8, motivo_clave: "incorporacion.ejercicio", documentos_refs: ["documento:ct:001"],
  confirma_revision_personal: true, confirma_ejercicio_sintetico: true };
const periodo = { desde: "2026-09-10T08:00:00Z", hasta: "2026-10-10T08:00:00Z" };
const recibo = { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2", expediente_ref: solicitud.expediente_ref,
  solicitud_personal_ref: solicitud.solicitud_personal_ref, relacion_ref: "relacion:personal:001", recibo_ref: "recibo:ct:001",
  actuacion_ref: "actuacion:ct:001", registrada_en: "2026-09-10T08:00:00.123456Z", periodo_incorporacion: periodo,
  version_solicitud_personal: 7, version_actual_expediente: 8, seguimiento_ref: "seguimiento:ct:001",
  version_seguimiento_anterior: 0, version_seguimiento_resultante: 1, auditoria_ref: "auditoria:ct:001", outbox_ref: "outbox:ct:001",
  ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false };
const proyeccion = { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", expediente_ref: solicitud.expediente_ref,
  version_actual_expediente: 8, preparacion: { solicitud_personal_ref: solicitud.solicitud_personal_ref,
    version_solicitud_personal: 7, version_seguimiento_esperada: 0, periodo_incorporacion: periodo,
    motivos: [solicitud.motivo_clave], documentos_refs: solicitud.documentos_refs, disponible: true }, recibo: null };

test("contrato real: tres versiones distintas, dos revisiones y copias inmutables", () => {
  const s = validarSolicitudIncorporacionEjercicio(solicitud);
  const p = validarPreparacionIncorporacionEjercicio(proyeccion, solicitud.expediente_ref);
  assert.deepEqual(s, solicitud);
  assert.deepEqual(p, proyeccion);
  assert.deepEqual(validarReciboIncorporacionEjercicio(recibo, solicitud), recibo);
  assert.notEqual(s.documentos_refs, solicitud.documentos_refs);
  assert.ok(Object.isFrozen(p.preparacion.periodo_incorporacion));
  assert.throws(() => p.preparacion.documentos_refs.push("documento:ajeno"));
  for (const cambio of [{ confirma_revision_personal: false }, { confirma_ejercicio_sintetico: false },
    { motivo_clave: "inventado con espacios" }, { documentos_refs: null }, { documentos_refs: ["doc:001", "doc:001"] },
    { version_actual_expediente_observada: 0 }, { version_actual_expediente_observada: Number.MAX_SAFE_INTEGER + 1 },
    { actor: "rrhh" }, { clave_idempotencia: "no-procede-del-navegador" }]) {
    assert.throws(() => validarSolicitudIncorporacionEjercicio({ ...solicitud, ...cambio }), TypeError);
  }
});

test("recuperación conserva versión y fecha originales aunque el expediente avance", () => {
  const p = validarPreparacionIncorporacionEjercicio({ ...proyeccion, version_actual_expediente: 9,
    preparacion: null, recibo }, solicitud.expediente_ref);
  assert.equal(p.recibo.version_actual_expediente, 8);
  assert.equal(p.recibo.registrada_en, recibo.registrada_en);
  for (const cambio of [{ expediente_ref: "expediente:ajeno" }, { solicitud_personal_ref: "solicitud:ajena" },
    { version_actual_expediente: 9 }, { version_seguimiento_resultante: 2 }, { firma_oficial: true },
    { eficacia_administrativa: true }, { ejercicio_sintetico: false }, { principal_ref: "principal:ajeno" },
    { registrada_en: "2026-02-31T00:00:00Z" }]) {
    assert.throws(() => validarReciboIncorporacionEjercicio({ ...recibo, ...cambio }, solicitud), TypeError);
  }
  assert.throws(() => validarPreparacionIncorporacionEjercicio({ ...proyeccion, recibo }, solicitud.expediente_ref), TypeError);
  assert.throws(() => validarPreparacionIncorporacionEjercicio({ ...proyeccion, preparacion: null }, solicitud.expediente_ref), TypeError);
});

test("fechas: compara microsegundos sin redondear y rechaza períodos imposibles", () => {
  const corto = { desde: "2026-09-10T08:00:00.123456Z", hasta: "2026-09-10T08:00:00.123457Z" };
  assert.deepEqual(validarReciboIncorporacionEjercicio({ ...recibo, periodo_incorporacion: corto }, solicitud).periodo_incorporacion, corto);
  for (const p of [{ ...corto, hasta: corto.desde }, { desde: corto.hasta, hasta: corto.desde },
    { ...periodo, desde: "2026-02-31T08:00:00Z" }, { ...periodo, hasta: "2026-10-10T08:00:00+00:00" }]) {
    assert.throws(() => validarReciboIncorporacionEjercicio({ ...recibo, periodo_incorporacion: p }, solicitud), TypeError);
  }
});

test("GET consulta sin efecto y POST200 usan el transporte compartido, sin claves nuevas", async () => {
  const llamadas = [], signal = new AbortController().signal;
  const cliente = crearIncorporacionEjercicioClienteHTTP({ validarOpciones: (opciones) => opciones ?? {},
    ejecutar: async (peticion) => { llamadas.push(peticion); return peticion.validarRespuesta(peticion.metodo === "GET" ? proyeccion : recibo); } });
  assert.deepEqual(await cliente.prepararIncorporacionEjercicio(solicitud.expediente_ref, { signal }), proyeccion);
  assert.deepEqual(await cliente.confirmarIncorporacionEjercicio(solicitud, { signal }), recibo);
  assert.equal(llamadas[0].ruta, `${RUTA_INCORPORACION_EJERCICIO}?expediente_ref=${encodeURIComponent(solicitud.expediente_ref)}`);
  assert.equal(llamadas[0].entrada, undefined);
  assert.equal(llamadas[0].efecto, false);
  assert.equal(llamadas[1].efecto, true);
  assert.equal(llamadas[1].estadoEsperado, 200);
  assert.equal(llamadas[1].signal, signal);
  assert.deepEqual(llamadas[1].entrada, solicitud);
  assert.equal(llamadas[1].rechazoDeterminado({ envelopeValido: true, estado: 403 }), true);
  assert.equal(llamadas[1].rechazoDeterminado({ envelopeValido: true, estado: 409 }), false);
  assert.equal(llamadas[1].rechazoDeterminado({ envelopeValido: true, estado: 503 }), false);
  assert.throws(() => cliente.prepararIncorporacionEjercicio("ref con espacios"), TypeError);
  assert.throws(() => cliente.confirmarIncorporacionEjercicio({ ...solicitud, actor: "rrhh" }), TypeError);
  assert.equal(llamadas.length, 2);
});

test("cliente compartido: contrato HTTP real de incorporación y errores nominales", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return Response.json({ data: opciones.method === "GET" ? proyeccion : recibo }, { headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  assert.deepEqual(await cliente.prepararIncorporacionEjercicio(solicitud.expediente_ref), proyeccion);
  assert.deepEqual(await cliente.confirmarIncorporacionEjercicio(solicitud), recibo);
  assert.equal(llamadas[0].opciones.body, undefined);
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[1].opciones.method, "POST");
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), solicitud);
  for (const [estado, codigo] of [[403, "acceso_denegado"], [409, "conflicto"], [503, "servicio_no_disponible"]]) {
    const fallido = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => Response.json({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.incorporacion_ejercicio.error.${codigo}`,
      correlacion_ref: "corr_no_disponible",
    } }, { status: estado, headers: { "content-type": "application/json; charset=utf-8" } }) });
    await assert.rejects(fallido.prepararIncorporacionEjercicio(solicitud.expediente_ref), {
      estado, codigo, envelopeValido: true, resultadoIndeterminado: false,
    });
    await assert.rejects(fallido.confirmarIncorporacionEjercicio(solicitud), {
      estado, codigo, envelopeValido: true, resultadoIndeterminado: estado !== 403,
    });
  }
});
