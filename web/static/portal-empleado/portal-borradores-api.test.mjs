import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  ErrorAPIBorradores,
  MATRIZ_FETCH_BORRADORES,
  RUTAS_API_BORRADORES,
  crearClienteBorradores,
} from "./portal-borradores-api.js";
import { ESQUEMAS_BORRADORES } from "./portal-borradores-contrato.js";
import {
  CLAVE_IDEMPOTENCIA_A,
  CLAVE_IDEMPOTENCIA_B,
  HUELLA_A,
  HUELLA_B,
  contenido,
  crearRespuestaControlada,
  detalle,
  etagEstado,
  fila,
  limites,
  lista,
  opciones,
  recibo,
  referenciaEstado,
  respuestaError,
  respuestaJSON,
  solicitudActualizar,
  solicitudCrear,
} from "./portal-borradores-fixtures.test-helper.mjs";

test("cliente GET omite cookies, no usa caché y valida envelope y ETag", async () => {
  const llamadas = [];
  let consultasToken = 0;
  const cliente = crearClienteBorradores({
    obtenerBearer: () => { consultasToken += 1; return "token-en-memoria"; },
    fetchImpl: async (ruta, opcionesFetch) => {
      llamadas.push({ ruta, opcionesFetch });
      if (ruta === RUTAS_API_BORRADORES.opciones) return respuestaJSON(opciones());
      if (ruta.startsWith(`${RUTAS_API_BORRADORES.lista}?`)) return respuestaJSON(lista());
      return respuestaJSON(detalle(), { etag: detalle().etag });
    },
  });
  assert.equal((await cliente.obtenerOpciones()).esquema, ESQUEMAS_BORRADORES.opciones);
  assert.equal((await cliente.listar()).elementos.length, 1);
  assert.equal((await cliente.obtenerDetalle("convocatoria:externa:2026#1", limites())).etag, detalle().etag);
  assert.equal(consultasToken, 3, "la credencial se obtiene de nuevo para cada petición");
  for (const llamada of llamadas) {
    assert.equal(llamada.opcionesFetch.credentials, "omit");
    assert.equal(llamada.opcionesFetch.cache, "no-store");
    assert.equal(llamada.opcionesFetch.redirect, "error");
    assert.equal(llamada.opcionesFetch.referrerPolicy, "no-referrer");
    assert.equal(llamada.opcionesFetch.headers.get("authorization"), "Bearer token-en-memoria");
  }
  assert.match(llamadas[1].ruta, /\?limite=40$/);
  assert.match(llamadas[2].ruta, /convocatoria%3Aexterna%3A2026\/versiones\/1$/);
});

test("el canal interno autenticado funciona sin proveedor Bearer ni Authorization", async () => {
  let opcionesFetch;
  const cliente = crearClienteBorradores({
    fetchImpl: async (_ruta, configuracionFetch) => {
      opcionesFetch = configuracionFetch;
      return respuestaJSON(opciones());
    },
  });
  assert.equal((await cliente.obtenerOpciones()).esquema, ESQUEMAS_BORRADORES.opciones);
  assert.equal(opcionesFetch.headers.has("authorization"), false);
  assert.equal(opcionesFetch.credentials, "omit");
});

test("un proveedor Bearer de tipo inválido falla cerrado antes de cualquier petición", () => {
  let peticiones = 0;
  assert.throws(
    () => crearClienteBorradores({
      obtenerBearer: { token: "no-es-un-proveedor" },
      fetchImpl: async () => { peticiones += 1; },
    }),
    /obtenerBearer debe ser una función o null/,
  );
  assert.equal(peticiones, 0);
});

