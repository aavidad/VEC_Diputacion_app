import test from "node:test";
import assert from "node:assert/strict";
import { MENSAJES_PORTAL, traducirPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";
import {
  ESQUEMA_REINCORPORACIONES_TITULAR,
  LIMITES_REINCORPORACIONES_TITULAR,
  cargarReincorporacionesTitularFicha,
  consultarReincorporacionesTitular,
  manejarClickReincorporacionesTitular,
  renderizarReincorporacionesTitular,
  rutaReincorporacionesTitular,
} from "./portal-bolsas-reincorporaciones.js?v=20261001-ct-a-i18n-v1";

const escaparHTML = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const item = Object.freeze({
  evento_ref: "evento:ct:1", expediente_ref: "expediente:ct:1", relacion_ref: "relacion:1",
  fecha_efectiva: "2026-09-28", recibo_ct_ref: "recibo:ct:1", cese_evento_ref: "cese:1",
  estado: "cese_aplicado", disponible_desde: "2026-10-01", regla_version: 1,
  regla_huella_sha256: "a".repeat(64),
});
const respuesta = (status, body) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
const cuerpo = (items) => ({ data: { esquema: ESQUEMA_REINCORPORACIONES_TITULAR, items } });

test("la consulta usa la ruta canónica y GET sin credenciales persistidas ni caché", async () => {
  assert.equal(rutaReincorporacionesTitular("bolsa:1", "participacion:1"), "/api/vec/bolsa/bolsas/bolsa:1/candidatos/participacion:1/reincorporaciones-titular");
  assert.equal(rutaReincorporacionesTitular("bolsa:a.b", "participacion:a..b"), "/api/vec/bolsa/bolsas/bolsa:a.b/candidatos/participacion:a..b/reincorporaciones-titular");
  assert.equal(rutaReincorporacionesTitular("bolsa/ajena", "participacion:1"), null);
  let llamada;
  const resultado = await consultarReincorporacionesTitular("bolsa:1", "participacion:1", {
    fetchImpl: async (ruta, opciones) => { llamada = { ruta, opciones }; return respuesta(200, cuerpo([item])); },
  });
  assert.equal(resultado.ok, true);
  assert.deepEqual(resultado.datos, [item]);
  assert.equal(llamada.opciones.method, "GET");
  assert.equal(llamada.opciones.headers.Accept, "application/json");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.cache, "no-store");
  assert.equal(llamada.opciones.redirect, "error");
  assert.equal(llamada.opciones.body, undefined);
  assert.equal(new URL(llamada.ruta, "https://vec.example").pathname, llamada.ruta);
});

test("rechaza segmentos de navegación literales, codificados y doblemente codificados sin emitir GET", async () => {
  const segmentos = [".", "..", "%2e", "%2E%2E", ".%2e", "%2e.", "%252e", "%252e%252e"];
  let llamadas = 0;
  const fetchImpl = () => { llamadas++; throw new Error("no debe consultar la red"); };
  for (const segmento of segmentos) {
    for (const [bolsa, participacion] of [[segmento, "participacion:1"], ["bolsa:1", segmento]]) {
      assert.equal(rutaReincorporacionesTitular(bolsa, participacion), null, `${bolsa} / ${participacion}`);
      const resultado = await consultarReincorporacionesTitular(bolsa, participacion, { fetchImpl });
      assert.deepEqual(resultado, { ok: false, status: 400, mensaje: traducirPortal("reincorporacion_error_referencia") });
    }
  }
  assert.equal(llamadas, 0);
});

test("falla cerrado ante esquema, fechas, estado o datos inesperados", async () => {
  const invalidos = [
    { data: { esquema: "otro", items: [] } }, cuerpo({}), cuerpo([{ ...item, fecha_efectiva: "2026-02-31" }]),
    cuerpo([{ ...item, estado: "pendiente_cese" }]), cuerpo([{ ...item, recibo_ct_ref: "" }]),
    cuerpo([{ ...item, regla_huella_sha256: "no-es-huella" }]),
  ];
  for (const body of invalidos) {
    const resultado = await consultarReincorporacionesTitular("b", "p", { fetchImpl: async () => respuesta(200, body) });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_contrato"));
  }
});

