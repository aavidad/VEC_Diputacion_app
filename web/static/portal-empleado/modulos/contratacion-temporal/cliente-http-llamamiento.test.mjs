import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { createHash, webcrypto } from "node:crypto";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  validarSolicitudSeleccionLlamamiento, validarReciboSeleccionLlamamiento,
  validarSolicitudComunicacionLlamamiento, CAMPOS_COMUNICACION_SIGUIENTE,
  validarSolicitudRespuestaRecibida, validarReciboRespuestaRecibida, CAMPOS_RESPUESTA_RECIBIDA,
  validarSolicitudResolucionLlamamiento, validarReciboResolucionLlamamiento, CAMPOS_RESOLUCION,
  validarSolicitudContinuacionLlamamiento, validarReciboContinuacionLlamamiento,
  CAMPOS_SIGUIENTE, CAMPOS_RECIBO_SIGUIENTE,
  snapshotsFormalizacionDesarrollo, validarSolicitudPropuestaFormalizacion, validarReciboPropuestaFormalizacion,
  CAMPOS_PROPUESTA, CAMPOS_RECIBO_PROPUESTA,
} from "./contrato-llamamiento.js";
import { RUTAS_LLAMAMIENTO, RUTA_PUBLICACIONES_FORMALIZACION, cargarPublicacionesFormalizacionDesarrollo } from "./cliente-http-llamamiento.js";

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
const SIGUIENTE = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174004",
  organizacion_ref: RESOLUCION.organizacion_ref, expediente_ref: RESOLUCION.expediente_ref,
  resolucion_ref: RESOLUCION_CONFIRMADA.resolucion_ref, intencion_ref: INTENCION_SIGUIENTE.referencia,
};
const CONTINUACION = {
  esquema: "vec.contratacion-temporal.continuacion-llamamiento.v1",
  organizacion_ref: SIGUIENTE.organizacion_ref, expediente_ref: SIGUIENTE.expediente_ref,
  resolucion_ref: SIGUIENTE.resolucion_ref, intencion_ref: SIGUIENTE.intencion_ref,
  llamamiento_anterior_ref: RESOLUCION.llamamiento_ref, llamamiento_ref: "llamamiento:siguiente:002",
  version_llamamiento: 1, recibo_bolsa_ref: "recibo:bolsa:siguiente:002",
  recibo_ref: "recibo:ct:siguiente:002", auditoria_ref: "auditoria:siguiente:002",
  confirmada_en: "2026-09-05T09:06:00.123450Z", estado_intencion: "despachada", estado_local: "confirmado",
};
const COMUNICACION_SIGUIENTE = {
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174006",
  organizacion_ref: CONTINUACION.organizacion_ref, expediente_ref: CONTINUACION.expediente_ref,
  llamamiento_ref: CONTINUACION.llamamiento_ref, version_esperada: 1,
  prueba_entrega_ref: CONTINUACION.recibo_ref, tipo_antecedente: "continuacion_confirmada",
};
const respuesta = (datos, status = 201) => new Response(JSON.stringify(datos), {
  status, headers: { "content-type": "application/json; charset=utf-8" },
});
const ASSET_FORMALIZACION = JSON.parse(await readFile(new URL("./formalizacion-desarrollo.json", import.meta.url), "utf8"));
const SNAPSHOTS_PROPUESTA = await snapshotsFormalizacionDesarrollo(ASSET_FORMALIZACION, webcrypto);
const PROPUESTA = validarSolicitudPropuestaFormalizacion({
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174005", expediente_ref: SELECCION.expediente_ref,
  llamamiento_ref: RESOLUCION.llamamiento_ref, resolucion_llamamiento_aceptada_ref: RESOLUCION_CONFIRMADA.resolucion_ref,
  recibo_resolucion_aceptada_ref: RESOLUCION_CONFIRMADA.recibo_local_ref, version_esperada: 6,
  ...SNAPSHOTS_PROPUESTA, anexos: [],
});
const PROPUESTA_CONFIRMADA = { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1",
  estado_local: "confirmado", propuesta_ref: "propuesta:sintetica:001", recibo_local_ref: "recibo:propuesta:001",
  version_resultante: 7, confirmada_en: "2026-09-06T10:00:00.123456Z" };