test("POST envía alta cerrada con Idempotency-Key y devuelve recibo probado", async () => {
  const clave = CLAVE_IDEMPOTENCIA_A;
  let llamada;
  const cliente = crearClienteBorradores({
    fetchImpl: async (ruta, opcionesFetch) => {
      llamada = { ruta, opcionesFetch };
      const guardado = recibo("crear");
      return respuestaJSON(guardado, {
        estado: 201,
        etag: guardado.etag,
        location: "/api/vec/bolsa/convocatorias/borradores/convocatoria%3Aexterna%3A2026/versiones/1",
      });
    },
  });
  const resultado = await cliente.crear(solicitudCrear(), limites(), { claveIdempotencia: clave });
  assert.equal(resultado.accion, "crear");
  assert.equal(llamada.ruta, RUTAS_API_BORRADORES.lista);
  assert.equal(llamada.opcionesFetch.method, "POST");
  assert.equal(llamada.opcionesFetch.headers.get("idempotency-key"), clave);
  assert.equal(llamada.opcionesFetch.headers.get("content-type"), "application/json");
  assert.deepEqual(JSON.parse(llamada.opcionesFetch.body), { data: solicitudCrear() });
  assert.match(llamada.ruta, /\/borradores$/);
});

test("POST rechaza cualquier éxito sin 201, ETag y Location canónica concordante", async () => {
  const clave = CLAVE_IDEMPOTENCIA_A;
  const ejecutarCon = async (configuracion) => {
    const cliente = crearClienteBorradores({
      fetchImpl: async () => respuestaJSON(recibo("crear"), configuracion),
    });
    return cliente.crear(solicitudCrear(), limites(), { claveIdempotencia: clave });
  };
  const location = "/api/vec/bolsa/convocatorias/borradores/convocatoria%3Aexterna%3A2026/versiones/1";
  await assert.rejects(
    ejecutarCon({ estado: 200, etag: recibo("crear").etag, location }),
    /HTTP 200/,
  );
  await assert.rejects(
    ejecutarCon({ estado: 201, etag: recibo("crear").etag }),
    /Location no corresponde/,
  );
  await assert.rejects(
    ejecutarCon({ estado: 201, etag: recibo("crear").etag, location: `${location}/otra` }),
    /Location no corresponde/,
  );
  await assert.rejects(
    ejecutarCon({ estado: 201, location }),
    /ETag obligatorio/,
  );
});

test("PUT distingue 409 idempotencia y 412 CAS sin alterar cambios locales", async () => {
  const solicitud = solicitudActualizar();
  const copia = structuredClone(solicitud);
  const clave = CLAVE_IDEMPOTENCIA_B;
  let llamada;
  const clienteConflicto = crearClienteBorradores({
    fetchImpl: async (ruta, opcionesFetch) => {
      llamada = { ruta, opcionesFetch };
      return respuestaError(409, "clave_idempotencia_reutilizada", "correlacion:borradores:409");
    },
  });
  await assert.rejects(
    clienteConflicto.actualizar("convocatoria:externa:2026#1", solicitud, limites(), {
      etag: etagEstado(3), claveIdempotencia: clave,
    }),
    (error) => error instanceof ErrorAPIBorradores
      && error.estado === 409
      && error.tipoConflicto === "idempotencia"
      && error.esConflictoIdempotencia
      && !error.esConflictoCAS
      && error.conservarCambiosLocales
      && error.codigo === "clave_idempotencia_reutilizada"
      && error.correlacion === "correlacion:borradores:409"
      && error.envelopeValido,
  );
  assert.deepEqual(solicitud, copia, "el cliente no consume ni modifica los cambios locales");
  assert.equal(llamada.opcionesFetch.method, "PUT");
  assert.equal(llamada.opcionesFetch.headers.get("if-match"), etagEstado(3));
  assert.equal(llamada.opcionesFetch.headers.get("idempotency-key"), clave);
  assert.match(llamada.ruta, /convocatoria%3Aexterna%3A2026\/versiones\/1$/);

  const clienteCAS = crearClienteBorradores({
    fetchImpl: async () => respuestaError(412, "revision_no_vigente", "correlacion:borradores:412"),
  });
  await assert.rejects(
    clienteCAS.actualizar("convocatoria:externa:2026#1", solicitud, limites(), {
      etag: etagEstado(3), claveIdempotencia: clave,
    }),
    (error) => error instanceof ErrorAPIBorradores
      && error.estado === 412
      && error.tipoConflicto === "cas"
      && error.esConflictoCAS
      && !error.esConflictoIdempotencia
      && error.conservarCambiosLocales
      && error.codigo === "revision_no_vigente"
      && error.correlacion === "correlacion:borradores:412",
  );
  assert.deepEqual(solicitud, copia, "el conflicto CAS tampoco consume los cambios locales");

  const clienteOK = crearClienteBorradores({
    fetchImpl: async () => {
      const guardado = recibo("actualizar");
      return respuestaJSON(guardado, { etag: guardado.etag });
    },
  });
  assert.equal((await clienteOK.actualizar("convocatoria:externa:2026#1", solicitud, limites(), {
    etag: etagEstado(3), claveIdempotencia: clave,
  })).accion, "actualizar");
});

