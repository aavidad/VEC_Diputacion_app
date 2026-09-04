import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  ErrorClienteHTTPContratacionTemporal,
  RUTAS_HTTP_CONTRATACION_TEMPORAL,
  crearClienteHTTPContratacionTemporal,
} from "./cliente-http.js";
import {
  validarReciboAnalisis,
  validarSolicitudRectificacionAnalisis,
  validarSolicitudRegistroAnalisis,
} from "./contrato-analisis.js";

const HUELLA = "9".repeat(64);

function solicitudRegistro(cambios = {}) {
  return {
    expediente_ref: "expediente:analisis:http:001",
    version_esperada: 1,
    clave_idempotencia: "11111111-2222-4333-8444-555555555555",
    artefacto_ref: "artefacto:analisis:http:001",
    analisis: {
      modalidad_clave: "interinidad",
      categoria_ref: "categoria:tecnico:001",
      grupo_subgrupo: "A2",
      causa_clave: "sustitucion",
      periodo: {
        inicio: "2026-09-01T00:00:00Z",
        fin: "2027-02-28T00:00:00Z",
      },
      porcentaje_jornada: 7_500,
      entrada_rc: {
        referencia: "entrada:rc:http:001",
        huella_sha256: HUELLA,
      },
    },
    ...cambios,
  };
}

function solicitudRectificacion(cambios = {}) {
  return {
    ...solicitudRegistro({ version_esperada: 2 }),
    motivo_rectificacion_clave: "ajuste_jornada",
    ...cambios,
  };
}

function recibo(operacion, versionResultante, cambios = {}) {
  return {
    esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
    operacion,
    expediente_ref: "expediente:analisis:http:001",
    version_resultante: versionResultante,
    recibo_ref: "recibo:analisis:http:001",
    confirmada_en: "2026-08-21T10:00:00.123Z",
    ...cambios,
  };
}

function respuestaJSON(contenido, estado = 201) {
  const texto = JSON.stringify(contenido);
  return new Response(texto, {
    status: estado,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "Content-Length": String(new TextEncoder().encode(texto).byteLength),
    },
  });
}

test("el contrato de análisis clona, congela y conserva el DTO exacto", () => {
  const registro = solicitudRegistro();
  const validado = validarSolicitudRegistroAnalisis(registro);
  registro.analisis.periodo.fin = "2099-01-01T00:00:00Z";
  registro.analisis.entrada_rc.referencia = "entrada:adulterada";

  assert.equal(validado.analisis.periodo.fin, "2027-02-28T00:00:00Z");
  assert.equal(
    validado.analisis.entrada_rc.referencia,
    "entrada:rc:http:001",
  );
  assert.equal(Object.isFrozen(validado), true);
  assert.equal(Object.isFrozen(validado.analisis), true);
  assert.equal(Object.isFrozen(validado.analisis.periodo), true);
  assert.equal(Object.isFrozen(validado.analisis.entrada_rc), true);

  const rectificacion = validarSolicitudRectificacionAnalisis(
    solicitudRectificacion(),
  );
  assert.equal(rectificacion.motivo_rectificacion_clave, "ajuste_jornada");
  assert.equal(Object.isFrozen(rectificacion), true);

  const salida = validarReciboAnalisis(recibo("registrar", 2));
  assert.equal(Object.isFrozen(salida), true);
  assert.equal(salida.operacion, "registrar");
});

test("los DTO cerrados rechazan autoridad, extras, accesores y formas inválidas", () => {
  const conAccesor = solicitudRegistro();
  Object.defineProperty(conAccesor, "expediente_ref", {
    enumerable: true,
    get() { throw new Error("accesor ejecutado"); },
  });
  const conSimbolo = solicitudRegistro();
  conSimbolo[Symbol("autoridad")] = "actor:fabricado";
  const analisisExtra = solicitudRegistro();
  analisisExtra.analisis.actor_ref = "actor:fabricado";
  const rcExtra = solicitudRegistro();
  rcExtra.analisis.entrada_rc.organizacion_ref = "organizacion:fabricada";
  const periodoLargo = solicitudRegistro();
  periodoLargo.analisis.periodo.fin = "2126-09-02T00:00:00Z";

  for (const entrada of [
    { ...solicitudRegistro(), actor_ref: "actor:fabricado" },
    { ...solicitudRegistro(), perfil_ref: "perfil:fabricado" },
    { ...solicitudRegistro(), organizacion_ref: "organizacion:fabricada" },
    { ...solicitudRegistro(), autorizacion_ref: "decision:fabricada" },
    { ...solicitudRegistro(), motivo_rectificacion_clave: "impropio" },
    conAccesor,
    conSimbolo,
    analisisExtra,
    rcExtra,
    periodoLargo,
    solicitudRegistro({ version_esperada: Number.MAX_SAFE_INTEGER }),
    solicitudRegistro({
      clave_idempotencia: "00000000-0000-4000-8000-000000000000",
    }),
    solicitudRegistro({ expediente_ref: "no" }),
    solicitudRegistro({ analisis: {
      ...solicitudRegistro().analisis,
      porcentaje_jornada: 0,
    } }),
    solicitudRegistro({ analisis: {
      ...solicitudRegistro().analisis,
      periodo: {
        inicio: "2026-09-01T00:00:01Z",
        fin: "2027-02-28T00:00:00Z",
      },
    } }),
  ]) {
    assert.throws(
      () => validarSolicitudRegistroAnalisis(entrada),
      /análisis|registro/u,
    );
  }

  assert.throws(
    () => validarSolicitudRectificacionAnalisis(solicitudRegistro()),
    /rectificación/u,
  );
  assert.throws(
    () => validarSolicitudRectificacionAnalisis(
      solicitudRectificacion({ motivo_rectificacion_clave: "" }),
    ),
    /rectificación/u,
  );

  for (const salida of [
    { ...recibo("registrar", 2), actor_ref: "actor:fabricado" },
    recibo("otra", 2),
    recibo("registrar", 1),
    recibo("registrar", 2, { confirmada_en: "2026-02-30T10:00:00Z" }),
  ]) {
    assert.throws(() => validarReciboAnalisis(salida), /recibo/u);
  }
});

