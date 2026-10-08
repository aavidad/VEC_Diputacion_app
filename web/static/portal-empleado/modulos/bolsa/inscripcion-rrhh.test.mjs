import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteInscripcionesRRHH } from "./inscripcion-rrhh-cliente.js";
import { leerRutaInscripcionesRRHH, rutaInscripcionesRRHH,
  montarInscripcionesRRHH } from "./inscripcion-rrhh-vista.js";
import { cargarTextos } from "../../../comun/textos.js";

const solicitud = { solicitud_ref: "solicitud:1", recibo_ref: "recibo:solicitud:1",
  convocatoria_ref: "convocatoria:1", categoria_ref: "categoria:1", categoria: "Auxiliar administrativo",
  declaracion_ref: "declaracion:1", bolsa_ref: null, persona_resumen: "Lucía Martín",
  estado: "pendiente", version: 1, registrada_en: "2026-10-08T10:00:00Z", motivo_codigo: null };
const detalle = { ...solicitud, bases_ref: "bases:1", catalogo_version: 1,
  plazo_inicio: "2026-10-01T00:00:00Z", plazo_fin: "2026-10-31T23:59:59Z",
  requisitos: [{ codigo: "titulo", descripcion: "Titulación requerida", obligatorio: true,
    estado: "cumple", fuente_ref: "fuente:1", evidencia_ref: "evidencia:1" }] };
const respuesta = (status, data) => ({ ok: status >= 200 && status < 300, status, redirected: false,
  headers: { get: (nombre) => nombre.toLowerCase() === "content-type" ? "application/json" : null },
  text: async () => JSON.stringify({ data }) });

test("cliente RRHH conserva filtros, identidad fuera del cuerpo y valida recibo", async () => {
  const llamadas = [];
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    if (ruta.includes("/decisiones")) return respuesta(201, { esquema: "vec.bolsa.inscripcion.decision.recibo.v1",
      solicitud_ref: "solicitud:1", recibo_ref: "recibo:1", estado: "admitida_a_convocatoria", version: 2,
      decidida_en: "2026-10-08T11:00:00Z", participacion_ref: null, repetida: false });
    return respuesta(200, { esquema: "vec.bolsa.inscripciones.rrhh.v1", solicitudes: [solicitud],
      total: 1, cursor_siguiente: null });
  } });
  assert.equal((await cliente.listar({ convocatoria: "convocatoria:1", idioma: "en" })).total, 1);
  const recibo = await cliente.decidir({ solicitudRef: "solicitud:1", decision: "admitir", versionEsperada: 1,
    claveIdempotencia: "inscripcion-12345678" });
  assert.equal(recibo.estado, "admitida_a_convocatoria");
  assert.equal(recibo.participacion_ref, null);
  assert.match(llamadas[0].ruta, /estado=pendiente.*convocatoria_ref=convocatoria%3A1.*idioma=en/u);
  assert.equal(llamadas[1].opciones.credentials, "same-origin");
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), { decision: "admitir", version_esperada: 1,
    clave_idempotencia: "inscripcion-12345678" });
});

test("una respuesta de otra solicitud o sin recibo no confirma la decisión", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => respuesta(200,
    { esquema: "vec.bolsa.inscripcion.rrhh.v1", solicitud: { ...detalle, solicitud_ref: "solicitud:ajena" } }) });
  await assert.rejects(cliente.detalle("solicitud:1"), /detalle incompatible/u);
  await assert.rejects(cliente.decidir({ solicitudRef: "solicitud:1", decision: "rechazar",
    motivoCodigo: "base", versionEsperada: 1, claveIdempotencia: "inscripcion-12345678" }), /recibo incompatible/u);
});

test("incorporación pide evidencia y acepta solo una participación verificada por el servidor", async () => {
  let cuerpo;
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (ruta, opciones) => {
    assert.match(ruta, /\/incorporaciones$/u);
    cuerpo = JSON.parse(opciones.body);
    return respuesta(201, { esquema: "vec.bolsa.inscripcion.incorporacion.recibo.v1",
      solicitud_ref: "solicitud:1", recibo_ref: "recibo:incorporacion:1", estado: "incorporada",
      version: 3, participacion_ref: "participacion:1", incorporada_en: "2026-10-09T00:00:00Z",
      repetida: false });
  } });
  const recibo = await cliente.incorporar({ solicitudRef: "solicitud:1", evidenciaRef: "acta:1",
    versionEsperada: 2, claveIdempotencia: "inscripcion-12345678" });
  assert.equal(recibo.participacion_ref, "participacion:1");
  assert.deepEqual(cuerpo, { evidencia_ref: "acta:1", version_esperada: 2,
    clave_idempotencia: "inscripcion-12345678" });
});

