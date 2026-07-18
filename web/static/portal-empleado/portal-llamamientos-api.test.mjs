import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClientePropuestasLlamamiento } from "./portal-llamamientos-api.js";

const HUELLA = "a".repeat(64);
const ETAG = `"vec-propuesta-llamamiento-v1.sha256-${HUELLA}"`;
const NECESIDAD = "necesidad:01K0VS7R";
const MAXIMO_RESPUESTA = 8 * 1024;
const MAXIMO_FRAGMENTOS = 256;
const CODIFICADOR = new TextEncoder();

function confirmacionValida() {
  const version = (referencia, caracter) => ({
    referencia,
    version: "1",
    huella_sha256: caracter.repeat(64),
  });
  return {
    esquema: "vec.bolsa.propuesta-llamamiento.confirmacion.v1",
    propuesta_ref: "propuesta:01K0VS7P",
    huella_propuesta_sha256: HUELLA,
    bolsa: version("bolsa:01K0VS7Q", "b"),
    necesidad: version(NECESIDAD, "c"),
    instantanea: version("instantanea:01K0VS7S", "d"),
    politica: version("politica:01K0VS7T", "e"),
    instante_referencia: "2026-07-18T08:00:00Z",
    instantanea_generada_en: "2026-07-18T08:00:00Z",
    total_participaciones_instantanea: "10",
    total_evaluaciones: "2",
    orden_seleccionado: "2",
    generada_en: "2026-07-18T08:00:01Z",
  };
}

function respuestaControlada({
  datos = confirmacionValida(),
  etag = ETAG,
  estado = 201,
  tipo = "application/json; charset=utf-8",
  texto,
  bytes,
  fragmentos,
  longitud,
  incluirLongitud = true,
} = {}) {
  const carga = bytes || CODIFICADOR.encode(texto ?? JSON.stringify({ data: datos }));
  const partes = fragmentos || [carga];
  const traza = {
    cancelacionesCuerpo: 0,
    cancelacionesLector: 0,
    liberaciones: 0,
    materializaciones: 0,
    lecturas: 0,
  };
  let indice = 0;
  const lector = {
    async read() {
      traza.lecturas += 1;
      if (indice >= partes.length) return { done: true, value: undefined };
      const value = partes[indice];
      indice += 1;
      return { done: false, value };
    },
    async cancel() { traza.cancelacionesLector += 1; },
    releaseLock() { traza.liberaciones += 1; },
  };
  const cabeceras = new Map();
  if (tipo !== null) cabeceras.set("content-type", tipo);
  if (etag !== null) cabeceras.set("etag", etag);
  if (incluirLongitud) cabeceras.set("content-length", longitud ?? String(carga.byteLength));
  const respuesta = {
    status: estado,
    headers: { get: (nombre) => cabeceras.get(nombre.toLowerCase()) ?? null },
    body: {
      getReader: () => lector,
      async cancel() { traza.cancelacionesCuerpo += 1; },
    },
    async json() { traza.materializaciones += 1; throw new Error("json() prohibido"); },
    async text() { traza.materializaciones += 1; throw new Error("text() prohibido"); },
  };
  return { respuesta, traza, carga };
}

function respuestaCreada(datos = confirmacionValida(), etag = ETAG) {
  return respuestaControlada({ datos, etag }).respuesta;
}

test("el cliente emite exactamente el POST y las dos cabeceras permitidas", async () => {
  const llamadas = [];
  const cliente = crearClientePropuestasLlamamiento({
    fetchImpl: async (...argumentos) => {
      llamadas.push(argumentos);
      return respuestaCreada();
    },
  });
  const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad: true });
  assert.equal(resultado.ok, true);
  assert.equal(resultado.confirmacion.necesidad.referencia, NECESIDAD);
  assert.equal(resultado.etag, ETAG);
  assert.equal(llamadas.length, 1);
  const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, "/api/vec/bolsa/propuestas-llamamiento");
  assert.deepEqual(opciones, {
    method: "POST",
    credentials: "omit",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify({
      data: {
        esquema: "vec.bolsa.propuesta-llamamiento.solicitud.v1",
        necesidad_id: NECESIDAD,
      },
    }),
  });
});

