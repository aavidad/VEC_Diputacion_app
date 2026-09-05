import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  validarSolicitudSeleccionLlamamiento, validarReciboSeleccionLlamamiento,
  validarSolicitudRespuestaRecibida, validarReciboRespuestaRecibida, CAMPOS_RESPUESTA_RECIBIDA,
  validarSolicitudResolucionLlamamiento, validarReciboResolucionLlamamiento, CAMPOS_RESOLUCION,
} from "./contrato-llamamiento.js";
import { RUTAS_LLAMAMIENTO } from "./cliente-http-llamamiento.js";

const SELECCION = {
  expediente_ref: "expediente:ct:sintetico:001", version_esperada: 6,
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
};
const COMUNICACION = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174001",
  organizacion_ref: "organizacion:sintetica:001", expediente_ref: SELECCION.expediente_ref,
  llamamiento_ref: "llamamiento:sintetico:001", version_esperada: 7,
  prueba_entrega_ref: "prueba:sintetica:001",
};
const RECIBO = {
  esquema: "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1",
  estado: "confirmado", recibo_ref: "recibo:sintetico:001", confirmada_en: "2026-09-05T08:00:00Z",
  organizacion_ref: "organizacion:sintetica:001", llamamiento_ref: "llamamiento:sintetico:001",
  version_llamamiento: 1,
};
const REGISTRO = {
  esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
  estado_local: "confirmado", comunicacion_ref: "comunicacion:sintetica:001",
  recibo_ref: "recibo:comunicacion:001", auditoria_ref: "auditoria:sintetica:001",
  version_resultante: 8, respuesta_hasta: "2026-09-06T08:00:00Z",
};
const RESPUESTA_RECIBIDA = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174002",
  organizacion_ref: COMUNICACION.organizacion_ref, expediente_ref: SELECCION.expediente_ref,
  llamamiento_ref: COMUNICACION.llamamiento_ref, comunicacion_ref: REGISTRO.comunicacion_ref,
  version_comunicacion_esperada: 2, respuesta: "aceptacion", correo_ref: "correo:sintetico:001",
  correo_sha256: "1234567890abcdef".repeat(4), recibida_en: "2026-09-05T08:30:00.000Z",
};
const registroRespuesta = (entrada = RESPUESTA_RECIBIDA) => ({
  ...entrada, esquema: "vec.contratacion-temporal.respuesta-recibida-llamamiento.v1",
  justificante_ref: "justificante:sintetico:001", recibo_ref: "recibo:respuesta:001",
  auditoria_ref: "auditoria:respuesta:001", registrada_en: "2026-09-05T09:00:00.123456Z",
  estado: "registrada_por_rrhh",
});
const RESOLUCION = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174003",
  organizacion_ref: RESPUESTA_RECIBIDA.organizacion_ref, expediente_ref: SELECCION.expediente_ref,
  llamamiento_ref: RESPUESTA_RECIBIDA.llamamiento_ref,
  comunicacion_ref: RESPUESTA_RECIBIDA.comunicacion_ref, version_esperada: 2,
  respuesta: "aceptacion", prueba_respuesta_ref: registroRespuesta().justificante_ref,
  revision_respuesta_rrhh: true, revision_plazo_rrhh: true,
  criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906",
};
const RESOLUCION_CONFIRMADA = {
  esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1",
  respuesta: "aceptacion", estado_plazo: "vigente", estado_local: "confirmado",
  resolucion_ref: "resolucion:sintetica:001", recibo_local_ref: "recibo:resolucion:001",
  auditoria_ref: "auditoria:resolucion:001", version_resultante: 3,
  resuelta_en: "2026-09-05T09:05:00.123450Z",
};
const INTENCION_SIGUIENTE = {
  referencia: "intencion:siguiente:001", estado_local: "pendiente",
  actualizada_en: "2026-09-05T09:05:00.12345Z",
};
const respuesta = (datos, status = 201) => new Response(JSON.stringify(datos), {
  status, headers: { "content-type": "application/json; charset=utf-8" },
});

