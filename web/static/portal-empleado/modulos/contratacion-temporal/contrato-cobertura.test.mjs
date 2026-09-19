import assert from "node:assert/strict";
import test from "node:test";

import {
  ErrorClienteHTTPContratacionTemporal,
  RUTAS_HTTP_CONTRATACION_TEMPORAL,
  crearClienteHTTPContratacionTemporal,
} from "./cliente-http.js";
import {
  liberaBloqueoResultadoCobertura,
  validarPropuestaCobertura,
  validarReciboCobertura,
  validarResultadoConsultaCobertura,
  validarSolicitudConsultaResultadoCobertura,
  validarSolicitudDecisionCobertura,
  validarSolicitudRectificacionCobertura,
} from "./contrato-cobertura.js";

const HUELLA_A = "a".repeat(64);
const HUELLA_B = "b".repeat(64);
const IDENTIDAD = Object.freeze({
  referencia: `propuesta-cobertura-semantica:sha256:${HUELLA_A}`,
  huella_sha256: HUELLA_A,
  canon: Object.freeze({
    dominio:
      "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica",
    version_esquema: 1,
    algoritmo: "sha-256",
  }),
});

function propuesta() {
  return {
    esquema: "vec.contratacion-temporal.propuesta-cobertura.v1",
    estado: "viable",
    via_recomendada: "bolsa_vigente",
    evaluaciones: [{
      via_clave: "bolsa_vigente",
      prioridad: 1,
      estado: "viable",
      resultados_omitidos: [],
      ausencias_bloqueantes: [],
      ausencias_admitidas: ["antiguedad_no_consta"],
      no_habilitantes: [],
      conflictos: [],
    }],
    identidad_semantica: structuredClone(IDENTIDAD),
  };
}

test("la propuesta admite v1 sin motivos alternativos", () => {
  const validada = validarPropuestaCobertura(propuesta());
  assert.equal(Object.hasOwn(validada, "motivos_alternativa"), false);
});

test("la propuesta valida motivos alternativos gobernados", () => {
  const validada = validarPropuestaCobertura({
    ...propuesta(),
    motivos_alternativa: [{
      clave: "motivo_eleccion_procedimiento_rrhh",
      via_clave: "bolsa_vigente",
      etiqueta_i18n: "contratacion_temporal.cobertura.motivo.eleccion_procedimiento_rrhh",
    }],
  });
  assert.equal(Object.isFrozen(validada.motivos_alternativa), true);
  assert.throws(() => validarPropuestaCobertura({
    ...propuesta(),
    motivos_alternativa: [{
      clave: "motivo_eleccion_procedimiento_rrhh", via_clave: "ausente",
      etiqueta_i18n: "contratacion_temporal.cobertura.motivo.eleccion_procedimiento_rrhh",
    }],
  }), /propuesta de cobertura/u);
});

function decision() {
  return {
    expediente_ref: "expediente:ct:0001",
    version_esperada: 1,
    clave_idempotencia: "4d36e96e-e325-4f9b-bebc-291d91d6f732",
    identidad_semantica: structuredClone(IDENTIDAD),
    via_elegida: "bolsa_vigente",
    motivo_clave: "",
  };
}

function rectificacion() {
  return {
    ...decision(),
    motivo_clave: "rectificacion",
    predecesora_ref: `decision-cobertura:sha256:${HUELLA_B}`,
    predecesora_huella: HUELLA_B,
  };
}

function reciboCobertura(cambios = {}) {
  return {
    esquema: "vec.contratacion-temporal.recibo-cobertura.v1",
    recibo_ref: "recibo:ct:cobertura:0001",
    estado: "aplicada",
    decision_cobertura_ref: `decision-cobertura:sha256:${HUELLA_B}`,
    version_resultante: 2,
    confirmada_en: "2026-07-26T09:16:00Z",
    ...cambios,
  };
}

function solicitudConsultaResultadoCobertura(cambios = {}) {
  return {
    expediente_ref: "expediente:ct:0001",
    clave_idempotencia: "4d36e96e-e325-4f9b-bebc-291d91d6f732",
    ...cambios,
  };
}

function resultadoConsultaCoberturaConfirmado(cambios = {}) {
  return {
    esquema: "vec.contratacion-temporal.resultado-consulta-cobertura.v1",
    estado: "confirmado",
    recibo: reciboCobertura(),
    ...cambios,
  };
}

function resultadoConsultaCoberturaNoObservable(cambios = {}) {
  return {
    esquema: "vec.contratacion-temporal.resultado-consulta-cobertura.v1",
    estado: "no_observable",
    ...cambios,
  };
}