test("admisión que dice incorporada exige referencia de participación", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => respuesta(201,
    { esquema: "vec.bolsa.inscripcion.decision.recibo.v1", solicitud_ref: "solicitud:1",
      recibo_ref: "recibo:1", estado: "incorporada", version: 2,
      decidida_en: "2026-10-09T00:00:00Z", repetida: false }) });
  await assert.rejects(cliente.decidir({ solicitudRef: "solicitud:1", decision: "admitir",
    versionEsperada: 1, claveIdempotencia: "inscripcion-12345678" }), /recibo incompatible/u);
});

test("los motivos se consultan sólo en el idioma activo", async () => {
  let ruta;
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (valor) => {
    ruta = valor;
    return respuesta(200, { esquema: "vec.bolsa.inscripcion.motivos.v1", catalogo_version: 3,
      motivos: [{ codigo: "falta_requisito", etiqueta: "Missing requirement", obligatorio: true }] });
  } });
  assert.equal((await cliente.motivos("rechazar", { idioma: "en" })).motivos[0].etiqueta, "Missing requirement");
  assert.equal(ruta, "/api/vec/bolsa/rrhh/inscripciones/motivos?decision=rechazar&idioma=en");
});

test("una respuesta excesiva se corta antes de convertirla en JSON", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => ({ ok: true, status: 200,
    redirected: false, headers: { get: (nombre) => nombre === "content-type" ? "application/json" : null },
    body: new ReadableStream({ start(controlador) { controlador.enqueue(new Uint8Array(256 * 1024 + 1)); } }),
    text: async () => { throw new Error("no debe leerse completa"); } }) });
  await assert.rejects(cliente.listar(), /respuesta excesiva/u);
});

test("filtro compartible conserva la URL y normaliza estado desconocido", () => {
  assert.deepEqual(leerRutaInscripcionesRRHH("?inscripcion_estado=otro&inscripcion_convocatoria=convocatoria%3A1"),
    { estado: "pendiente", convocatoria: "convocatoria:1", cursor: "" });
  assert.equal(rutaInscripcionesRRHH("https://vec.example/portal-empleado/?lang=en#solicitudes",
    { estado: "rechazada", convocatoria: "convocatoria:1", cursor: "c:2" }),
  "/portal-empleado/?lang=en&inscripcion_estado=rechazada&inscripcion_convocatoria=convocatoria%3A1&inscripcion_cursor=c%3A2#solicitudes");
});

test("catálogos propios de la pantalla tienen las mismas claves y no necesitan el otro idioma al cargar", async () => {
  const es = await cargarTextos("bolsa-inscripcion-rrhh", { idioma: "es", porDefecto: "es" });
  const en = await cargarTextos("bolsa-inscripcion-rrhh", { idioma: "en", porDefecto: "es" });
  assert.deepEqual(Object.keys(es.seccion("rrhh")), Object.keys(en.seccion("rrhh")));
  assert.equal(en.traducir("rrhh.estado_pendiente"), "Pending");
});

test("denegación borra lista y la respuesta tardía no vuelve a mostrarla", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  const localizacion = new URL("https://vec.example/portal-empleado/#solicitudes");
  let denegaciones = 0;
  const vista = await montarInscripcionesRRHH({ raiz, localizacion, historial: { pushState() {} },
    alDenegacion: () => { denegaciones++; },
    cliente: { listar: async () => { throw Object.assign(new Error("denegado"), { estado: 403 }); },
      detalle: async () => solicitud, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(denegaciones, 1);
  assert.match(raiz.innerHTML, /No puede consultar estas solicitudes/u);
  assert.doesNotMatch(raiz.innerHTML, /Lucía Martín/u);
  vista.desmontar();
  assert.equal(raiz.innerHTML, "");
});

test("RRHH no puede confirmar admisión con un requisito obligatorio pendiente", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/#solicitudes"),
    historial: { pushState() {} },
    cliente: { listar: async () => ({ solicitudes: [solicitud], total: 1, cursor_siguiente: null }),
      detalle: async () => ({ ...detalle, requisitos: [{ ...detalle.requisitos[0], estado: "pendiente" }] }),
      motivos: async () => ({ motivos: [] }), decidir: async () => { throw new Error("no debe decidir"); },
      incorporar: async () => { throw new Error("no debe incorporar"); } } });
  await new Promise((r) => setTimeout(r, 0));
  const accion = { dataset: { inscripcionAbrir: "solicitud:1" }, matches: () => false };
  eventos.get("click")({ target: { closest: () => accion } });
  await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /data-inscripcion-decidir="admitir" disabled/u);
  assert.match(raiz.innerHTML, /Pendiente de comprobar/u);
  assert.match(raiz.innerHTML, /Auxiliar administrativo/u);
  vista.desmontar();
});
