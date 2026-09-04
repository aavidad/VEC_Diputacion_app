import assert from "node:assert/strict";
import test from "node:test";

import {
  crearFiscalizacionClienteHTTP,
  RUTA_RESULTADOS_FISCALIZACION,
} from "./cliente-http-fiscalizacion.js";

const solicitud = {
  expediente_ref: "expediente:ct:fiscalizacion:cliente-001",
  version_esperada: 5,
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
  resultado: "favorable",
  observaciones: "",
};

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-fiscalizacion.v1",
    operacion: "registrar_resultado",
    expediente_ref: solicitud.expediente_ref,
    version_resultante: 6,
    resultado: solicitud.resultado,
    fase_resultante: "fiscalizacion",
    estado_resultante: "en_curso",
    recibo_ref: "recibo:fiscalizacion:cliente:001",
    auditoria_ref: "auditoria:fiscalizacion:cliente:001",
    evento_ref: "evento:fiscalizacion:cliente:001",
    actor_ref: "actor:intervencion:cliente:001",
    registrada_en: "2026-09-04T19:00:00Z",
  };
}

test("envía el POST exacto y liga el recibo al expediente, versión y resultado", async () => {
  let operacion;
  const cliente = crearFiscalizacionClienteHTTP({
    ejecutar(configuracion) {
      operacion = configuracion;
      return Promise.resolve(configuracion.validarRespuesta(recibo()));
    },
    validarOpciones: (opciones) => ({ signal: opciones?.signal }),
    serializarAcotado: JSON.stringify,
  });
  const resultado = await cliente.registrarResultadoFiscalizacion(solicitud);
  assert.deepEqual(resultado, recibo());
  assert.equal(RUTA_RESULTADOS_FISCALIZACION,
    "/api/vec/contratacion-temporal/fiscalizaciones/resultados");
  assert.equal(operacion.ruta, RUTA_RESULTADOS_FISCALIZACION);
  assert.equal(operacion.estadoEsperado, 201);
  assert.equal(operacion.efecto, true);
  assert.deepEqual(operacion.entrada, solicitud);
});

test("rechaza recibos de otra versión o resultado", async () => {
  const cliente = crearFiscalizacionClienteHTTP({
    async ejecutar(configuracion) {
      return configuracion.validarRespuesta({
        ...recibo(), version_resultante: 7,
      });
    },
    validarOpciones: () => ({ signal: undefined }),
    serializarAcotado: JSON.stringify,
  });
  await assert.rejects(
    cliente.registrarResultadoFiscalizacion(solicitud),
    TypeError,
  );
});