function respuestaJSON(datos, estado) {
  const texto = JSON.stringify(datos);
  return new Response(texto, {
    status: estado,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "Content-Length": String(new TextEncoder().encode(texto).byteLength),
    },
  });
}

test("la consulta de resultado tiene entrada y unión cerradas", () => {
  const solicitud = solicitudConsultaResultadoCobertura();
  const validada = validarSolicitudConsultaResultadoCobertura(solicitud);
  solicitud.expediente_ref = "expediente:adulterado";
  assert.equal(validada.expediente_ref, "expediente:ct:0001");
  assert.equal(Object.isFrozen(validada), true);

  const confirmado = validarResultadoConsultaCobertura(
    resultadoConsultaCoberturaConfirmado(),
  );
  const denegado = validarResultadoConsultaCobertura(
    resultadoConsultaCoberturaConfirmado({
      recibo: {
        esquema: "vec.contratacion-temporal.recibo-cobertura.v1",
        recibo_ref: "recibo:ct:cobertura:denegada",
        estado: "denegada",
        confirmada_en: "2026-07-26T09:16:00Z",
      },
    }),
  );
  const noObservable = validarResultadoConsultaCobertura(
    resultadoConsultaCoberturaNoObservable(),
  );
  assert.equal(Object.isFrozen(confirmado), true);
  assert.equal(Object.isFrozen(confirmado.recibo), true);
  assert.equal(liberaBloqueoResultadoCobertura(confirmado), true);
  assert.equal(liberaBloqueoResultadoCobertura(denegado), true);
  assert.equal(liberaBloqueoResultadoCobertura(noObservable), false);

  const entradaConAccesor = solicitudConsultaResultadoCobertura();
  Object.defineProperty(entradaConAccesor, "expediente_ref", {
    enumerable: true, get() { throw new Error("accesor ejecutado"); },
  });
  for (const entrada of [
    { ...solicitudConsultaResultadoCobertura(), actor_ref: "actor:fabricado" },
    { ...solicitudConsultaResultadoCobertura(), version_esperada: 1 },
    { ...solicitudConsultaResultadoCobertura(), [Symbol("oculto")]: true },
    entradaConAccesor,
    solicitudConsultaResultadoCobertura({
      clave_idempotencia: "00000000-0000-4000-8000-000000000000",
    }),
    solicitudConsultaResultadoCobertura({ clave_idempotencia: "no-uuid" }),
  ]) {
    assert.throws(
      () => validarSolicitudConsultaResultadoCobertura(entrada),
      /consulta de resultado de cobertura/u,
    );
  }

  const salidaConAccesor = resultadoConsultaCoberturaConfirmado();
  Object.defineProperty(salidaConAccesor, "estado", {
    enumerable: true, get() { throw new Error("accesor ejecutado"); },
  });
  for (const salida of [
    resultadoConsultaCoberturaConfirmado({ recibo: undefined }),
    resultadoConsultaCoberturaConfirmado({ campo_extra: true }),
    { ...resultadoConsultaCoberturaConfirmado(), [Symbol("oculto")]: true },
    salidaConAccesor,
    resultadoConsultaCoberturaNoObservable({ recibo: reciboCobertura() }),
    resultadoConsultaCoberturaNoObservable({ estado: "ausente" }),
    resultadoConsultaCoberturaNoObservable({ esquema: "otro" }),
    {
      ...resultadoConsultaCoberturaConfirmado(),
      recibo: { ...reciboCobertura(), campo_extra: true },
    },
    resultadoConsultaCoberturaConfirmado({
      recibo: { ...reciboCobertura(), [Symbol("oculto")]: true },
    }),
    ...[
      "2026-07-26T11:16:00+02:00",
      "2026-02-30T09:16:00Z",
      "2026-07-26T24:00:00Z",
      "2026-07-26T09:16:00.1234567Z",
    ].map((confirmada_en) => resultadoConsultaCoberturaConfirmado({
      recibo: reciboCobertura({ confirmada_en }),
    })),
  ]) {
    assert.throws(
      () => validarResultadoConsultaCobertura(salida),
      /resultado de consulta de cobertura|recibo de cobertura/u,
    );
  }
});