test("cliente falla cerrado ante JSON, ETag, bearer o selector no canónicos", async () => {
  const sinETag = crearClienteBorradores({ fetchImpl: async () => respuestaJSON(detalle()) });
  await assert.rejects(sinETag.obtenerDetalle("convocatoria:externa:2026#1", limites()), /ETag obligatorio/);
  const etagDistinto = crearClienteBorradores({
    fetchImpl: async () => respuestaJSON(detalle(), { etag: '"otro-etag"' }),
  });
  await assert.rejects(etagDistinto.obtenerDetalle("convocatoria:externa:2026#1", limites()), /no es canónica/);
  const noJSON = crearClienteBorradores({
    fetchImpl: async () => new Response("texto", { status: 200, headers: { "content-type": "text/plain" } }),
  });
  await assert.rejects(noJSON.obtenerOpciones(), /no respondió con JSON/);
  const bearerInyectado = crearClienteBorradores({
    obtenerBearer: () => "token\r\nX-Injected: yes",
    fetchImpl: async () => { throw new Error("no debe ejecutarse"); },
  });
  await assert.rejects(bearerInyectado.obtenerOpciones(), /Credencial en memoria no válida/);
  const bearerNoASCII = crearClienteBorradores({
    obtenerBearer: () => "token-🔐",
    fetchImpl: async () => { throw new Error("no debe ejecutarse"); },
  });
  await assert.rejects(bearerNoASCII.obtenerOpciones(), /Credencial en memoria no válida/);
  const cliente = crearClienteBorradores({ fetchImpl: async () => respuestaJSON(lista()) });
  await assert.rejects(cliente.listar({ limite: 51 }), /Selector de listado no válido/);

  const listaCruzada = lista();
  listaCruzada.selector = { limite: 40, texto: "otra consulta" };
  const clienteListaCruzada = crearClienteBorradores({
    fetchImpl: async () => respuestaJSON(listaCruzada),
  });
  await assert.rejects(clienteListaCruzada.listar(), /no respeta el contrato/);

  const detalleCruzado = detalle();
  detalleCruzado.referencia_estado.referencia = "convocatoria:distinta:2026#1";
  const clienteDetalleCruzado = crearClienteBorradores({
    fetchImpl: async () => respuestaJSON(detalleCruzado, { etag: detalleCruzado.etag }),
  });
  await assert.rejects(
    clienteDetalleCruzado.obtenerDetalle("convocatoria:externa:2026#1", limites()),
    /no respeta el contrato/,
  );

  for (const ambigua of [
    "convocatoria/externa#1", "convocatoria%2Fexterna#1", "convocatoria#externa#1",
    "convocatoria:externa#01", "convocatoria:externa#0",
  ]) {
    await assert.rejects(
      clienteDetalleCruzado.obtenerDetalle(ambigua, limites()),
      /Referencia de borrador no válida/,
    );
  }

  const cabecerasRotas = crearClienteBorradores({
    HeadersImpl: class { constructor() { throw new TypeError("fallo controlado"); } },
    fetchImpl: async () => { throw new Error("no debe ejecutarse"); },
  });
  await assert.rejects(
    cabecerasRotas.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores && /cabeceras seguras/.test(error.message),
  );

  const envelopeRaw = crearClienteBorradores({
    fetchImpl: async () => new Response(JSON.stringify(opciones()), {
      status: 200, headers: { "content-type": "application/json" },
    }),
  });
  await assert.rejects(
    envelopeRaw.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores && /contrato de borradores/.test(error.message),
  );
});

