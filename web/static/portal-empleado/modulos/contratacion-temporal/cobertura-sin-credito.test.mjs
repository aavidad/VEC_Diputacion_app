import assert from "node:assert/strict";
import test from "node:test";

import { montarFormularioCobertura } from "./formulario-cobertura.js?v=20261001-ct-a-i18n-v1";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { CONFLICTOS_SIN_CREDITO_COBERTURA } from "./cliente-http-transporte.js";

const EXPEDIENTE = "expediente:ct:prueba:credito:001";

function respuestaError(codigo, estado = 409) {
  return new Response(JSON.stringify({ error: {
    codigo, clave_i18n: `api.contratacion_temporal.cobertura.error.${codigo}`,
    correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
  } }), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "", eventos,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    contains: () => true,
    querySelector: () => ({ focus() {}, scrollIntoView() {} }),
    querySelectorAll: () => [],
    replaceChildren() { this.innerHTML = ""; },
  };
}

// El cliente real lee la respuesta de forma asíncrona: se espera a un turno de temporizador.
const estabilizar = () => new Promise((resolver) => { setTimeout(resolver, 20); });

test("la propuesta y la decisión admiten solo los motivos de crédito conocidos", async () => {
  for (const codigo of CONFLICTOS_SIN_CREDITO_COBERTURA) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuestaError(codigo) });
    await assert.rejects(cliente.proponerCobertura({ expediente_ref: EXPEDIENTE, version_esperada: 2 }),
      (error) => error.codigo === codigo && error.estado === 409 && error.envelopeValido === true);
  }
  const ajeno = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuestaError("sin_credito_inventado"),
  });
  await assert.rejects(ajeno.proponerCobertura({ expediente_ref: EXPEDIENTE, version_esperada: 2 }),
    (error) => error.codigo === "respuesta_error_no_valida");
});

test("sin crédito la pantalla dice el motivo en llano y no ofrece decidir", async () => {
  const esperados = {
    sin_credito_retencion_rechazada: /la retención de crédito está rechazada y sin crédito no se tramita/u,
    sin_credito_analisis_pendiente: /falta el análisis de RRHH con la retención de crédito/u,
    sin_credito_partidas_sin_coste: /consta el estado de las partidas, pero falta el coste aproximado/u,
  };
  for (const [codigo, texto] of Object.entries(esperados)) {
    const raiz = raizFalsa();
    let decisiones = 0;
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuestaError(codigo) });
    const desmontar = montarFormularioCobertura({
      raiz,
      cliente: { ...cliente, async decidirCobertura() { decisiones += 1; } },
      contexto: { expediente_ref: EXPEDIENTE, version_esperada: 2 },
      generarClaveIdempotencia: () => "11111111-1111-4111-8111-111111111111",
      confirmarOperacion: () => true,
    });
    await estabilizar();
    assert.match(raiz.innerHTML, texto);
    assert.doesNotMatch(raiz.innerHTML, /sin_credito|409|data-ct-cobertura-evaluacion/u);
    assert.equal(decisiones, 0);
    desmontar();
  }
});
