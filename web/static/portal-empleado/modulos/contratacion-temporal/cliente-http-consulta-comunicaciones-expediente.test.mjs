import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal, RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js?v=20261008-alta-circular-v3";
import {
  crearConsultaComunicacionesExpedienteClienteHTTP, RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE,
  validarPaginaComunicacionesExpediente, instanteOrdenComunicacion,
} from "./cliente-http-consulta-comunicaciones-expediente.js";

const expediente_ref = "expediente:ct140:001";
const fila = (n, registrada_en = `2026-09-24T10:${String(n).padStart(2, "0")}:00Z`) => ({
  organizacion_ref: "organizacion:ct140", expediente_ref,
  llamamiento_ref: `llamamiento:ct140:${n}`, comunicacion_ref: `comunicacion:ct140:${n}`,
  version: 2, estado: "registrada_localmente", registrada_en,
  recibo_comunicacion_ref: `recibo:comunicacion:${n}`, antecedente_tipo: "seleccion_confirmada",
  recibo_antecedente_ref: `recibo:seleccion:${n}`,
  estado_respuesta: "sin_respuesta",
});
const cursor = `comunicacion:ct140:10#${"a".repeat(64)}`;

test("GET CT140 fija límite 10 y cursor, sin organización ni clave en URL", async () => {
  const llamadas = [];
  const cliente = crearConsultaComunicacionesExpedienteClienteHTTP({
    validarOpciones: ({ signal }) => ({ signal }),
    ejecutar: async (opciones) => {
      llamadas.push(opciones);
      return opciones.validarRespuesta({ expediente_ref, comunicaciones: [fila(1)] });
    },
  });
  const controlador = new AbortController();
  await cliente.consultarComunicacionesExpediente({ expediente_ref }, { signal: controlador.signal });
  await cliente.consultarComunicacionesExpediente({ expediente_ref, cursor }, { signal: controlador.signal });
  assert.equal(llamadas[0].ruta, `${RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE}?expediente_ref=expediente%3Act140%3A001&limite=10`);
  assert.equal(llamadas[1].ruta, `${llamadas[0].ruta}&cursor=${encodeURIComponent(cursor)}`);
  assert.ok(llamadas.every((opciones) => opciones.metodo === "GET" && opciones.efecto === false
    && opciones.estadoEsperado === 200 && opciones.signal === controlador.signal
    && !Object.hasOwn(opciones, "entrada") && !/clave_idempotencia|organizacion_ref/u.test(opciones.ruta)));
});

test("CT140 valida once campos, orden microsegundo, cursor y referencias del expediente", () => {
  const pagina = { expediente_ref, comunicaciones: [fila(1, "2026-09-24T10:00:00.12345Z"),
    fila(2, "2026-09-24T10:00:00.123450Z")] };
  assert.equal(instanteOrdenComunicacion(pagina.comunicaciones[0].registrada_en),
    instanteOrdenComunicacion(pagina.comunicaciones[1].registrada_en));
  assert.equal(validarPaginaComunicacionesExpediente(pagina, { expediente_ref }).comunicaciones.length, 2);
  assert.equal(validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: [{ ...fila(1), estado_respuesta: "registrada" }] }, { expediente_ref })
    .comunicaciones[0].estado_respuesta, "registrada");
  const sinEstadoRespuesta = fila(1);
  delete sinEstadoRespuesta.estado_respuesta;
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: [sinEstadoRespuesta] }, { expediente_ref }), TypeError);
  for (const cambio of [{ expediente_ref: "expediente:ajeno" }, { version: 3 },
    { estado: "otro" }, { estado_respuesta: "inventado" }, { organizacion_ref: ["organizacion:ct140"] },
    { antecedente_tipo: "inventado" }, { recibo_antecedente_ref: "persona@example.invalid" },
    { actor_ref: "actor:ajeno" }]) {
    assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
      comunicaciones: [{ ...fila(1), ...cambio }] }, { expediente_ref }), TypeError);
  }
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: Array.from({ length: 11 }, (_, i) => fila(i + 1)) }, { expediente_ref }), TypeError);
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: [fila(2), fila(1)] }, { expediente_ref }), TypeError);
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: [fila(1)], siguiente_cursor: cursor }, { expediente_ref }), TypeError);
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: Array(2) }, { expediente_ref }), TypeError);
});

test("cliente común compone GET CT140 y conserva 403/404/503 con prefijo exacto", async () => {
  assert.equal(RUTAS_HTTP_CONTRATACION_TEMPORAL.consultaComunicacionesExpediente,
    RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE);
  const pagina = { expediente_ref, comunicaciones: [fila(1)] };
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    assert.equal(ruta, `${RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE}?expediente_ref=expediente%3Act140%3A001&limite=10`);
    assert.equal(opciones.method, "GET");
    assert.equal(opciones.cache, "no-store");
    assert.equal(Object.hasOwn(opciones, "body"), false);
    return new Response(JSON.stringify({ data: pagina }), { status: 200,
      headers: { "Content-Type": "application/json; charset=utf-8" } });
  } });
  assert.deepEqual(await cliente.consultarComunicacionesExpediente({ expediente_ref }),
    validarPaginaComunicacionesExpediente(pagina, { expediente_ref }));
  for (const [status, codigo] of [[403, "acceso_denegado"], [404, "recurso_no_encontrado"],
    [503, "servicio_no_disponible"]]) {
    const denegado = crearClienteHTTPContratacionTemporal({ fetchImpl: async () =>
      new Response(JSON.stringify({ error: { codigo,
        clave_i18n: `api.contratacion_temporal.comunicacion_llamamiento.error.${codigo}`,
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
      } }), { status, headers: { "Content-Type": "application/json; charset=utf-8" } }) });
    await assert.rejects(denegado.consultarComunicacionesExpediente({ expediente_ref }), (error) => {
      assert.equal(error.estado, status);
      assert.equal(error.codigo, codigo);
      assert.equal(error.envelopeValido, true);
      return true;
    });
  }
});