test("selector local estricto impide cualquier fetch con entrada no canónica", async () => {
  let llamadas = 0;
  const cliente = crearClienteBorradores({
    fetchImpl: async () => { llamadas += 1; return respuestaJSON(lista()); },
  });
  const invalidos = [
    { cursor: " cursor:1" },
    { cursor: "cursor/ambiguo" },
    { categoria: "Auxiliar_Administrativo" },
    { categoria: "auxiliar administrativo" },
    { texto: "cafe\u0301" },
    { texto: "texto\u200Boculto" },
    { texto: `texto\uD800` },
  ];
  for (const selector of invalidos) {
    await assert.rejects(cliente.listar(selector), /Selector de listado no válido/);
  }
  assert.equal(llamadas, 0);
});

test("las mutaciones requieren una clave idempotente explícita antes de hacer fetch", async () => {
  let llamadas = 0;
  const cliente = crearClienteBorradores({
    fetchImpl: async () => { llamadas += 1; throw new Error("no debe ejecutarse"); },
  });
  await assert.rejects(
    cliente.crear(solicitudCrear(), limites()),
    /clave de idempotencia explícita/,
  );
  await assert.rejects(
    cliente.actualizar("convocatoria:externa:2026#1", solicitudActualizar(), limites(), {
      etag: etagEstado(3),
    }),
    /clave de idempotencia explícita/,
  );
  const contratoRoto = solicitudCrear();
  contratoRoto.actor = "actor:declarado:cliente";
  await assert.rejects(
    cliente.crear(contratoRoto, limites(), { claveIdempotencia: CLAVE_IDEMPOTENCIA_A }),
    (error) => error instanceof ErrorAPIBorradores && /solicitud de alta/.test(error.message),
  );
  assert.equal(llamadas, 0);
});

test("matriz Fetch moderna falla cerrada sin body.getReader y nunca materializa", async () => {
  assert.equal(MATRIZ_FETCH_BORRADORES.fallbackMaterializador, false);
  assert.ok(MATRIZ_FETCH_BORRADORES.requiere.includes("Response.body.getReader"));
  let materializado = false;
  let cancelaciones = 0;
  const cliente = crearClienteBorradores({
    fetchImpl: async () => ({
      status: 200,
      headers: new Headers({
        "content-type": "application/json",
        "content-length": "128",
      }),
      body: { async cancel() { cancelaciones += 1; } },
      text: async () => { materializado = true; return JSON.stringify({ data: opciones() }); },
    }),
  });
  await assert.rejects(cliente.obtenerOpciones(), /lectura incremental acotada/);
  assert.equal(materializado, false);
  assert.equal(cancelaciones, 1);
});

test("streaming admite 4 MiB exactos y rechaza el byte siguiente", async () => {
  const maximo = 4 * 1024 * 1024;
  const exacta = crearRespuestaControlada({
    fragmentos: [new Uint8Array(maximo).fill(0x20)],
    cabeceras: { "content-length": String(maximo) },
  });
  const clienteExacto = crearClienteBorradores({ fetchImpl: async () => exacta.respuesta });
  await assert.rejects(
    clienteExacto.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores
      && error.estado === 200
      && /JSON no válido/.test(error.message),
    "el límite exacto llega al parser; no se clasifica como exceso",
  );
  assert.equal(exacta.traza.cancelacionesCuerpo, 1);

  const excesiva = crearRespuestaControlada({
    fragmentos: [new Uint8Array(maximo), new Uint8Array(1)],
  });
  const clienteExceso = crearClienteBorradores({ fetchImpl: async () => excesiva.respuesta });
  await assert.rejects(
    clienteExceso.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores && error.estado === 413,
  );
  assert.ok(excesiva.traza.cancelacionesLector >= 1);
  assert.equal(excesiva.traza.liberaciones, 1);
});