test("rechaza más de 512 KiB sin Content-Length antes de decodificar JSON", async () => {
  let cancelaciones = 0;
  const body = new ReadableStream({
    start(controlador) { controlador.enqueue(new Uint8Array(4_194_304)); },
    cancel() { cancelaciones++; },
  });
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => new Response(body, { status: 200 }),
  });
  assert.equal(resultado.ok, false);
  assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_contrato"));
  assert.equal(cancelaciones, 1);
  assert.equal(LIMITES_REINCORPORACIONES_TITULAR.maximoBytes, 512 * 1024);
  assert.equal(Object.hasOwn(LIMITES_REINCORPORACIONES_TITULAR, "maximoFragmentos"), false);
});

test("acepta JSON válido de 762 bytes en 762 trozos y fragmentos vacíos", async () => {
  const texto = JSON.stringify(cuerpo([]));
  const bytes = new TextEncoder().encode(texto + " ".repeat(762 - texto.length));
  assert.equal(bytes.byteLength, 762);
  const body = new ReadableStream({
    start(controlador) {
      for (let indice = 0; indice < bytes.byteLength; indice++) {
        if (indice % 17 === 0) controlador.enqueue(new Uint8Array(0));
        controlador.enqueue(bytes.subarray(indice, indice + 1));
      }
      controlador.enqueue(new Uint8Array(0));
      controlador.close();
    },
  });
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => new Response(body, { status: 200 }),
  });
  assert.deepEqual(resultado, { ok: true, datos: [] });
});

test("90.000 fragmentos vacíos mantienen constante el número de carreras de cancelación", async () => {
  const bytes = new TextEncoder().encode(JSON.stringify(cuerpo([])));
  let vacios = 0;
  const body = new ReadableStream({
    pull(controlador) {
      if (vacios++ < 90_000) controlador.enqueue(new Uint8Array(0));
      else { controlador.enqueue(bytes); controlador.close(); }
    },
  });
  const raceOriginal = Promise.race;
  let carreras = 0;
  Promise.race = function (promesas) {
    carreras++;
    return Reflect.apply(raceOriginal, this, [promesas]);
  };
  try {
    const resultado = await consultarReincorporacionesTitular("b", "p", {
      fetchImpl: async () => new Response(body, { status: 200 }),
    });
    assert.deepEqual(resultado, { ok: true, datos: [] });
    assert.ok(carreras <= 3, `${carreras} carreras para ${vacios} fragmentos vacíos`);
  } finally {
    Promise.race = raceOriginal;
  }
});

test("un flujo infinito de fragmentos vacíos termina por tiempo sin aceptar JSON", async () => {
  let cancelaciones = 0;
  const body = new ReadableStream({
    pull(controlador) { controlador.enqueue(new Uint8Array(0)); },
    cancel() { cancelaciones++; },
  });
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => new Response(body, { status: 200 }),
    limites: { tiempoMs: 20 },
  });
  assert.equal(resultado.ok, false);
  assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_red"));
  assert.equal(cancelaciones, 1);
});

test("límite técnico menor y Content-Length excesivo cancelan el cuerpo sin leerlo", async () => {
  let lecturas = 0;
  let cancelaciones = 0;
  const respuestaGrande = {
    ok: true, status: 200, headers: { get: () => "33" },
    body: { cancel: () => { cancelaciones++; }, getReader: () => { lecturas++; throw new Error("no debe leer"); } },
  };
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => respuestaGrande,
    limites: { maximoBytes: 32 },
  });
  assert.equal(resultado.ok, false);
  assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_contrato"));
  assert.equal(lecturas, 0);
  assert.equal(cancelaciones, 1);
});

