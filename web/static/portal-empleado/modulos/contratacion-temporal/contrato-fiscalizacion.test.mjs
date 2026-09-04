import assert from "node:assert/strict";
import test from "node:test";

import {
  validarReciboResultadoFiscalizacion,
  validarSolicitudResultadoFiscalizacion,
} from "./contrato-fiscalizacion.js";

const BASE = {
  expediente_ref: "expediente:ct:fiscalizacion:001",
  version_esperada: 5,
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
};

test("acepta únicamente los tres resultados y exige sus observaciones", () => {
  for (const [resultado, observaciones] of [
    ["favorable", ""],
    ["favorable_con_observaciones", "Condición sintética pendiente."],
    ["desfavorable", "Reparo sintético para subsanación."],
  ]) {
    const entrada = { ...BASE, resultado, observaciones };
    assert.deepEqual(
      validarSolicitudResultadoFiscalizacion(JSON.stringify(entrada)),
      entrada,
    );
  }
  assert.throws(() => validarSolicitudResultadoFiscalizacion(JSON.stringify({
    ...BASE, resultado: "favorable_con_observaciones", observaciones: "",
  })), TypeError);
  assert.throws(() => validarSolicitudResultadoFiscalizacion(JSON.stringify({
    ...BASE, resultado: "favorable", observaciones: "No permitidas.",
  })), TypeError);
  assert.throws(() => validarSolicitudResultadoFiscalizacion(JSON.stringify({
    ...BASE, resultado: "reparo", observaciones: "Reparo",
  })), TypeError);
});

test("rechaza campos abiertos y valida el retorno opcional por parejas", () => {
  assert.throws(() => validarSolicitudResultadoFiscalizacion(JSON.stringify({
    ...BASE, resultado: "favorable", observaciones: "", actor_ref: "actor:inyectado",
  })), TypeError);
  const recibo = {
    esquema: "vec.contratacion-temporal.recibo-fiscalizacion.v1",
    operacion: "registrar_resultado",
    expediente_ref: BASE.expediente_ref,
    version_resultante: 6,
    resultado: "desfavorable",
    fase_resultante: "subsanacion_unidad",
    estado_resultante: "incidencia",
    recibo_ref: "recibo:fiscalizacion:sintetico:001",
    auditoria_ref: "auditoria:fiscalizacion:sintetica:001",
    evento_ref: "evento:fiscalizacion:sintetico:001",
    actor_ref: "actor:intervencion:sintetico:001",
    registrada_en: "2026-09-04T19:00:00Z",
    unidad_retorno_ref: "unidad:retorno:sintetica:001",
    responsable_retorno_ref: "responsable:retorno:sintetico:001",
  };
  assert.deepEqual(
    validarReciboResultadoFiscalizacion(JSON.stringify(recibo)),
    recibo,
  );
  assert.throws(() => validarReciboResultadoFiscalizacion(JSON.stringify({
    ...recibo, fase_resultante: "fiscalizacion",
  })), TypeError);
  delete recibo.responsable_retorno_ref;
  assert.throws(
    () => validarReciboResultadoFiscalizacion(JSON.stringify(recibo)),
    TypeError,
  );
});