test("streaming admite 65.536 fragmentos exactos y cancela el 65.537", async () => {
  const payload = Buffer.from(JSON.stringify({ data: opciones() }), "utf8");
  const construir = (cantidad) => {
    let indice = 0;
    const traza = { cancelacionesLector: 0, cancelacionesCuerpo: 0, liberaciones: 0 };
    const lector = {
      async read() {
        if (indice >= cantidad) return { done: true, value: undefined };
        const value = indice === 0 ? payload : new Uint8Array(0);
        indice += 1;
        return { done: false, value };
      },
      async cancel() { traza.cancelacionesLector += 1; },
      releaseLock() { traza.liberaciones += 1; },
    };
    const respuesta = {
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      body: {
        getReader: () => lector,
        async cancel() { traza.cancelacionesCuerpo += 1; },
      },
    };
    return { respuesta, traza };
  };

  const exacta = construir(65_536);
  const clienteExacto = crearClienteBorradores({ fetchImpl: async () => exacta.respuesta });
  assert.equal((await clienteExacto.obtenerOpciones()).esquema, ESQUEMAS_BORRADORES.opciones);
  assert.equal(exacta.traza.cancelacionesLector, 0);
  assert.equal(exacta.traza.cancelacionesCuerpo, 0);
  assert.equal(exacta.traza.liberaciones, 1);

  const excesiva = construir(65_537);
  const clienteExceso = crearClienteBorradores({ fetchImpl: async () => excesiva.respuesta });
  await assert.rejects(
    clienteExceso.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores && error.estado === 413,
  );
  assert.ok(excesiva.traza.cancelacionesLector >= 1);
  assert.equal(excesiva.traza.liberaciones, 1);
});

test("UTF-8 inválido cancela lector y conserva la causa fatal", async () => {
  const controlada = crearRespuestaControlada({
    fragmentos: [Uint8Array.of(0xC3, 0x28)],
  });
  const cliente = crearClienteBorradores({ fetchImpl: async () => controlada.respuesta });
  await assert.rejects(
    cliente.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores
      && /UTF-8 válido/.test(error.message)
      && error.cause instanceof TypeError,
  );
  assert.ok(controlada.traza.cancelacionesLector >= 1);
  assert.equal(controlada.traza.liberaciones, 1);
});

