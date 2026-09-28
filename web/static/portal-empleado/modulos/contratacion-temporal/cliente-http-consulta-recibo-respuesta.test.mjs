import assert from "node:assert/strict";
import test from "node:test";
import {
  crearConsultaReciboRespuestaClienteHTTP, RUTA_CONSULTA_RECIBO_RESPUESTA,
  validarConsultaReciboRespuesta, validarReciboRespuestaConsultado,
} from "./cliente-http-consulta-recibo-respuesta.js";

const consulta = { organizacion_ref: "organizacion:sintetica:001", comunicacion_ref: "comunicacion:sintetica:001" };
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
      assert.equal(opciones.ruta, `${RUTA_CONSULTA_RECIBO_RESPUESTA}?organizacion_ref=organizacion%3Asintetica%3A001&comunicacion_ref=comunicacion%3Asintetica%3A001`);
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
    { ...consulta, comunicacion_ref: "persona@example.invalid" }]) {
    assert.throws(() => validarConsultaReciboRespuesta(entrada), TypeError);
  }
  for (const cambio of [{ organizacion_ref: "organizacion:ajena" },
    { comunicacion_ref: "comunicacion:ajena" }, { respuesta: "otra" },
    { justificante_ref: [recibo.justificante_ref] }, { estado: "replay_registrada_por_rrhh" },
    { registrada_en: "2026-02-30T09:00:00Z" }, { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174002" }]) {
    assert.throws(() => validarReciboRespuestaConsultado({ ...recibo, ...cambio }, consulta), TypeError);
  }
});