test("POST canónico selección y comunicación usan el transporte común sin cabeceras de identidad", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return respuesta({ data: ruta === RUTAS_LLAMAMIENTO.seleccionLlamamiento
        ? RECIBO : REGISTRO });
    },
  });
  assert.deepEqual(await cliente.seleccionarLlamamiento(SELECCION), RECIBO);
  assert.deepEqual(await cliente.registrarComunicacionLlamamiento(COMUNICACION), REGISTRO);
  assert.equal(llamadas[0].ruta, RUTAS_LLAMAMIENTO.seleccionLlamamiento);
  assert.equal(llamadas[1].ruta, RUTAS_LLAMAMIENTO.comunicacionLlamamiento);
  assert.equal(llamadas[0].opciones.body, JSON.stringify(SELECCION));
  assert.equal(llamadas[1].opciones.body, JSON.stringify(COMUNICACION));
  for (const { opciones } of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.deepEqual([...opciones.headers.keys()], ["accept", "content-type"]);
  }
});

test("recuperación acepta los mismos recibos con HTTP 200", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta) => respuesta({ data: ruta === RUTAS_LLAMAMIENTO.seleccionLlamamiento
      ? RECIBO : { ...REGISTRO, estado_local: "replay_confirmado" } }, 200),
  });
  assert.deepEqual(await cliente.seleccionarLlamamiento(SELECCION), RECIBO);
  assert.equal((await cliente.registrarComunicacionLlamamiento(COMUNICACION)).estado_local,
    "replay_confirmado");
});

test("registro local acepta intención y fecha pero rechaza plazo de respuesta inventado", async () => {
  const local = { ...REGISTRO, estado_local: "registrada_localmente",
    registrada_en: "2026-09-05T08:00:00Z", intencion_envio_ref: "intencion:sintetica:001" };
  delete local.respuesta_hasta;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuesta({ data: local }),
  });
  assert.equal((await cliente.registrarComunicacionLlamamiento(COMUNICACION)).estado_local,
    "registrada_localmente");
  local.respuesta_hasta = "2026-09-06T08:00:00Z";
  await assert.rejects(cliente.registrarComunicacionLlamamiento(COMUNICACION),
    (error) => error.resultadoIndeterminado === true);
});

test("rechaza candidatos arbitrarios, getters y fechas imposibles sin ejecutar el getter", () => {
  assert.throws(() => validarSolicitudSeleccionLlamamiento({
    ...SELECCION, candidatura_ref: "persona:inventada",
  }), TypeError);
  const entrada = { ...SELECCION };
  Object.defineProperty(entrada, "expediente_ref", { get() { assert.fail("getter"); } });
  assert.throws(() => validarSolicitudSeleccionLlamamiento(entrada), TypeError);
  assert.throws(() => validarReciboSeleccionLlamamiento({
    ...RECIBO, confirmada_en: "2026-02-30T08:00:00Z",
  }), TypeError);
});

test("no confirma un recibo incompatible ni el de otra versión", async () => {
  for (const [metodo, solicitud, recibo] of [
    ["seleccionarLlamamiento", SELECCION, { ...RECIBO, nombre: "dato no permitido" }],
    ["registrarComunicacionLlamamiento", COMUNICACION, { ...REGISTRO, version_resultante: 9 }],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuesta({ data: recibo }),
    });
    await assert.rejects(cliente[metodo](solicitud),
      (error) => error.resultadoIndeterminado === true);
  }
});

