import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  validarSolicitudSeleccionLlamamiento, validarReciboSeleccionLlamamiento,
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