test("una capacidad distinta de true bloquea sin ejecutar fetch", async () => {
  let llamadas = 0;
  const cliente = crearClientePropuestasLlamamiento({
    fetchImpl: async () => { llamadas += 1; return respuestaCreada(); },
  });
  for (const capacidad of [false, undefined, null, 1, "true"]) {
    const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.bloqueada, true);
  }
  assert.equal(llamadas, 0);
});

test("la respuesta falla cerrada ante extras, ETag ausente o necesidad distinta", async () => {
  const extra = { ...confirmacionValida(), detalle: [] };
  for (const respuesta of [
    respuestaCreada(extra),
    respuestaCreada(confirmacionValida(), null),
    respuestaCreada({ ...confirmacionValida(), necesidad: {
      ...confirmacionValida().necesidad,
      referencia: "necesidad:OTRA",
    } }),
    { ...respuestaCreada(), status: 200 },
  ]) {
    const cliente = crearClientePropuestasLlamamiento({ fetchImpl: async () => respuesta });
    const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad: true });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.bloqueada, true);
  }
});

test("Content-Length es obligatorio, decimal canónico, positivo y como máximo 8 KiB", async () => {
  const casos = [
    respuestaControlada({ incluirLongitud: false }),
    ...["", "01", "+1", " 10", "10 ", "1e3", "-1", "8192.0"].map(
      (longitud) => respuestaControlada({ longitud }),
    ),
    respuestaControlada({ longitud: "0" }),
    respuestaControlada({ longitud: String(MAXIMO_RESPUESTA + 1) }),
  ];
  for (const controlada of casos) {
    const cliente = crearClientePropuestasLlamamiento({
      fetchImpl: async () => controlada.respuesta,
    });
    const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad: true });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.bloqueada, true);
    assert.equal(controlada.traza.lecturas, 0);
    assert.ok(controlada.traza.cancelacionesCuerpo >= 1);
    assert.equal(controlada.traza.materializaciones, 0);
  }
});

test("la lectura streaming exige que el tamaño real coincida exactamente con el declarado", async () => {
  const normal = respuestaControlada();
  const corta = respuestaControlada({
    bytes: normal.carga,
    longitud: String(normal.carga.byteLength + 1),
  });
  const larga = respuestaControlada({
    bytes: normal.carga,
    longitud: String(normal.carga.byteLength - 1),
  });
  for (const controlada of [corta, larga]) {
    const cliente = crearClientePropuestasLlamamiento({ fetchImpl: async () => controlada.respuesta });
    const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad: true });
    assert.equal(resultado.ok, false);
    assert.match(resultado.mensaje, /no coincide con Content-Length/);
    assert.ok(controlada.traza.cancelacionesLector >= 1);
    assert.equal(controlada.traza.liberaciones, 1);
    assert.equal(controlada.traza.materializaciones, 0);
  }
});

test("el byte 8.193 se rechaza y cancela tanto si se declara como si llega oculto", async () => {
  const declarado = respuestaControlada({ longitud: String(MAXIMO_RESPUESTA + 1) });
  const oculto = respuestaControlada({
    bytes: new Uint8Array(MAXIMO_RESPUESTA + 1).fill(0x20),
    longitud: String(MAXIMO_RESPUESTA),
  });

  const clienteDeclarado = crearClientePropuestasLlamamiento({
    fetchImpl: async () => declarado.respuesta,
  });
  const resultadoDeclarado = await clienteDeclarado.solicitar({
    necesidadId: NECESIDAD,
    capacidad: true,
  });
  assert.equal(resultadoDeclarado.ok, false);
  assert.match(resultadoDeclarado.mensaje, /límite de 8 KiB/);
  assert.ok(declarado.traza.cancelacionesCuerpo >= 1);
  assert.equal(declarado.traza.lecturas, 0);

  const clienteOculto = crearClientePropuestasLlamamiento({ fetchImpl: async () => oculto.respuesta });
  const resultadoOculto = await clienteOculto.solicitar({ necesidadId: NECESIDAD, capacidad: true });
  assert.equal(resultadoOculto.ok, false);
  assert.match(resultadoOculto.mensaje, /no coincide con Content-Length/);
  assert.ok(oculto.traza.cancelacionesLector >= 1);
  assert.equal(oculto.traza.liberaciones, 1);
});