test("conserva códigos de conflicto y no trata caídas del servicio como rechazos previos", async () => {
  for (const [metodo, solicitud, status, codigo, prefijo, indeterminado] of [
    ["seleccionarLlamamiento", SELECCION, 409, "conflicto_no_reintentable", "seleccion", true],
    ["registrarComunicacionLlamamiento", COMUNICACION, 409, "version_en_conflicto", "comunicacion", true],
    ["seleccionarLlamamiento", SELECCION, 503, "servicio_no_disponible", "seleccion", true],
    ["seleccionarLlamamiento", SELECCION, 403, "acceso_denegado", "seleccion", false],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuesta({ error: {
        codigo, clave_i18n: `api.contratacion_temporal.${prefijo}_llamamiento.error.${codigo}`,
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
      } }, status),
    });
    await assert.rejects(cliente[metodo](solicitud), (error) => {
      assert.equal(error.codigo, codigo);
      assert.equal(error.envelopeValido, true);
      assert.equal(error.resultadoIndeterminado, indeterminado);
      return true;
    });
  }
});

test("respuesta recibida: POST canónico diez campos, eco normalizado UTC y replay", async () => {
  for (const [status, estado, opcion] of [[201, "registrada_por_rrhh", "aceptacion"],
    [200, "replay_registrada_por_rrhh", "renuncia"]]) {
    const esperada = { ...RESPUESTA_RECIBIDA, respuesta: opcion };
    const eco = { ...registroRespuesta(esperada), recibida_en: "2026-09-05T08:30:00Z", estado };
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
      assert.equal(ruta, "/api/vec/contratacion-temporal/llamamientos/respuestas/registro");
      assert.equal(opciones.body, JSON.stringify(esperada));
      assert.deepEqual(Object.keys(JSON.parse(opciones.body)), CAMPOS_RESPUESTA_RECIBIDA);
      assert.equal(opciones.credentials, "same-origin");
      assert.deepEqual([...opciones.headers.keys()], ["accept", "content-type"]);
      return respuesta({ data: eco }, status);
    } });
    const desordenada = Object.fromEntries(Object.entries(esperada).reverse());
    assert.deepEqual(await cliente.registrarRespuestaRecibida(desordenada), eco);
  }
});

test("respuesta recibida rechaza campos ajenos, huella o fecha inválidas antes de HTTP", () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: () => assert.fail("HTTP") });
  const casos = [
    { actor_ref: "actor:inventado" }, { contenido: "correo" }, { version_resultante: 3 },
    { version_comunicacion_esperada: 3 }, { respuesta: "expiracion_gobernada" },
    { correo_ref: "persona@example.invalid" }, { correo_sha256: "0".repeat(64) },
    { correo_sha256: "A".repeat(64) }, { correo_sha256: "a".repeat(63) },
    { recibida_en: "2026-02-30T08:30:00Z" }, { recibida_en: "2026-09-05T08:30:00+00:00" },
    { recibida_en: "2026-09-05T08:30:00.1234567Z" }, { recibida_en: "0000-01-01T00:00:00Z" },
  ];
  for (const cambio of casos) {
    assert.throws(() => cliente.registrarRespuestaRecibida({ ...RESPUESTA_RECIBIDA, ...cambio }), TypeError);
  }
  const getter = { ...RESPUESTA_RECIBIDA };
  Object.defineProperty(getter, "correo_sha256", { get() { assert.fail("getter"); } });
  assert.throws(() => validarSolicitudRespuestaRecibida(getter), TypeError);
});

test("recibo de respuesta exige todo el eco y conserva diferencias de un microsegundo", () => {
  const s = { ...RESPUESTA_RECIBIDA, recibida_en: "2026-09-05T08:30:00.123450Z" };
  const eco = { ...registroRespuesta(s), recibida_en: "2026-09-05T08:30:00.12345Z" };
  assert.equal(validarReciboRespuestaRecibida(eco, s).recibida_en, eco.recibida_en);
  for (const campo of CAMPOS_RESPUESTA_RECIBIDA) {
    const cambio = campo === "recibida_en" ? "2026-09-05T08:30:00.123451Z" : "otro";
    assert.throws(() => validarReciboRespuestaRecibida({ ...eco, [campo]: cambio }, s), TypeError);
  }
  for (const cambio of [{ version_resultante: 3 }, { actor_ref: "actor:inventado" },
    { estado: "aceptada" }, { registrada_en: "2026-09-05T09:00:00.1234567Z" }]) {
    assert.throws(() => validarReciboRespuestaRecibida({ ...eco, ...cambio }, s), TypeError);
  }
  for (const registrada of ["2026-09-05T08:30:00.12345Z", "2026-09-05T08:30:00.123451Z"]) {
    assert.equal(validarReciboRespuestaRecibida({ ...eco, registrada_en: registrada }, s).registrada_en, registrada);
  }
  assert.throws(() => validarReciboRespuestaRecibida({
    ...eco, registrada_en: "2026-09-05T08:30:00.123449Z",
  }, s), TypeError);
});