test("toda salida remota rechazada cancela cuerpo o lector sin sustituir la causa", async () => {
  const json = (valor) => Buffer.from(JSON.stringify(valor), "utf8");
  const casos = [
    {
      nombre: "estado inesperado",
      controlada: crearRespuestaControlada({ estado: 201, fragmentos: [json({ data: opciones() })] }),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "cuerpo",
    },
    {
      nombre: "tipo rechazado",
      controlada: crearRespuestaControlada({ tipo: "text/plain", fragmentos: [json({ data: opciones() })] }),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "cuerpo",
    },
    {
      nombre: "Content-Length no canónico",
      controlada: crearRespuestaControlada({
        fragmentos: [json({ data: opciones() })], cabeceras: { "content-length": "01" },
      }),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "cuerpo",
    },
    {
      nombre: "fragmento no byte",
      controlada: crearRespuestaControlada({ fragmentos: ["no-son-bytes"] }),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "lector",
    },
    {
      nombre: "flujo vacío",
      controlada: crearRespuestaControlada(),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "lector",
    },
    {
      nombre: "contrato de datos inválido",
      controlada: crearRespuestaControlada({ fragmentos: [json({ data: {} })] }),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "cuerpo",
    },
    {
      nombre: "ETag ausente",
      controlada: crearRespuestaControlada({ fragmentos: [json({ data: detalle() })] }),
      ejecutar: (cliente) => cliente.obtenerDetalle("convocatoria:externa:2026#1", limites()),
      cancelacion: "cuerpo",
    },
    {
      nombre: "Location ausente",
      controlada: crearRespuestaControlada({
        estado: 201,
        fragmentos: [json({ data: recibo("crear") })],
        cabeceras: { etag: recibo("crear").etag },
      }),
      ejecutar: (cliente) => cliente.crear(solicitudCrear(), limites(), {
        claveIdempotencia: CLAVE_IDEMPOTENCIA_A,
      }),
      cancelacion: "cuerpo",
    },
    {
      nombre: "error remoto válido",
      controlada: crearRespuestaControlada({
        estado: 503,
        fragmentos: [json({
          error: { codigo: "servicio_no_disponible", correlacion_ref: "correlacion:borradores:503" },
        })],
      }),
      ejecutar: (cliente) => cliente.obtenerOpciones(),
      cancelacion: "cuerpo",
    },
  ];

  for (const caso of casos) {
    const cliente = crearClienteBorradores({ fetchImpl: async () => caso.controlada.respuesta });
    await assert.rejects(caso.ejecutar(cliente), ErrorAPIBorradores, caso.nombre);
    const contador = caso.cancelacion === "lector"
      ? caso.controlada.traza.cancelacionesLector
      : caso.controlada.traza.cancelacionesCuerpo;
    assert.ok(contador >= 1, `${caso.nombre}: se cancela ${caso.cancelacion}`);
  }

  const causaLectura = new Error("causa original de lectura");
  const lecturaRota = crearRespuestaControlada({
    falloLectura: causaLectura,
    falloCancelacionLector: new Error("fallo secundario al cancelar lector"),
    falloCancelacionCuerpo: new Error("fallo secundario al cancelar cuerpo"),
  });
  const clienteRoto = crearClienteBorradores({ fetchImpl: async () => lecturaRota.respuesta });
  await assert.rejects(
    clienteRoto.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores && error.cause === causaLectura,
  );
  assert.ok(lecturaRota.traza.cancelacionesLector >= 1);
  assert.ok(lecturaRota.traza.cancelacionesCuerpo >= 1);
});

test("AbortSignal gobierna credencial, fetch y lectura streaming", async () => {
  const comprobarAborto = (causa) => (error) => error instanceof ErrorAPIBorradores
    && error.codigo === "operacion_abortada"
    && error.cause === causa;

  const causaCredencial = new Error("aborto durante credencial");
  const controladorCredencial = new AbortController();
  let signalCredencial;
  let fetchTrasCredencial = false;
  const clienteCredencial = crearClienteBorradores({
    obtenerBearer: (signal) => {
      signalCredencial = signal;
      return new Promise(() => {});
    },
    fetchImpl: async () => { fetchTrasCredencial = true; throw new Error("inalcanzable"); },
  });
  const esperaCredencial = clienteCredencial.obtenerOpciones({ signal: controladorCredencial.signal });
  await Promise.resolve();
  controladorCredencial.abort(causaCredencial);
  await assert.rejects(esperaCredencial, comprobarAborto(causaCredencial));
  assert.equal(signalCredencial, controladorCredencial.signal);
  assert.equal(fetchTrasCredencial, false);

  const causaFetch = new Error("aborto durante fetch");
  const controladorFetch = new AbortController();
  let signalFetch;
  let resolverFetch;
  const respuestaTardia = crearRespuestaControlada({
    fragmentos: [Buffer.from(JSON.stringify({ data: opciones() }), "utf8")],
  });
  const clienteFetch = crearClienteBorradores({
    fetchImpl: (_ruta, opcionesFetch) => {
      signalFetch = opcionesFetch.signal;
      return new Promise((resolve) => { resolverFetch = resolve; });
    },
  });
  const esperaFetch = clienteFetch.obtenerOpciones({ signal: controladorFetch.signal });
  await Promise.resolve();
  controladorFetch.abort(causaFetch);
  await assert.rejects(esperaFetch, comprobarAborto(causaFetch));
  assert.equal(signalFetch, controladorFetch.signal);
  resolverFetch(respuestaTardia.respuesta);
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(respuestaTardia.traza.cancelacionesCuerpo, 1, "una respuesta tardía se descarta cerrada");

  const causaStream = new Error("aborto durante streaming");
  const controladorStream = new AbortController();
  const streamPendiente = crearRespuestaControlada({ lecturaPendiente: true });
  const clienteStream = crearClienteBorradores({ fetchImpl: async () => streamPendiente.respuesta });
  const esperaStream = clienteStream.obtenerOpciones({ signal: controladorStream.signal });
  await Promise.resolve();
  await Promise.resolve();
  controladorStream.abort(causaStream);
  await assert.rejects(esperaStream, comprobarAborto(causaStream));
  assert.ok(streamPendiente.traza.cancelacionesLector >= 1);
  assert.ok(streamPendiente.traza.cancelacionesCuerpo >= 1);

  const preAbortado = new AbortController();
  const causaPrevia = new Error("aborto previo");
  preAbortado.abort(causaPrevia);
  let credencialInvocada = false;
  const clientePreAbortado = crearClienteBorradores({
    obtenerBearer: () => { credencialInvocada = true; return "token"; },
    fetchImpl: async () => { throw new Error("inalcanzable"); },
  });
  await assert.rejects(
    clientePreAbortado.obtenerOpciones({ signal: preAbortado.signal }),
    comprobarAborto(causaPrevia),
  );
  assert.equal(credencialInvocada, false);
});