test("UTF-8 fatal y JSON inválido o no exacto fallan cerrados sin materializar", async () => {
  const utf8 = respuestaControlada({ bytes: Uint8Array.of(0xC3, 0x28), longitud: "2" });
  const jsonRoto = respuestaControlada({ texto: "{" });
  const textoValido = JSON.stringify({ data: confirmacionValida() });
  const jsonConEspacio = respuestaControlada({ texto: `${textoValido} ` });
  const jsonDuplicado = respuestaControlada({
    texto: `{"data":${JSON.stringify(confirmacionValida())},"data":${JSON.stringify(confirmacionValida())}}`,
  });
  const casos = [
    [utf8, /UTF-8 válido/],
    [jsonRoto, /JSON no válido/],
    [jsonConEspacio, /representación JSON exacta/],
    [jsonDuplicado, /representación JSON exacta/],
  ];
  for (const [controlada, patron] of casos) {
    const cliente = crearClientePropuestasLlamamiento({ fetchImpl: async () => controlada.respuesta });
    const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad: true });
    assert.equal(resultado.ok, false);
    assert.match(resultado.mensaje, patron);
    assert.ok(controlada.traza.cancelacionesCuerpo + controlada.traza.cancelacionesLector >= 1);
    assert.equal(controlada.traza.materializaciones, 0);
  }
  assert.ok(utf8.traza.cancelacionesLector >= 1);
  assert.equal(utf8.traza.liberaciones, 1);
});

test("el límite de fragmentos admite 256 y cancela el fragmento 257 incluso si está vacío", async () => {
  const base = respuestaControlada();
  const fragmentar = (cantidad) => respuestaControlada({
    bytes: base.carga,
    fragmentos: [base.carga, ...Array.from({ length: cantidad - 1 }, () => new Uint8Array())],
    longitud: String(base.carga.byteLength),
  });
  const exacta = fragmentar(MAXIMO_FRAGMENTOS);
  const clienteExacto = crearClientePropuestasLlamamiento({ fetchImpl: async () => exacta.respuesta });
  assert.equal((await clienteExacto.solicitar({ necesidadId: NECESIDAD, capacidad: true })).ok, true);
  assert.equal(exacta.traza.cancelacionesLector, 0);
  assert.equal(exacta.traza.liberaciones, 1);

  const excesiva = fragmentar(MAXIMO_FRAGMENTOS + 1);
  const clienteExceso = crearClientePropuestasLlamamiento({ fetchImpl: async () => excesiva.respuesta });
  const resultado = await clienteExceso.solicitar({ necesidadId: NECESIDAD, capacidad: true });
  assert.equal(resultado.ok, false);
  assert.match(resultado.mensaje, /demasiados fragmentos/);
  assert.ok(excesiva.traza.cancelacionesLector >= 1);
  assert.equal(excesiva.traza.liberaciones, 1);
});

test("el cliente falla cerrado si el cuerpo no expone un lector incremental cancelable", async () => {
  const controlada = respuestaControlada();
  controlada.respuesta.body = {
    async cancel() { controlada.traza.cancelacionesCuerpo += 1; },
  };
  const cliente = crearClientePropuestasLlamamiento({ fetchImpl: async () => controlada.respuesta });
  const resultado = await cliente.solicitar({ necesidadId: NECESIDAD, capacidad: true });
  assert.equal(resultado.ok, false);
  assert.match(resultado.mensaje, /lectura incremental acotada/);
  assert.ok(controlada.traza.cancelacionesCuerpo >= 1);
  assert.equal(controlada.traza.materializaciones, 0);
});

test("el adaptador no implementa lecturas inexistentes ni añade identidad o reintentos", async () => {
  const fuente = await readFile(new URL("portal-llamamientos-api.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /method:\s*"GET"|\/propuestas-llamamiento\//);
  assert.doesNotMatch(fuente, /Idempotency-Key|Authorization|Cookie|credentials:\s*"include"/i);
  assert.doesNotMatch(fuente, /setTimeout|retry|reintento/i);
  assert.doesNotMatch(fuente, /respuesta\.(?:json|text)\s*\(/);
});
