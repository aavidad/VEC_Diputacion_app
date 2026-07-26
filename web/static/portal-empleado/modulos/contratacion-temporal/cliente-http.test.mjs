import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  ErrorClienteHTTPContratacionTemporal,
  RUTAS_HTTP_CONTRATACION_TEMPORAL,
  crearClienteHTTPContratacionTemporal,
} from "./cliente-http.js";
import {
  validarPropuestaCobertura,
  validarReciboCobertura,
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

test("las cuatro operaciones usan rutas y opciones HTTP exactas", async () => {
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

  assert.deepEqual(
    llamadas.map(({ ruta }) => ruta),
    Object.values(RUTAS_HTTP_CONTRATACION_TEMPORAL),
  );
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