test("error envelope es cerrado, acotado y nunca muestra texto del servidor", async () => {
  const clienteValido = crearClienteBorradores({
    fetchImpl: async () => respuestaError(
      422, "contenido_borrador_invalido", "correlacion:borradores:422",
    ),
  });
  await assert.rejects(
    clienteValido.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores
      && error.estado === 422
      && error.codigo === "contenido_borrador_invalido"
      && error.correlacionRef === "correlacion:borradores:422"
      && error.envelopeValido,
  );

  const textoServidor = "detalle interno sensible del servidor";
  const clienteTexto = crearClienteBorradores({
    fetchImpl: async () => respuestaError(
      500, "error_interno", "correlacion:borradores:500", { mensaje: textoServidor },
    ),
  });
  await assert.rejects(
    clienteTexto.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores
      && error.estado === 500
      && error.codigo === "respuesta_error_no_valida"
      && !error.message.includes(textoServidor),
  );

  const demasiadoGrande = crearRespuestaControlada({
    estado: 500,
    fragmentos: [new Uint8Array(16 * 1024 + 1)],
    cabeceras: { "content-length": String(16 * 1024 + 1) },
  });
  const clienteGrande = crearClienteBorradores({ fetchImpl: async () => demasiadoGrande.respuesta });
  await assert.rejects(
    clienteGrande.obtenerOpciones(),
    (error) => error instanceof ErrorAPIBorradores
      && error.estado === 500
      && error.codigo === "respuesta_error_no_valida",
  );
  assert.ok(demasiadoGrande.traza.cancelacionesCuerpo >= 1);
});

test("fuente del cliente no contiene cookies, storage ni adaptador de presentación", async () => {
  const directorio = new URL("./", import.meta.url);
  const [api, contrato] = await Promise.all([
    readFile(new URL("portal-borradores-api.js", directorio), "utf8"),
    readFile(new URL("portal-borradores-contrato.js", directorio), "utf8"),
  ]);
  const codigo = `${api}\n${contrato}`;
  assert.doesNotMatch(codigo, /document\.cookie|localStorage|sessionStorage|datos-presentacion\.js/);
  assert.doesNotMatch(api, /new TextEncoder/);
  assert.doesNotMatch(api, /respuesta\.text\s*\(/);
  assert.match(api, /credentials: "omit"/);
  assert.match(api, /crypto|getRandomValues/);
  assert.match(api, /"Idempotency-Key"/);
  assert.match(api, /"If-Match"/);
  assert.match(api, /\/api\/vec\/bolsa\/convocatorias\/borradores\/opciones/);
  assert.match(api, /\/api\/vec\/bolsa\/convocatorias\/borradores/);
});
