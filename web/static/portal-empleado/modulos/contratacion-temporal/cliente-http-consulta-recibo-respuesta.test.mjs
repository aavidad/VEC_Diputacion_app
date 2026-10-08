import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal, RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js?v=20261008-alta-circular-v3";
import {
  crearConsultaReciboRespuestaClienteHTTP, RUTA_CONSULTA_RECIBO_RESPUESTA,
  validarConsultaReciboRespuesta, validarReciboRespuestaConsultado,
} from "./cliente-http-consulta-recibo-respuesta.js";

const consulta = { organizacion_ref: "organizacion:sintetica:001",
  expediente_ref: "expediente:ct:sintetico:001", comunicacion_ref: "comunicacion:sintetica:001" };
const recibo = {
  esquema: "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1",
  ...consulta, respuesta: "aceptacion", justificante_ref: "justificante:sintetico:001",
  recibo_ref: "recibo:respuesta:001", auditoria_ref: "auditoria:respuesta:001",
  registrada_en: "2026-09-05T09:00:00.123456Z", estado: "registrada_por_rrhh",
};

test("GET envía solo referencias opacas y valida el recibo autorizado", async () => {
  const controlador = new AbortController();
  const cliente = crearConsultaReciboRespuestaClienteHTTP({
    validarOpciones: ({ signal }) => ({ signal }),
    ejecutar: async (opciones) => {
      assert.equal(opciones.metodo, "GET");
      assert.equal(opciones.ruta, `${RUTA_CONSULTA_RECIBO_RESPUESTA}?organizacion_ref=organizacion%3Asintetica%3A001&expediente_ref=expediente%3Act%3Asintetico%3A001&comunicacion_ref=comunicacion%3Asintetica%3A001`);
      assert.equal(opciones.signal, controlador.signal);
      assert.equal(opciones.estadoEsperado, 200);
      assert.equal(opciones.efecto, false);
      assert.equal(Object.hasOwn(opciones, "entrada"), false);
      return opciones.validarRespuesta(recibo, 200);
    },
  });
  assert.deepEqual(await cliente.consultarReciboRespuesta(consulta, { signal: controlador.signal }), recibo);
});

test("GET rechaza referencias y recibos ajenos, añadidos o de tipos incorrectos", () => {
  for (const entrada of [{ ...consulta, clave_idempotencia: "clave:ajena" },
    { ...consulta, organizacion_ref: [consulta.organizacion_ref] },
    { organizacion_ref: consulta.organizacion_ref, comunicacion_ref: consulta.comunicacion_ref },
    { ...consulta, comunicacion_ref: "persona@example.invalid" }]) {
    assert.throws(() => validarConsultaReciboRespuesta(entrada), TypeError);
  }
  for (const cambio of [{ organizacion_ref: "organizacion:ajena" },
    { expediente_ref: "expediente:ajeno" },
    { comunicacion_ref: "comunicacion:ajena" }, { respuesta: "otra" },
    { justificante_ref: [recibo.justificante_ref] }, { estado: "replay_registrada_por_rrhh" },
    { registrada_en: "2026-02-30T09:00:00Z" }, { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174002" }]) {
    assert.throws(() => validarReciboRespuestaConsultado({ ...recibo, ...cambio }, consulta), TypeError);
  }
});

test("cliente común compone GET de recibo sin cuerpo ni clave y valida 404 uniforme", async () => {
  assert.equal(RUTAS_HTTP_CONTRATACION_TEMPORAL.consultaReciboRespuesta, RUTA_CONSULTA_RECIBO_RESPUESTA);
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    assert.equal(ruta, `${RUTA_CONSULTA_RECIBO_RESPUESTA}?organizacion_ref=organizacion%3Asintetica%3A001&expediente_ref=expediente%3Act%3Asintetico%3A001&comunicacion_ref=comunicacion%3Asintetica%3A001`);
    assert.equal(opciones.method, "GET");
    assert.equal(Object.hasOwn(opciones, "body"), false);
    return new Response(JSON.stringify({ data: recibo }), { status: 200,
      headers: { "Content-Type": "application/json; charset=utf-8" } });
  } });
  assert.deepEqual(await cliente.consultarReciboRespuesta(consulta), recibo);
  const ausente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () =>
    new Response(JSON.stringify({ error: { codigo: "recurso_no_encontrado",
      clave_i18n: "api.contratacion_temporal.respuesta_recibida.error.recurso_no_encontrado",
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }), { status: 404, headers: { "Content-Type": "application/json; charset=utf-8" } }) });
  await assert.rejects(ausente.consultarReciboRespuesta(consulta), (error) =>
    error.estado === 404 && error.envelopeValido === true);
});

test("GET CT139 admite tres referencias máximas codificadas y rechaza entrada mayor", async () => {
  const maxima = `a${":/#".repeat(53)}`;
  assert.equal(maxima.length, 160);
  let ruta = "", llamadas = 0;
  const cliente = crearConsultaReciboRespuestaClienteHTTP({
    validarOpciones: () => ({ signal: undefined }),
    ejecutar: async (opciones) => { ruta = opciones.ruta; llamadas += 1; return null; },
  });
  await cliente.consultarReciboRespuesta({ organizacion_ref: maxima,
    expediente_ref: maxima, comunicacion_ref: maxima });
  const query = ruta.split("?")[1];
  assert.ok(query.length > 400 && query.length <= 2048);
  assert.match(query, /%3A%2F%23/u);
  assert.equal(llamadas, 1);
  assert.throws(() => cliente.consultarReciboRespuesta({
    organizacion_ref: `a${":/#".repeat(250)}`,
    expediente_ref: maxima, comunicacion_ref: maxima,
  }), TypeError);
  assert.equal(llamadas, 1);
});