test("registrar y rectificar usan rutas, cuerpos y opciones HTTP exactas", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.registroAnalisis
        ? respuestaJSON({ data: recibo("registrar", 2) })
        : respuestaJSON({ data: recibo("rectificar", 3) });
    },
  });

  const registro = solicitudRegistro();
  const rectificacion = solicitudRectificacion();
  const recibidoRegistro = await cliente.registrarAnalisis(registro);
  const recibidoRectificacion = await cliente.rectificarAnalisis(
    rectificacion,
  );

  assert.equal(recibidoRegistro.operacion, "registrar");
  assert.equal(recibidoRectificacion.operacion, "rectificar");
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), [
    "/api/vec/contratacion-temporal/analisis/registros",
    "/api/vec/contratacion-temporal/analisis/rectificaciones",
  ]);
  for (const { ruta, opciones } of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.mode, "same-origin");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.equal(opciones.referrerPolicy, "no-referrer");
    assert.equal(ruta.includes("?"), false);
    assert.deepEqual(
      [...opciones.headers.keys()].sort(),
      ["accept", "content-type"],
    );
    assert.equal(
      opciones.headers.get("content-type"),
      "application/json; charset=utf-8",
    );
    for (const prohibida of [
      "authorization",
      "cookie",
      "actor",
      "perfil",
      "organizacion",
      "x-forwarded-user",
    ]) {
      assert.equal(opciones.headers.has(prohibida), false, prohibida);
    }
  }
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), registro);
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), rectificacion);
});

test("la validación local impide tocar la red con un DTO abierto", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuestaJSON({ data: recibo("registrar", 2) });
    },
  });
  await assert.rejects(cliente.registrarAnalisis({
    ...solicitudRegistro(),
    identidad_ref: "identidad:fabricada",
  }));
  await assert.rejects(cliente.rectificarAnalisis(
    solicitudRectificacion({ motivo_rectificacion_clave: "" }),
  ));
  assert.equal(llamadas, 0);
});

test("el recibo queda ligado a operación, expediente y versión", async () => {
  const casos = [
    recibo("rectificar", 2),
    recibo("registrar", 2, { expediente_ref: "expediente:otro:001" }),
    recibo("registrar", 3),
  ];
  for (const salida of casos) {
    let llamadas = 0;
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => {
        llamadas += 1;
        return respuestaJSON({ data: salida });
      },
    });
    await assert.rejects(
      cliente.registrarAnalisis(solicitudRegistro()),
      (error) => error instanceof ErrorClienteHTTPContratacionTemporal
        && error.codigo === "respuesta_incompatible"
        && error.resultadoIndeterminado === true
        && error.requiereRecuperacion === true
        && error.reintentoPermitido === false,
    );
    assert.equal(llamadas, 1);
  }
});

test("transporte perdido y respuesta excesiva quedan indeterminados sin reintento", async () => {
  let llamadasTransporte = 0;
  const clienteTransporte = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadasTransporte += 1;
      throw new Error("detalle privado de red");
    },
  });
  await assert.rejects(
    clienteTransporte.registrarAnalisis(solicitudRegistro()),
    (error) => error.resultadoIndeterminado === true
      && error.requiereRecuperacion === true
      && error.reintentoPermitido === false
      && !/detalle privado/u.test(error.message),
  );
  assert.equal(llamadasTransporte, 1);

  let llamadasExceso = 0;
  const clienteExceso = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadasExceso += 1;
      return new Response("{}", {
        status: 201,
        headers: {
          "Content-Type": "application/json; charset=utf-8",
          "Content-Length": String(16 * 1024 + 1),
        },
      });
    },
  });
  await assert.rejects(
    clienteExceso.registrarAnalisis(solicitudRegistro()),
    (error) => error.codigo === "respuesta_excesiva"
      && error.resultadoIndeterminado === true
      && error.reintentoPermitido === false,
  );
  assert.equal(llamadasExceso, 1);
});