test("publicaciones: asset real, cuatro huellas UTF-8, certificado limitado al mismo origen sin caché", async () => {
  const snapshots = await cargarPublicacionesFormalizacionDesarrollo({ criptografia: webcrypto,
    fetchImpl: async (ruta, opciones) => {
      assert.equal(ruta, RUTA_PUBLICACIONES_FORMALIZACION);
      assert.equal(opciones.method, "GET"); assert.equal(opciones.credentials, "same-origin");
      assert.equal(opciones.mode, "same-origin"); assert.equal(opciones.cache, "no-store");
      assert.equal(opciones.redirect, "error"); assert.equal(opciones.body, undefined);
      return respuesta(ASSET_FORMALIZACION, 200);
    } });
  assert.deepEqual(snapshots, SNAPSHOTS_PROPUESTA);
  for (const [campo, p] of Object.entries(ASSET_FORMALIZACION.publicaciones)) {
    assert.equal(snapshots[campo].huella_sha256, createHash("sha256").update(p.contenido, "utf8").digest("hex"));
    assert.ok(Object.isFrozen(snapshots[campo]));
  }
  for (const fallo of ["404", "esquema", "referencia", "version", "contenido", "extra", "grande", "crypto", "aborto"]) {
    const asset = structuredClone(ASSET_FORMALIZACION), control = new AbortController();
    if (fallo === "esquema") asset.esquema = "otro";
    if (fallo === "referencia") asset.publicaciones.plantilla.referencia = "plantilla:ajena";
    if (fallo === "version") asset.publicaciones.plan_firma.version = 2;
    if (fallo === "contenido") asset.publicaciones.tipo_formalizacion.contenido = "";
    if (fallo === "extra") asset.publicaciones.actor = "persona:ajena";
    if (fallo === "grande") asset.publicaciones.plantilla.contenido = "a".repeat(17000);
    if (fallo === "aborto") control.abort();
    await assert.rejects(cargarPublicacionesFormalizacionDesarrollo({ signal: control.signal,
      criptografia: fallo === "crypto" ? {} : webcrypto,
      fetchImpl: async () => respuesta(asset, fallo === "404" ? 404 : 200) }), TypeError, fallo);
  }
});

test("propuesta: POST once campos existentes, seis de recibo y versión de expediente 6→7, 201/200", async () => {
  assert.deepEqual(CAMPOS_PROPUESTA, ["clave_idempotencia", "expediente_ref", "llamamiento_ref",
    "resolucion_llamamiento_aceptada_ref", "recibo_resolucion_aceptada_ref", "version_esperada",
    "tipo_formalizacion", "plantilla", "anexos", "politica_firma", "plan_firma"]);
  assert.equal(CAMPOS_RECIBO_PROPUESTA.length, 6);
  for (const status of [201, 200]) {
    const recibo = { ...PROPUESTA_CONFIRMADA, estado_local: status === 201 ? "confirmado" : "replay_confirmado" };
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
      assert.equal(ruta, "/api/vec/contratacion-temporal/formalizacion/propuestas");
      assert.equal(opciones.body, JSON.stringify(PROPUESTA)); assert.equal(opciones.method, "POST");
      return respuesta({ data: recibo }, status);
    } });
    assert.deepEqual(await cliente.prepararPropuestaFormalizacion(PROPUESTA), recibo);
  }
  for (const mutacion of [{ organizacion_ref: "org:ajena" }, { version_esperada: 3 }, { anexos: null },
    { anexos: [{}] }, { plantilla: { ...PROPUESTA.plantilla, huella_sha256: "0".repeat(64) } }])
    assert.throws(() => validarSolicitudPropuestaFormalizacion({ ...PROPUESTA, ...mutacion }), TypeError);
  for (const mutacion of [{ auditoria_ref: "aud:interna" }, { version_resultante: 4 },
    { estado_local: "firmada" }, { propuesta_ref: "" }, { confirmada_en: "2026-09-06T10:00:00.1234567Z" }])
    assert.throws(() => validarReciboPropuestaFormalizacion({ ...PROPUESTA_CONFIRMADA, ...mutacion }, PROPUESTA), TypeError);
});