test("stream detenido termina por timeout y cancela reader aunque read no responda", async () => {
  let cancelaciones = 0;
  const lector = { read: () => new Promise(() => {}), cancel: () => { cancelaciones++; }, releaseLock() {} };
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => ({ ok: true, status: 200, headers: { get: () => null }, body: { getReader: () => lector } }),
    limites: { tiempoMs: 20 },
  });
  assert.equal(resultado.ok, false);
  assert.equal(resultado.abortada, undefined);
  assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_red"));
  assert.ok(cancelaciones >= 1);
});

test("timeout nunca acepta JSON completo de un stream nativo que no cerró", async () => {
  let cancelaciones = 0;
  const body = new ReadableStream({
    start(controlador) { controlador.enqueue(new TextEncoder().encode(JSON.stringify(cuerpo([])))); },
    cancel() { cancelaciones++; },
  });
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => new Response(body, { status: 200 }),
    limites: { tiempoMs: 20 },
  });
  assert.equal(resultado.ok, false);
  assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_red"));
  assert.equal(cancelaciones, 1);
});

test("AbortSignal externo cancela stream detenido y devuelve cancelación", async () => {
  let cancelaciones = 0;
  let lecturaEmpezada;
  const leyendo = new Promise((resolver) => { lecturaEmpezada = resolver; });
  const controlador = new AbortController();
  const lector = { read: () => { lecturaEmpezada(); return new Promise(() => {}); }, cancel: () => { cancelaciones++; }, releaseLock() {} };
  const tarea = consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => ({ ok: true, status: 200, headers: { get: () => null }, body: { getReader: () => lector } }),
    signal: controlador.signal,
  });
  await leyendo;
  controlador.abort();
  const resultado = await tarea;
  assert.equal(resultado.abortada, true);
  assert.ok(cancelaciones >= 1);
});

test("AbortSignal externo nunca acepta JSON completo de stream nativo sin fin", async () => {
  let cancelaciones = 0;
  let lecturaPendiente;
  const pendiente = new Promise((resolver) => { lecturaPendiente = resolver; });
  const body = new ReadableStream({
    start(controlador) { controlador.enqueue(new TextEncoder().encode(JSON.stringify(cuerpo([])))); },
    pull() { lecturaPendiente(); },
    cancel() { cancelaciones++; },
  });
  const controlador = new AbortController();
  const tarea = consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => new Response(body, { status: 200 }),
    signal: controlador.signal,
  });
  await pendiente;
  controlador.abort();
  const resultado = await tarea;
  assert.equal(resultado.ok, false);
  assert.equal(resultado.abortada, true);
  assert.equal(cancelaciones, 1);
});

test("rechaza respuestas sin stream sin llamar a json() ilimitado", async () => {
  let jsonLlamado = false;
  const resultado = await consultarReincorporacionesTitular("b", "p", {
    fetchImpl: async () => ({ ok: true, status: 200, body: null, json: () => { jsonLlamado = true; return cuerpo([item]); } }),
  });
  assert.equal(resultado.ok, false);
  assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_contrato"));
  assert.equal(jsonLlamado, false);
});

test("denegación y servicio pendiente se comunican sin reutilizar datos ni revelar errores internos", async () => {
  for (const [status, clave] of [[401, "reincorporacion_error_401"], [403, "reincorporacion_error_403"], [404, "reincorporacion_error_404"], [503, "reincorporacion_error_503"]]) {
    const resultado = await consultarReincorporacionesTitular("b", "p", {
      fetchImpl: async () => respuesta(status, { error: { detalle: "secreto interno" } }),
    });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.mensaje, traducirPortal(clave));
    assert.equal(JSON.stringify(resultado).includes("secreto"), false);
  }
});

test("cancela el resultado antiguo al cambiar de ficha y deja la situación sin tocar", async () => {
  let terminar;
  const actual = { candidato: { participacion_ref: "part:1", estado_clave: "trabajando" } };
  const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: actual };
  let renders = 0;
  const tarea = cargarReincorporacionesTitularFicha(actual, {
    estado, renderizar: () => { renders++; },
    consultar: () => new Promise((resolve) => { terminar = resolve; }),
  });
  assert.equal(actual.reincorporacionesTitular.carga, "cargando");
  actual.controladorReincorporaciones.abort();
  estado.modalFicha = { candidato: { participacion_ref: "part:2" } };
  terminar({ ok: true, datos: [item] });
  await tarea;
  assert.equal(actual.reincorporacionesTitular.carga, "cargando");
  assert.equal(actual.candidato.estado_clave, "trabajando");
  assert.equal(renders, 1);
});

