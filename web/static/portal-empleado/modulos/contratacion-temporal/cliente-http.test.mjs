import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
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

function comandoAlta() {
  return {
    clave_idempotencia: "4d36e96e-e325-4f9b-bebc-291d91d6f732",
    solicitud: {
      centro_ref: "centro:solicitante:001",
      contacto_ref: "contacto:opaco:001",
      categoria_ref: "categoria:tecnica:001",
      grupo_subgrupo: "A1",
      motivo_clave: "necesidad_temporal",
      detalle: "Cobertura temporal de una necesidad catalogada.",
      periodo: {
        inicio: "2026-08-01T00:00:00Z",
        fin: "2026-12-31T00:00:00Z",
      },
      rc: { existe: false },
      documentos_adjuntos: ["documento:opaco:001"],
      observaciones: "Tramitación ordinaria.",
    },
  };
}

function reciboAlta() {
  return {
    expediente_ref: "expediente:ct:0001",
    numero_visible: "2026/CT-0001",
    version: 1,
    recibo_ref: "recibo:ct:0001",
    confirmada_en: "2026-07-26T09:15:00Z",
  };
}

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
    assert.equal(opciones.credentials, "omit");
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

test("el inventario expone siete rutas y los cinco flujos previos siguen intactos", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      if (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.alta) {
        return respuestaJSON({ data: reciboAlta() }, 201);
      }
      if (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.propuestaCobertura) {
        return respuestaJSON({ data: propuesta() }, 200);
      }
      if (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.resultadoCobertura) {
        return respuestaJSON({
          data: resultadoConsultaCoberturaConfirmado(),
        }, 200);
      }
      return respuestaJSON({ data: reciboCobertura() }, 201);
    },
  });

  await cliente.registrarSolicitud(comandoAlta());
  await cliente.proponerCobertura({
    expediente_ref: "expediente:ct:0001",
    version_esperada: 1,
  });
  await cliente.decidirCobertura(decision());
  await cliente.rectificarCobertura(rectificacion());
  await cliente.consultarResultadoCobertura(
    solicitudConsultaResultadoCobertura(),
  );
  assert.deepEqual(
    llamadas.map(({ ruta }) => ruta),
    Object.values(RUTAS_HTTP_CONTRATACION_TEMPORAL).slice(0, 5),
  );
  assert.deepEqual(Object.values(RUTAS_HTTP_CONTRATACION_TEMPORAL), [
    "/api/vec/contratacion-temporal/solicitudes",
    "/api/vec/contratacion-temporal/cobertura/propuesta",
    "/api/vec/contratacion-temporal/cobertura/decisiones",
    "/api/vec/contratacion-temporal/cobertura/rectificaciones",
    "/api/vec/contratacion-temporal/cobertura/resultados",
    "/api/vec/contratacion-temporal/analisis/registros",
    "/api/vec/contratacion-temporal/analisis/rectificaciones",
  ]);
  for (const { ruta, opciones } of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.credentials, "omit");
    assert.equal(opciones.mode, "same-origin");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.equal(opciones.referrerPolicy, "no-referrer");
    assert.equal(ruta.includes("?"), false);
    assert.deepEqual(
      [...opciones.headers.keys()].sort(),
      ["accept", "content-type"],
    );
    for (const prohibida of [
      "authorization",
      "cookie",
      "idempotency-key",
      "remote-user",
      "x-forwarded-user",
      "x-auth-subject",
      "x-vec-subject",
      "actor",
      "perfil",
      "organizacion",
      "roles",
    ]) {
      assert.equal(opciones.headers.has(prohibida), false, prohibida);
    }
  }
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), comandoAlta());
  assert.deepEqual(JSON.parse(llamadas[2].opciones.body), decision());
  assert.deepEqual(JSON.parse(llamadas[3].opciones.body), rectificacion());
  assert.deepEqual(
    JSON.parse(llamadas[4].opciones.body),
    solicitudConsultaResultadoCobertura(),
  );
});

test("la configuración es cerrada y no admite autoridad del navegador", () => {
  let llamadas = 0;
  assert.throws(
    () => crearClienteHTTPContratacionTemporal(null),
    /configuración .* no válida/u,
  );
  for (const configuracion of [
    [],
    { baseURL: "https://otro.example" },
    { bearer: "secreto" },
    { headers: { Authorization: "fabricada" } },
    { cookies: true },
  ]) {
    assert.throws(
      () => crearClienteHTTPContratacionTemporal(Array.isArray(configuracion)
        ? configuracion
        : {
          fetchImpl: async () => { llamadas += 1; },
          ...configuracion,
        }),
      /configuración .* no válida/u,
    );
  }
  assert.equal(llamadas, 0);
});