test("propuesta: errores con namespace propio y ningún éxito falso ni reintento automático", async () => {
  for (const [status, codigo] of [[403, "acceso_denegado"], [409, "resolucion_no_aceptada"],
    [409, "clave_idempotencia_reutilizada"], [409, "version_en_conflicto"], [503, "servicio_no_disponible"]]) {
    let llamadas = 0;
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => {
      llamadas += 1;
      return respuesta({ error: { codigo, clave_i18n: `api.contratacion_temporal.propuesta_formalizacion.error.${codigo}`,
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef" } }, status);
    } });
    await assert.rejects(cliente.prepararPropuestaFormalizacion(PROPUESTA), (e) => e.envelopeValido && e.codigo === codigo);
    assert.equal(llamadas, 1);
  }
  for (const data of [{ ...PROPUESTA_CONFIRMADA, documento_ref: "doc:ajeno" },
    { ...PROPUESTA_CONFIRMADA, estado_local: "replay_confirmado" }]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data }) });
    await assert.rejects(cliente.prepararPropuestaFormalizacion(PROPUESTA), (e) => e.resultadoIndeterminado === true);
  }
});

test("manifiestos publican todos los recursos del llamamiento sin duplicados", async () => {
  const recursos = [
    "cliente-http-llamamiento.js", "contrato-llamamiento.js",
    "formulario-llamamiento.js", "i18n-llamamiento.js", "renderizado-llamamiento.js",
  ];
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const contenido = await readFile(new URL(`../../../../${nombre}`, import.meta.url), "utf8");
    const rutas = contenido.trim().split(/\r?\n/u);
    for (const recurso of recursos) {
      const ruta = `static/portal-empleado/modulos/contratacion-temporal/${recurso}`;
      assert.doesNotMatch(ruta, /presentacion|demo/iu, "No debe activar la exclusión de material de presentación");
      assert.equal(rutas.filter((entrada) => entrada === ruta).length, 1, `${nombre}: ${recurso}`);
      assert.ok((await readFile(new URL(`./${recurso}`, import.meta.url), "utf8")).length > 0);
    }
  }
});

test("la bandeja con error conserva navegación y reintento sin formulario de llamamiento", () => {
  const html = renderizarModuloContratacionTemporal({
    vista: "cuadro", carga: "error", cuadro: null, expediente: null,
    tipo_mensaje: "error", mensaje_clave: "estado_error_carga",
  }, { llamamientoDisponible: true });
  assert.match(html, /ct-exp-navegacion/u);
  assert.match(html, /role="alert"/u);
  assert.match(html, /data-ct-exp-accion="reintentar"/u);
  assert.doesNotMatch(html, /data-ct-exp-llamamiento/u);
});

test("siguiente usa POST cinco campos canónicos y recibo exacto catorce campos 201/200", async () => {
  assert.deepEqual(CAMPOS_SIGUIENTE, ["clave_idempotencia", "organizacion_ref", "expediente_ref", "resolucion_ref", "intencion_ref"]);
  assert.equal(CAMPOS_RECIBO_SIGUIENTE.length, 14);
  for (const [status, estado_local] of [[201, "confirmado"], [200, "replay_confirmado"]]) {
    const eco = { ...CONTINUACION, estado_local };
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
      assert.equal(ruta, "/api/vec/contratacion-temporal/llamamientos/siguientes");
      assert.equal(opciones.body, JSON.stringify(SIGUIENTE));
      assert.equal(opciones.method, "POST"); assert.equal(opciones.cache, "no-store");
      assert.equal(opciones.redirect, "error"); assert.equal(opciones.credentials, "same-origin");
      assert.deepEqual([...opciones.headers.keys()], ["accept", "content-type"]);
      return respuesta({ data: eco }, status);
    } });
    const recibido = await cliente.continuarLlamamiento(Object.fromEntries(Object.entries(SIGUIENTE).reverse()));
    assert.deepEqual(recibido, eco); assert.ok(Object.isFrozen(recibido));
    assert.deepEqual(validarReciboContinuacionLlamamiento(eco, SIGUIENTE, RESOLUCION.llamamiento_ref), eco);
  }
});