test("la ficha muestra recibo y fecha sin inventar actor ni mutar disponibilidad", () => {
  const html = renderizarReincorporacionesTitular({ estado: { carga: "listo", items: [{ ...item, recibo_ct_ref: "recibo:<script>" }] }, escaparHTML });
  assert.match(html, /<section[^>]+aria-labelledby="reincorporacion-titulo"/);
  assert.match(html, /<h4 id="reincorporacion-titulo">Reincorporación de la persona titular<\/h4>/);
  assert.doesNotMatch(html, /Reflejo recibido desde Contratación temporal|La disponibilidad se consulta en su situación actual/);
  assert.match(html, /Cese aplicado en Bolsa/);
  assert.match(html, /recibo:&lt;script&gt;/);
  assert.match(html, /<time datetime="2026-10-01">1\/10\/26<\/time>/);
  assert.match(html, /role="region"/);
  assert.doesNotMatch(html, /actor|data-bolsa-accion="cambiar|<script>/i);
  assert.match(renderizarReincorporacionesTitular({ estado: { carga: "pendiente", error: traducirPortal("reincorporacion_error_404") }, escaparHTML }), /Reintentar consulta/);
  assert.doesNotMatch(renderizarReincorporacionesTitular({ estado: { carga: "denegado" }, escaparHTML }), /data-reincorporacion-accion="reintentar"/);
});

test("401 y 403 opcionales muestran su denegación sin reintento ni ocultar al candidato", async () => {
  for (const status of [401, 403]) {
    const modalFicha = { candidato: { participacion_ref: "participacion:1", nombre_visible: "Persona autorizada" } };
    const estado = { bolsaSeleccionada: "bolsa:1", modalFicha };
    await cargarReincorporacionesTitularFicha(modalFicha, {
      estado,
      renderizar() {},
      consultar: async () => ({ ok: false, status, mensaje: traducirPortal(`reincorporacion_error_${status}`) }),
    });
    assert.equal(modalFicha.reincorporacionesTitular.carga, "denegado");
    assert.deepEqual(modalFicha.reincorporacionesTitular.items, []);
    assert.equal(modalFicha.candidato.nombre_visible, "Persona autorizada");
    const html = renderizarReincorporacionesTitular({ estado: modalFicha.reincorporacionesTitular, escaparHTML });
    assert.match(html, /role="alert"/);
    assert.doesNotMatch(html, /<table|recibo:ct:1|data-reincorporacion-accion="reintentar"|Reflejo recibido desde Contratación temporal/);
    const accion = { dataset: { reincorporacionAccion: "reintentar" } };
    assert.equal(manejarClickReincorporacionesTitular({ target: { closest: () => accion }, preventDefault() {} }, {
      estado, renderizar() {}, consultar() { assert.fail("una denegación no se reintenta"); },
    }), true);
  }
});

test("catálogo común cubre todos los textos y el control pagina sin llamada de red", () => {
  assert.equal(typeof MENSAJES_PORTAL.reincorporacion_titulo, "string");
  assert.equal(traducirPortal("reincorporacion_mostrando", { desde: 1, hasta: 2, total: 3 }), "Mostrando 1 a 2 de 3");
  const modalFicha = { reincorporacionesTitular: { carga: "listo", items: Array(8).fill(item), pagina: 0 } };
  const estado = { modalFicha };
  let renders = 0;
  const control = { dataset: { reincorporacionAccion: "pagina", pagina: "1" } };
  const evento = { target: { closest: () => control }, preventDefault() {} };
  assert.equal(manejarClickReincorporacionesTitular(evento, { estado, renderizar: () => { renders++; } }), true);
  assert.equal(modalFicha.reincorporacionesTitular.pagina, 1);
  assert.equal(renders, 1);
});