test("los desenlaces HTTP ambiguos quedan indeterminados y no se reintentan", async () => {
  const casos = [
    [408, "peticion_cancelada"],
    [500, "error_interno"],
    [502, "resultado_no_confiable"],
    [503, "servicio_no_disponible"],
    [503, "operacion_pendiente"],
    [504, "plazo_agotado"],
  ];
  const operaciones = [
    ["registrar", "registrarAnalisis", solicitudRegistro],
    ["rectificar", "rectificarAnalisis", solicitudRectificacion],
  ];

  for (const [operacion, metodo, crearSolicitud] of operaciones) {
    for (const [estado, codigo] of casos) {
      let llamadas = 0;
      const cliente = crearClienteHTTPContratacionTemporal({
        fetchImpl: async () => {
          llamadas += 1;
          return respuestaJSON({ error: {
            codigo,
            clave_i18n:
              `api.contratacion_temporal.cobertura.error.${codigo}`,
            correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
          } }, estado);
        },
      });
      await assert.rejects(
        cliente[metodo](crearSolicitud()),
        (error) => error.codigo === codigo
          && error.estado === estado
          && error.envelopeValido === true
          && error.resultadoIndeterminado === true
          && error.requiereRecuperacion === true
          && error.reintentoPermitido === false
          && error.repetible === false
          && error.correlacionRef
            === "corr_0123456789abcdef0123456789abcdef",
        `${operacion}: ${estado}/${codigo}`,
      );
      assert.equal(llamadas, 1, `${operacion}: ${estado}/${codigo}`);
    }
  }
});

test("solo los rechazos previos al efecto conservan resultado determinado", async () => {
  const casos = [
    [400, "peticion_no_valida"],
    [400, "peticion_no_permitida"],
    [401, "autenticacion_requerida"],
    [403, "acceso_denegado"],
    [404, "recurso_no_encontrado"],
    [405, "metodo_no_permitido"],
    [406, "representacion_no_aceptable"],
    [409, "conflicto"],
    [413, "peticion_demasiado_grande"],
    [415, "tipo_contenido_no_admitido"],
    [422, "contenido_no_valido"],
  ];

  for (const [estado, codigo] of casos) {
    let llamadas = 0;
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => {
        llamadas += 1;
        return respuestaJSON({ error: {
          codigo,
          clave_i18n:
            `api.contratacion_temporal.cobertura.error.${codigo}`,
          correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
        } }, estado);
      },
    });
    await assert.rejects(
      cliente.registrarAnalisis(solicitudRegistro()),
      (error) => error.codigo === codigo
        && error.envelopeValido === true
        && error.resultadoIndeterminado === false
        && error.requiereRecuperacion === false
        && error.reintentoPermitido === false,
      `${estado}/${codigo}`,
    );
    assert.equal(llamadas, 1, `${estado}/${codigo}`);
  }
});

test("cancelar tras enviar conserva el resultado indeterminado", async () => {
  let resolverRespuesta;
  let llamadas = 0;
  const controlador = new AbortController();
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return new Promise((resolve) => {
        resolverRespuesta = resolve;
      });
    },
  });
  const operacion = cliente.registrarAnalisis(
    solicitudRegistro(),
    { signal: controlador.signal },
  );
  controlador.abort();
  await assert.rejects(
    operacion,
    (error) => error.name === "AbortError"
      && error.resultadoIndeterminado === true
      && error.reintentoPermitido === false,
  );
  assert.equal(llamadas, 1);
  resolverRespuesta(respuestaJSON({ data: recibo("registrar", 2) }));
});

test("AbortSignal preabortada evita el envío del análisis", async () => {
  let llamadas = 0;
  const controlador = new AbortController();
  controlador.abort();
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuestaJSON({ data: recibo("registrar", 2) });
    },
  });
  await assert.rejects(
    cliente.registrarAnalisis(
      solicitudRegistro(),
      { signal: controlador.signal },
    ),
    (error) => error.name === "AbortError"
      && error.codigo === "operacion_abortada",
  );
  assert.equal(llamadas, 0);
});

test("el contrato y cliente no crean autoridad ni almacenamiento de navegador", async () => {
  const fuentes = await Promise.all([
    "./contrato-analisis.js",
    "./cliente-http.js",
  ].map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  const fuente = fuentes.join("\n");
  assert.doesNotMatch(
    fuente,
    /document\.cookie|localStorage|sessionStorage|indexedDB/u,
  );
  assert.doesNotMatch(
    fuente,
    /setTimeout|Retry-After|adaptador-presentacion|datos-presentacion/u,
  );
  assert.match(fuente, /MAXIMO_SOLICITUD_ANALISIS_BYTES\s*=\s*64\s*\*\s*1024/u);
  assert.match(fuente, /MAXIMO_RESPUESTA_ANALISIS_BYTES\s*=\s*16\s*\*\s*1024/u);
});