test("siguiente rechaza campos extra, referencias inválidas, getters y clave no válida antes de HTTP", () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: () => assert.fail("HTTP") });
  for (const cambio of [{ clave_idempotencia: "otra" },
    { clave_idempotencia: "00000000-0000-4000-8000-000000000000" }, { version_esperada: 3 },
    { llamamiento_anterior_ref: RESOLUCION.llamamiento_ref }, { actor_ref: "actor:inventado" },
    { candidatura_ref: "persona:inventada" }, { respuesta: "renuncia" },
    ...CAMPOS_SIGUIENTE.filter((campo) => campo.endsWith("_ref")).flatMap((campo) =>
      ["", "persona@example.invalid", null, "a".repeat(161)].map((valor) => ({ [campo]: valor })))]) {
    assert.throws(() => cliente.continuarLlamamiento({ ...SIGUIENTE, ...cambio }), TypeError);
  }
  for (const campo of CAMPOS_SIGUIENTE) {
    const incompleta = { ...SIGUIENTE }; delete incompleta[campo];
    assert.throws(() => validarSolicitudContinuacionLlamamiento(incompleta), TypeError);
    const getter = { ...SIGUIENTE };
    Object.defineProperty(getter, campo, { get() { assert.fail("getter"); } });
    assert.throws(() => validarSolicitudContinuacionLlamamiento(getter), TypeError);
  }
});

test("siguiente liga cuatro referencias, anterior distinto de nuevo y fechas UTC sin aceptar recibos inventados", async () => {
  const cambios = [{ esquema: "otro" }, { version_llamamiento: "1" }, { version_llamamiento: 2 },
    { estado_intencion: "pendiente" }, { estado_local: "aceptado" }, { Seleccion: {} }, { posicion: 2 },
    { llamamiento_ref: CONTINUACION.llamamiento_anterior_ref },
    ...["organizacion_ref", "expediente_ref", "resolucion_ref", "intencion_ref"].map((campo) => ({ [campo]: "ref:ajena" })),
    ...CAMPOS_RECIBO_SIGUIENTE.filter((campo) => campo.endsWith("_ref")).map((campo) => ({ [campo]: "persona@example.invalid" })),
    ...["0000-01-01T00:00:00Z", "2026-02-30T00:00:00Z", "2026-09-05T09:06:00+00:00",
      "2026-09-05T09:06:00.1234567Z"].map((confirmada_en) => ({ confirmada_en }))];
  for (const cambio of cambios) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data: { ...CONTINUACION, ...cambio } }) });
    await assert.rejects(cliente.continuarLlamamiento(SIGUIENTE), (e) => e.resultadoIndeterminado === true);
  }
  for (const campo of CAMPOS_RECIBO_SIGUIENTE) {
    const incompleta = { ...CONTINUACION }; delete incompleta[campo];
    assert.throws(() => validarReciboContinuacionLlamamiento(incompleta, SIGUIENTE), TypeError);
    const getter = { ...CONTINUACION };
    Object.defineProperty(getter, campo, { get() { assert.fail("getter de recibo"); } });
    assert.throws(() => validarReciboContinuacionLlamamiento(getter, SIGUIENTE), TypeError);
  }
  assert.throws(() => validarReciboContinuacionLlamamiento(CONTINUACION, SIGUIENTE, "llamamiento:ajeno"), TypeError);
  for (const confirmada_en of ["2026-09-05T09:06:00Z", "2026-09-05T09:06:00.12345Z", "2026-09-05T09:06:00.123451Z"]) {
    assert.equal(validarReciboContinuacionLlamamiento({ ...CONTINUACION, confirmada_en }, SIGUIENTE).confirmada_en, confirmada_en);
  }
});