test("entrada adicional o identidad adulterada se rechazan antes de red", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuestaJSON({ data: reciboCobertura() }, 201);
    },
  });
  await assert.rejects(
    cliente.decidirCobertura({ ...decision(), actor_ref: "actor:fabricado" }),
    /contrato cerrado/u,
  );
  const adulterada = decision();
  adulterada.identidad_semantica.huella_sha256 = HUELLA_B;
  await assert.rejects(
    cliente.decidirCobertura(adulterada),
    /identidad semántica no válida/u,
  );
  await assert.rejects(cliente.decidirCobertura({
    ...decision(),
    clave_idempotencia: "00000000-0000-4000-8000-000000000000",
  }));
  await assert.rejects(cliente.decidirCobertura({
    ...decision(),
    expediente_ref: `expediente:${"x".repeat(151)}`,
  }));
  const simbolo = decision();
  simbolo[Symbol("campo_oculto")] = "prohibido";
  await assert.rejects(cliente.decidirCobertura(simbolo));
  assert.equal(llamadas, 0);
});

test("DTO y respuestas validadas son copias congeladas", () => {
  const original = decision();
  const validada = validarSolicitudDecisionCobertura(original);
  original.identidad_semantica.canon.algoritmo = "otro";
  assert.equal(validada.identidad_semantica.canon.algoritmo, "sha-256");
  assert.equal(Object.isFrozen(validada.identidad_semantica.canon), true);
  assert.equal(Object.isFrozen(validarPropuestaCobertura(propuesta())), true);
  assert.equal(Object.isFrozen(validarReciboCobertura(reciboCobertura())), true);
  assert.equal(Object.isFrozen(
    validarSolicitudRectificacionCobertura(rectificacion()),
  ), true);
  const clavesRepetidas = propuesta();
  clavesRepetidas.evaluaciones[0].conflictos = ["antiguedad_no_consta"];
  assert.throws(() => validarPropuestaCobertura(clavesRepetidas));
  const listaDispersa = propuesta();
  listaDispersa.evaluaciones[0].resultados_omitidos = Array(1);
  assert.throws(() => validarPropuestaCobertura(listaDispersa));
  assert.throws(() => validarReciboCobertura({
    ...reciboCobertura(),
    confirmada_en: "2026-07-26T09:16:00.1234567Z",
  }));
  assert.throws(() => validarReciboCobertura({
    ...reciboCobertura(),
    confirmada_en: "0000-07-26T09:16:00Z",
  }));
});

test("operacion_pendiente nunca se reintenta y conserva solo correlación", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuestaJSON({
        error: {
          codigo: "operacion_pendiente",
          clave_i18n:
            "api.contratacion_temporal.cobertura.error.operacion_pendiente",
          correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
        },
      }, 503);
    },
  });
  await assert.rejects(
    cliente.decidirCobertura(decision()),
    (error) => {
      assert.equal(
        error instanceof ErrorClienteHTTPContratacionTemporal,
        true,
      );
      assert.equal(error.codigo, "operacion_pendiente");
      assert.equal(error.resultadoIndeterminado, true);
      assert.equal(error.reintentoPermitido, false);
      assert.equal(
        error.correlacionRef,
        "corr_0123456789abcdef0123456789abcdef",
      );
      return true;
    },
  );
  assert.equal(llamadas, 1);
});

test("la pérdida de transporte solo deja indeterminado un efecto enviado", async () => {
  let llamadasEfecto = 0;
  const clienteEfecto = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadasEfecto += 1;
      throw new Error("detalle privado de red");
    },
  });
  await assert.rejects(
    clienteEfecto.decidirCobertura(decision()),
    (error) => {
      assert.equal(error.resultadoIndeterminado, true);
      assert.equal(error.requiereRecuperacion, true);
      assert.equal(error.reintentoPermitido, false);
      assert.doesNotMatch(error.message, /detalle privado/u);
      assert.equal(Object.hasOwn(error, "cause"), false);
      return true;
    },
  );
  assert.equal(llamadasEfecto, 1);

  const clienteConsulta = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      throw new Error("detalle privado de red");
    },
  });
  await assert.rejects(
    clienteConsulta.proponerCobertura({
      expediente_ref: "expediente:ct:0001",
      version_esperada: 1,
    }),
    (error) => error.codigo === "servicio_no_disponible"
      && error.resultadoIndeterminado === false,
  );
});

test("un recibo aplicado debe corresponder a la versión enviada", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuestaJSON({
      data: reciboCobertura({ version_resultante: 3 }),
    }, 201),
  });
  await assert.rejects(
    cliente.decidirCobertura(decision()),
    (error) => error.resultadoIndeterminado === true
      && error.reintentoPermitido === false,
  );
  await assert.rejects(
    cliente.rectificarCobertura(rectificacion()),
    (error) => error.resultadoIndeterminado === true
      && error.reintentoPermitido === false,
  );
});