test("respuesta recibida conserva errores genéricos y distingue rechazo previo de resultado ambiguo", async () => {
  for (const [status, codigo, indeterminado] of [[403, "acceso_denegado", false],
    [422, "contenido_no_valido", false], [409, "clave_idempotencia_reutilizada", true],
    [409, "version_en_conflicto", true], [503, "servicio_no_disponible", true],
    [502, "resultado_no_confiable", true]]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.respuesta_recibida.error.${codigo}`,
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }, status) });
    await assert.rejects(cliente.registrarRespuestaRecibida(RESPUESTA_RECIBIDA), (error) => {
      assert.equal(error.codigo, codigo);
      assert.equal(error.envelopeValido, true);
      assert.equal(error.resultadoIndeterminado, indeterminado);
      return true;
    });
  }
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () =>
    respuesta({ data: { ...registroRespuesta(), correo_ref: "correo:otro" } }) });
  await assert.rejects(cliente.registrarRespuestaRecibida(RESPUESTA_RECIBIDA),
    (error) => error.resultadoIndeterminado === true);
});

test("resolución envía once campos canónicos y valida 201/200 con intención solo en renuncia", async () => {
  assert.deepEqual(CAMPOS_RESOLUCION, ["clave_idempotencia", "organizacion_ref", "expediente_ref",
    "llamamiento_ref", "comunicacion_ref", "version_esperada", "respuesta", "prueba_respuesta_ref",
    "revision_respuesta_rrhh", "revision_plazo_rrhh", "criterio_validacion_ref"]);
  for (const [opcion, status, estado_local] of ["aceptacion", "renuncia"].flatMap((opcion) =>
    [[opcion, 201, "confirmado"], [opcion, 200, "replay_confirmado"]])) {
    const solicitud = { ...RESOLUCION, respuesta: opcion };
    const eco = { ...RESOLUCION_CONFIRMADA, respuesta: opcion, estado_local,
      ...(opcion === "renuncia" ? { intencion_siguiente: { ...INTENCION_SIGUIENTE } } : {}) };
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
      assert.equal(ruta, "/api/vec/contratacion-temporal/llamamientos/resoluciones");
      assert.equal(opciones.body, JSON.stringify(solicitud));
      assert.deepEqual(Object.keys(JSON.parse(opciones.body)), CAMPOS_RESOLUCION);
      assert.equal(opciones.method, "POST");
      assert.equal(opciones.credentials, "same-origin");
      assert.equal(opciones.cache, "no-store");
      assert.equal(opciones.redirect, "error");
      assert.deepEqual([...opciones.headers.keys()], ["accept", "content-type"]);
      return respuesta({ data: eco }, status);
    } });
    const desordenada = Object.fromEntries(Object.entries(solicitud).reverse());
    const resultado = await cliente.resolverLlamamiento(desordenada);
    assert.deepEqual(resultado, eco);
    assert.ok(Object.isFrozen(resultado));
    if (opcion === "renuncia") assert.ok(Object.isFrozen(resultado.intencion_siguiente));
  }
});

for (const opcion of ["aceptacion", "renuncia"]) test(`solicitud ${opcion} exige v2 y rechaza autoridad añadida antes de HTTP`, () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: () => assert.fail("HTTP") });
  const solicitud = { ...RESOLUCION, respuesta: opcion };
  for (const cambio of [{ version_esperada: 3 }, { version_esperada: "2" },
    { respuesta: "" }, { respuesta: "expiracion_gobernada" }, { respuesta: "aceptada" },
    { estado_plazo: "vigente" }, { actor_ref: "actor:inventado" }, { politica_ref: "politica:inventada" },
    { evaluacion_plazo_ref: "evaluacion:inventada" }, { correo_sha256: "a".repeat(64) },
    { prueba_respuesta_ref: "persona@example.invalid" }, { clave_idempotencia: "otra" },
    { criterio_validacion_ref: "politica:inventada" }, { criterio_validacion_ref: "" },
    ...["revision_respuesta_rrhh", "revision_plazo_rrhh"].flatMap((campo) =>
      [false, "true", "false", 1, null, undefined].map((valor) => ({ [campo]: valor })))]) {
    assert.throws(() => cliente.resolverLlamamiento({ ...solicitud, ...cambio }), TypeError);
  }
  for (const campo of CAMPOS_RESOLUCION) {
    const incompleta = { ...solicitud }; delete incompleta[campo];
    assert.throws(() => validarSolicitudResolucionLlamamiento(incompleta), TypeError);
  }
  for (const campo of ["prueba_respuesta_ref", "revision_respuesta_rrhh", "revision_plazo_rrhh", "criterio_validacion_ref"]) {
    const getter = { ...solicitud };
    Object.defineProperty(getter, campo, { get() { assert.fail("getter"); } });
    assert.throws(() => validarSolicitudResolucionLlamamiento(getter), TypeError);
  }
});

test("recibo resolución exige nueve campos, aceptación, plazo vigente y versión 3, sin atribuciones extra", async () => {
  for (const cambio of [{ esquema: "otro" }, { respuesta: "renuncia" }, { estado_plazo: "vencido" },
    { estado_local: "registrada_por_rrhh" }, { version_resultante: 2 }, { version_resultante: "3" },
    { estado_local: "validacion_registrada" }, { Seleccion: {} },
    { resolucion_ref: "" }, { recibo_local_ref: "persona@example.invalid" }, { auditoria_ref: null },
    { intencion_siguiente: null }, { intencion_siguiente: {} }, { intencion_siguiente: INTENCION_SIGUIENTE },
    { actor_ref: "actor:inventado" },
    { resuelta_en: "2026-02-30T09:05:00Z" }, { resuelta_en: "2026-09-05T09:05:00.1234567Z" },
    { resuelta_en: "2026-09-05T09:05:00+00:00" }, { resuelta_en: "0000-01-01T00:00:00Z" }]) {
    const eco = { ...RESOLUCION_CONFIRMADA, ...cambio };
    assert.throws(() => validarReciboResolucionLlamamiento(eco, RESOLUCION), TypeError);
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data: eco }) });
    await assert.rejects(cliente.resolverLlamamiento(RESOLUCION), (e) => e.resultadoIndeterminado === true);
  }
  for (const campo of Object.keys(RESOLUCION_CONFIRMADA)) {
    const incompleta = { ...RESOLUCION_CONFIRMADA }; delete incompleta[campo];
    assert.throws(() => validarReciboResolucionLlamamiento(incompleta, RESOLUCION), TypeError);
  }
  assert.throws(() => validarReciboResolucionLlamamiento(registroRespuesta(), RESOLUCION), TypeError);
  const getter = { ...RESOLUCION_CONFIRMADA };
  Object.defineProperty(getter, "estado_plazo", { get() { assert.fail("getter"); } });
  assert.throws(() => validarReciboResolucionLlamamiento(getter, RESOLUCION), TypeError);
});

test("renuncia exige intención exacta pendiente, UTC no anterior y copia inmutable sin getters", async () => {
  const solicitud = { ...RESOLUCION, respuesta: "renuncia" };
  const base = { ...RESOLUCION_CONFIRMADA, respuesta: "renuncia" };
  const campos = Object.keys(INTENCION_SIGUIENTE);
  const casos = [undefined, null, {}, [], "pendiente",
    ...campos.map((campo) => Object.fromEntries(Object.entries(INTENCION_SIGUIENTE).filter(([c]) => c !== campo))),
    ...[{ referencia: "" }, { referencia: "persona@example.invalid" },
      { estado_local: "despachada" }, { estado_local: "confirmado" }, { estado_local: null },
      { candidatura_ref: "candidatura:inventada" }, { comando_opaco_ref: "comando:privado" },
      ...["2026-09-05T09:05:00.123449Z", "2026-09-05T09:05:00.1234501Z",
        "2026-09-05T09:05:00+00:00", "2026-02-30T09:05:00Z", "0000-01-01T00:00:00Z"]
        .map((actualizada_en) => ({ actualizada_en }))].map((cambio) => ({ ...INTENCION_SIGUIENTE, ...cambio })),
  ];
  for (const intencion_siguiente of casos) {
    const eco = { ...base, intencion_siguiente };
    assert.throws(() => validarReciboResolucionLlamamiento(eco, solicitud), TypeError);
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data: eco }) });
    await assert.rejects(cliente.resolverLlamamiento(solicitud), (e) => e.resultadoIndeterminado === true);
  }
  assert.throws(() => validarReciboResolucionLlamamiento(base, solicitud), TypeError);
  for (const actualizada_en of [INTENCION_SIGUIENTE.actualizada_en, "2026-09-05T09:05:00.123451Z"]) {
    const intencion_siguiente = { ...INTENCION_SIGUIENTE, actualizada_en };
    const validado = validarReciboResolucionLlamamiento({ ...base, intencion_siguiente }, solicitud);
    assert.ok(Object.isFrozen(validado.intencion_siguiente));
    assert.notEqual(validado.intencion_siguiente, intencion_siguiente);
    intencion_siguiente.estado_local = "despachada";
    assert.equal(validado.intencion_siguiente.estado_local, "pendiente");
    assert.equal(validado.intencion_siguiente.actualizada_en, actualizada_en);
  }
  for (const campo of campos) {
    const intencion_siguiente = { ...INTENCION_SIGUIENTE };
    Object.defineProperty(intencion_siguiente, campo, { get() { assert.fail("getter anidado"); } });
    assert.throws(() => validarReciboResolucionLlamamiento({ ...base, intencion_siguiente }, solicitud), TypeError);
  }
  const getter = { ...base };
  Object.defineProperty(getter, "intencion_siguiente", { enumerable: true, get() { assert.fail("getter raíz"); } });
  assert.throws(() => validarReciboResolucionLlamamiento(getter, solicitud), TypeError);
});

test("409 pendiente es conocido sin efecto solo en resolución y con su prefijo exacto", async () => {
  const codigo = "validacion_respuesta_pendiente";
  const clave = `api.contratacion_temporal.comunicacion_llamamiento.error.${codigo}`;
  for (const [metodo, solicitud, status, clave_i18n, valido] of [
    ["resolverLlamamiento", RESOLUCION, 409, clave, true],
    ["resolverLlamamiento", { ...RESOLUCION, respuesta: "renuncia" }, 409, clave, true],
    ["registrarComunicacionLlamamiento", COMUNICACION, 409, clave, false],
    ["registrarRespuestaRecibida", RESPUESTA_RECIBIDA, 409,
      `api.contratacion_temporal.respuesta_recibida.error.${codigo}`, false],
    ["resolverLlamamiento", RESOLUCION, 403, clave, false],
    ["resolverLlamamiento", RESOLUCION, 409, `api.contratacion_temporal.resolucion.error.${codigo}`, false],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ error: {
      codigo, clave_i18n, correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }, status) });
    await assert.rejects(cliente[metodo](solicitud), (e) => {
      assert.equal(e.envelopeValido, valido);
      assert.equal(e.resultadoIndeterminado, !valido);
      if (valido) { assert.equal(e.codigo, codigo); assert.equal(e.estado, 409); }
      return true;
    });
  }
});