test("siguiente conserva prefijo común y solo rechazos previos conocidos son determinados", async () => {
  for (const [status, codigo, indeterminado] of [[422, "contenido_no_valido", false], [403, "acceso_denegado", false],
    [409, "clave_idempotencia_reutilizada", true], [502, "resultado_no_confiable", true], [503, "servicio_no_disponible", true]]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.comunicacion_llamamiento.error.${codigo}`,
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }, status) });
    await assert.rejects(cliente.continuarLlamamiento(SIGUIENTE), (e) => {
      assert.equal(e.codigo, codigo); assert.equal(e.envelopeValido, true);
      assert.equal(e.resultadoIndeterminado, indeterminado); return true;
    });
  }
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
  assert.equal(Object.keys(JSON.parse(llamadas[1].opciones.body)).length, 6);
  assert.doesNotMatch(llamadas[1].opciones.body, /tipo_antecedente/u);
  for (const { opciones } of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.deepEqual([...opciones.headers.keys()], ["accept", "content-type"]);
  }
});

test("aviso al sucesor reutiliza ruta y siete campos canónicos; recibo local 201/200 sin plazo", async () => {
  assert.deepEqual(CAMPOS_COMUNICACION_SIGUIENTE, ["clave_idempotencia", "organizacion_ref",
    "expediente_ref", "llamamiento_ref", "version_esperada", "prueba_entrega_ref", "tipo_antecedente"]);
  const local = { esquema: REGISTRO.esquema, estado_local: "registrada_localmente",
    comunicacion_ref: "comunicacion:sucesor:002", recibo_ref: "recibo:aviso:sucesor:002",
    auditoria_ref: "auditoria:sucesor:002", version_resultante: 2,
    registrada_en: "2026-09-05T09:07:00.123456Z", intencion_envio_ref: "intencion:aviso:sucesor:002" };
  for (const status of [201, 200]) {
    const data = { ...local, estado_local: status === 201 ? "registrada_localmente" : "replay_registrada_localmente" };
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
      assert.equal(ruta, RUTAS_LLAMAMIENTO.comunicacionLlamamiento);
      assert.equal(opciones.body, JSON.stringify(COMUNICACION_SIGUIENTE));
      assert.match(opciones.body, /,"tipo_antecedente":"continuacion_confirmada"\}$/u);
      assert.deepEqual([...opciones.headers.keys()], ["accept", "content-type"]);
      return respuesta({ data }, status);
    } });
    const obtenido = await cliente.registrarComunicacionLlamamiento(
      Object.fromEntries(Object.entries(COMUNICACION_SIGUIENTE).reverse()));
    assert.deepEqual(obtenido, data); assert.equal(obtenido.recibo_ref, local.recibo_ref);
    assert.equal(obtenido.registrada_en, local.registrada_en);
  }
  for (const data of [{ ...REGISTRO, version_resultante: 2 },
    { ...local, respuesta_hasta: "2026-09-06T08:00:00Z" }, { ...local, version_resultante: 3 }]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data }) });
    await assert.rejects(cliente.registrarComunicacionLlamamiento(COMUNICACION_SIGUIENTE),
      (error) => error.resultadoIndeterminado === true);
  }
});
test("comunicación admite solo seis campos o antecedente nominal de continuación, sin getters", () => {
  for (const cambio of [{ tipo_antecedente: undefined }, { tipo_antecedente: null },
    { tipo_antecedente: "seleccion_confirmada" }, { tipo_antecedente: "continuacion_confirmada " },
    { tipo_antecedente: true }, { version_esperada: 2 }, { prueba_entrega_ref: "" },
    { actor_ref: "actor:ajeno" }]) {
    assert.throws(() => validarSolicitudComunicacionLlamamiento({ ...COMUNICACION_SIGUIENTE, ...cambio }), TypeError);
  }
  const entrada = { ...COMUNICACION_SIGUIENTE };
  Object.defineProperty(entrada, "tipo_antecedente", { get() { assert.fail("getter"); } });
  assert.throws(() => validarSolicitudComunicacionLlamamiento(entrada), TypeError);
  assert.deepEqual(validarSolicitudComunicacionLlamamiento(COMUNICACION), COMUNICACION);
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
    ["registrarComunicacionLlamamiento", COMUNICACION_SIGUIENTE, 409, "clave_idempotencia_reutilizada", "comunicacion", true],
    ["registrarComunicacionLlamamiento", COMUNICACION_SIGUIENTE, 503, "servicio_no_disponible", "comunicacion", true],
    ["registrarComunicacionLlamamiento", COMUNICACION_SIGUIENTE, 403, "acceso_denegado", "comunicacion", false],
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