test("un error remoto privado o incoherente se redacta", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuestaJSON({
      error: {
        codigo: "servicio_no_disponible",
        clave_i18n:
          "api.contratacion_temporal.cobertura.error.servicio_no_disponible",
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
        detalle: "postgres://usuario:clave@servidor/bd",
      },
    }, 503),
  });
  await assert.rejects(
    cliente.proponerCobertura({
      expediente_ref: "expediente:ct:0001",
      version_esperada: 1,
    }),
    (error) => {
      assert.equal(error.codigo, "respuesta_error_no_valida");
      assert.doesNotMatch(error.message, /postgres|usuario|clave/u);
      assert.equal(error.correlacionRef, null);
      return true;
    },
  );
});

test("códigos y prefijos imposibles para la ruta fallan cerrados", async () => {
  const casos = [
    {
      ruta: "propuesta",
      codigo: "operacion_pendiente",
      clave:
        "api.contratacion_temporal.cobertura.error.operacion_pendiente",
    },
    {
      ruta: "propuesta",
      codigo: "operacion_pendiente",
      clave: "api.vec.ruta_exacta.error.operacion_pendiente",
    },
  ];
  for (const caso of casos) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuestaJSON({
        error: {
          codigo: caso.codigo,
          clave_i18n: caso.clave,
          correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
        },
      }, 503),
    });
    await assert.rejects(
      cliente.proponerCobertura({
        expediente_ref: "expediente:ct:0001",
        version_esperada: 1,
      }),
      (error) => error.codigo === "respuesta_error_no_valida",
    );
  }

  const clienteAlta = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuestaJSON({
      error: {
        codigo: "conflicto",
        clave_i18n: "api.contratacion_temporal.alta.error.conflicto",
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
      },
    }, 409),
  });
  await assert.rejects(
    clienteAlta.registrarSolicitud(comandoAlta()),
    (error) => error.codigo === "respuesta_error_no_valida"
      && error.resultadoIndeterminado === true,
  );
});

test("redirección y respuesta excesiva fallan cerradas", async () => {
  const redirigida = respuestaJSON({ data: propuesta() }, 200);
  Object.defineProperty(redirigida, "redirected", { value: true });
  const clienteRedireccion = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => redirigida,
  });
  await assert.rejects(
    clienteRedireccion.proponerCobertura({
      expediente_ref: "expediente:ct:0001",
      version_esperada: 1,
    }),
    (error) => error.codigo === "respuesta_incompatible",
  );

  const clienteExcesivo = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => new Response("{}", {
      status: 200,
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Content-Length": String(300 * 1024),
      },
    }),
  });
  await assert.rejects(
    clienteExcesivo.proponerCobertura({
      expediente_ref: "expediente:ct:0001",
      version_esperada: 1,
    }),
    (error) => error.codigo === "respuesta_excesiva",
  );
});

test("AbortSignal preabortada no toca la red", async () => {
  let llamadas = 0;
  const controlador = new AbortController();
  controlador.abort();
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuestaJSON({ data: reciboAlta() }, 201);
    },
  });
  await assert.rejects(
    cliente.registrarSolicitud(
      comandoAlta(),
      { signal: controlador.signal },
    ),
    (error) => error.name === "AbortError"
      && error.codigo === "operacion_abortada",
  );
  assert.equal(llamadas, 0);
});

test("una implementación de cabeceras que inyecta autoridad falla antes de red", async () => {
  let llamadas = 0;
  class CabecerasHostiles extends Headers {
    constructor() {
      super({ Authorization: "Bearer prohibido" });
    }
  }
  const cliente = crearClienteHTTPContratacionTemporal({
    HeadersImpl: CabecerasHostiles,
    fetchImpl: async () => {
      llamadas += 1;
    },
  });
  await assert.rejects(
    cliente.proponerCobertura({
      expediente_ref: "expediente:ct:0001",
      version_esperada: 1,
    }),
    (error) => error.codigo === "cabeceras_no_disponibles",
  );
  assert.equal(llamadas, 0);
});

test("el código productivo no contiene cookies, storage, demo ni reintentos", async () => {
  const fuente = await readFile(
    new URL("./cliente-http.js", import.meta.url),
    "utf8",
  );
  assert.doesNotMatch(
    fuente,
    /document\.cookie|localStorage|sessionStorage|indexedDB/u,
  );
  assert.doesNotMatch(
    fuente,
    /adaptador-presentacion|datos-presentacion|setTimeout|Retry-After/u,
  );
  assert.match(fuente, /credentials:\s*"omit"/u);
  assert.match(fuente, /mode:\s*"same-origin"/u);
});