test("la ruta de resultado aplica una lista positiva de errores", async () => {
  const aceptados = [
    [400, "peticion_no_valida"], [400, "peticion_no_permitida"],
    [401, "autenticacion_requerida"], [403, "acceso_denegado"],
    [404, "recurso_no_encontrado"], [405, "metodo_no_permitido"],
    [406, "representacion_no_aceptable"], [409, "conflicto"],
    [413, "peticion_demasiado_grande"], [415, "tipo_contenido_no_admitido"],
    [422, "contenido_no_valido"], [503, "servicio_no_disponible"],
  ];
  const rechazados = [
    [408, "peticion_cancelada"], [500, "error_interno"],
    [502, "resultado_no_confiable"], [504, "plazo_agotado"],
    [503, "operacion_pendiente"],
  ];
  for (const [estado, codigo] of [...aceptados, ...rechazados]) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuestaJSON({ error: {
        codigo,
        clave_i18n:
          `api.contratacion_temporal.cobertura.error.${codigo}`,
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
      } }, estado),
    });
    await assert.rejects(
      cliente.consultarResultadoCobertura(
        solicitudConsultaResultadoCobertura(),
      ),
      (error) => aceptados.some(
        ([estadoAceptado, codigoAceptado]) =>
          estadoAceptado === estado && codigoAceptado === codigo,
      )
        ? error.codigo === codigo && error.envelopeValido === true
        : error.codigo === "respuesta_error_no_valida"
          && error.envelopeValido === false,
    );
  }
});

test("la consulta usa la ruta, cuerpo y opciones exactas para 200 y 202", async () => {
  const llamadas = [];
  const respuestas = [
    respuestaJSON({ data: resultadoConsultaCoberturaConfirmado() }, 200),
    respuestaJSON({ data: resultadoConsultaCoberturaNoObservable() }, 202),
  ];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return respuestas.shift();
    },
  });

  const confirmado = await cliente.consultarResultadoCobertura(
    solicitudConsultaResultadoCobertura(),
  );
  const noObservable = await cliente.consultarResultadoCobertura(
    solicitudConsultaResultadoCobertura(),
  );

  assert.equal(liberaBloqueoResultadoCobertura(confirmado), true);
  assert.equal(liberaBloqueoResultadoCobertura(noObservable), false);
  assert.equal(llamadas.length, 2);
  for (const { ruta, opciones } of llamadas) {
    assert.equal(
      ruta,
      "/api/vec/contratacion-temporal/cobertura/resultados",
    );
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.equal(opciones.referrerPolicy, "no-referrer");
    assert.deepEqual(
      Object.keys(JSON.parse(opciones.body)).sort(),
      ["clave_idempotencia", "expediente_ref"],
    );
  }
});

test("la consulta es single-flight y nunca reenvía el efecto", async () => {
  let resolverConsulta;
  let llamadasEfecto = 0;
  let llamadasConsulta = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta) => {
      if (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.decisionCobertura) {
        llamadasEfecto += 1;
        return respuestaJSON({
          error: {
            codigo: "operacion_pendiente",
            clave_i18n:
              "api.contratacion_temporal.cobertura.error.operacion_pendiente",
            correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
          },
        }, 503);
      }
      llamadasConsulta += 1;
      return new Promise((resolve) => {
        resolverConsulta = resolve;
      });
    },
  });

  await assert.rejects(
    cliente.decidirCobertura(decision()),
    (error) => error.requiereRecuperacion === true,
  );
  const primera = cliente.consultarResultadoCobertura(
    solicitudConsultaResultadoCobertura(),
  );
  const segunda = cliente.consultarResultadoCobertura(
    solicitudConsultaResultadoCobertura(),
  );
  assert.equal(primera, segunda);
  assert.equal(llamadasConsulta, 1);
  resolverConsulta(respuestaJSON({
    data: resultadoConsultaCoberturaNoObservable(),
  }, 202));
  const [resultadoPrimero, resultadoSegundo] = await Promise.all([
    primera,
    segunda,
  ]);
  assert.equal(liberaBloqueoResultadoCobertura(resultadoPrimero), false);
  assert.deepEqual(resultadoPrimero, resultadoSegundo);
  assert.equal(llamadasEfecto, 1);
  assert.equal(llamadasConsulta, 1);
});

test("estado HTTP y rama cruzados conservan el bloqueo", async () => {
  for (const [estado, resultado] of [
    [200, resultadoConsultaCoberturaNoObservable()],
    [202, resultadoConsultaCoberturaConfirmado()],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuestaJSON({ data: resultado }, estado),
    });
    await assert.rejects(
      cliente.consultarResultadoCobertura(
        solicitudConsultaResultadoCobertura(),
      ),
      (error) => error.codigo === "respuesta_incompatible"
        && error.resultadoIndeterminado === false,
    );
  }
});

