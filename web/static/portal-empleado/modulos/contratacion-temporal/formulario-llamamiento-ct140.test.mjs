import assert from "node:assert/strict";
import test from "node:test";
import {
  EXPEDIENTE, recibo, raizPrueba, montar, archivoCorreo, declaracion, justificante,
} from "./formulario-llamamiento-pruebas.js";
import { MENSAJES_LLAMAMIENTO_EN } from "./i18n-llamamiento.js";

const fila = (n) => ({
  organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
  llamamiento_ref: `llamamiento:ct140:${String(n).padStart(2, "0")}`,
  comunicacion_ref: `comunicacion:ct140:${String(n).padStart(2, "0")}`,
  version: 2, estado: "registrada_localmente",
  registrada_en: `2026-09-24T10:${String(n).padStart(2, "0")}:00Z`,
  recibo_comunicacion_ref: `recibo:comunicacion:${n}`,
  antecedente_tipo: n === 12 ? "continuacion_confirmada" : "seleccion_confirmada",
  recibo_antecedente_ref: `recibo:antecedente:${n}`,
});
const cursor = `${fila(10).comunicacion_ref}#${"a".repeat(64)}`;
const contexto = { expediente_ref: EXPEDIENTE, version_esperada: 6 };
const sinRespuesta = () => Object.assign(new Error("sin respuesta"), {
  estado: 404, codigo: "recurso_no_encontrado", envelopeValido: true,
});
const esperar = () => new Promise(setImmediate);
const elegir = (raiz, indice) => raiz.eventos.get("click")({ preventDefault() {}, target: {
  closest: (selector) => selector === "[data-ct-comunicacion-indice]"
    ? { dataset: { ctComunicacionIndice: String(indice) } } : null,
} });

for (const ingles of [false, true]) test(`CT140 pagina completa 12 filas, ordinal ${ingles ? "EN" : "ES"}, GET404 y POST con antecedente`, async () => {
  const raiz = raizPrueba(), consultas = [], escrituras = [];
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async (consulta) => {
      consultas.push(consulta);
      return consulta.cursor
        ? { expediente_ref: EXPEDIENTE, comunicaciones: [fila(11), fila(12)] }
        : { expediente_ref: EXPEDIENTE, comunicaciones: Array.from({ length: 10 }, (_, i) => fila(i + 1)),
          siguiente_cursor: cursor };
    },
    consultarReciboRespuesta: async () => { throw sinRespuesta(); },
    registrarRespuestaRecibida: async (solicitud) => {
      escrituras.push(solicitud);
      return { ...justificante(solicitud),
        estado: ingles ? "replay_registrada_por_rrhh" : "registrada_por_rrhh" };
    },
  }, { contexto, ...(ingles ? { mensajes: MENSAJES_LLAMAMIENTO_EN, locale: "en-GB" } : {}) });
  await esperar();
  assert.equal(consultas.length, 2);
  assert.deepEqual(consultas[1], { expediente_ref: EXPEDIENTE, cursor });
  assert.match(raiz.innerHTML, ingles
    ? /Call 2 of 12 · 24\/09\/2026 · Reply status to check/u
    : /Llamamiento 2 de 12 · 24\/09\/2026 · Respuesta por comprobar/u);
  assert.doesNotMatch(raiz.innerHTML, /comunicacion:ct140|llamamiento:ct140|recibo:antecedente/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  elegir(raiz, 11);
  await esperar();
  assert.match(raiz.innerHTML, ingles ? /Response pending/u : /Respuesta pendiente/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta_siguiente"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion_siguiente"/u);
  await raiz.archivo(archivoCorreo(), "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", declaracion());
  assert.equal(escrituras.length, 1);
  assert.equal(escrituras[0].comunicacion_ref, fila(12).comunicacion_ref);
  assert.equal(escrituras[0].llamamiento_ref, fila(12).llamamiento_ref);
  assert.equal(escrituras[0].version_comunicacion_esperada, 2);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="respuesta_siguiente"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="resolucion_siguiente"/u);
  cerrar();
});

test("CT140 descarta páginas parciales ante 503 de cursor y bloquea POST", async () => {
  const raiz = raizPrueba(); let llamadas = 0, escrituras = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async (consulta) => {
      llamadas += 1;
      if (consulta.cursor) throw Object.assign(new Error("ancla cambiada"), {
        estado: 503, codigo: "servicio_no_disponible", envelopeValido: true,
      });
      return { expediente_ref: EXPEDIENTE,
        comunicaciones: Array.from({ length: 10 }, (_, i) => fila(i + 1)), siguiente_cursor: cursor };
    },
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  }, { contexto });
  await esperar();
  assert.equal(llamadas, 2);
  assert.match(raiz.innerHTML, /No se pudo comprobar la lista completa/u);
  assert.doesNotMatch(raiz.innerHTML, /Llamamiento 1 de|data-ct-comunicacion-indice|data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  cerrar();
});

test("CT140 no presenta ordinal al llegar a 100 filas con cursor restante", async () => {
  const raiz = raizPrueba(); let paginas = 0;
  const cerrar = montar(raiz, { consultarComunicacionesExpediente: async () => {
    const inicio = paginas * 10 + 1;
    paginas += 1;
    const comunicaciones = Array.from({ length: 10 }, (_, i) => {
      const n = inicio + i;
      return { ...fila(n), registrada_en: new Date(Date.UTC(2026, 8, 24, 10, n)).toISOString().replace(".000Z", "Z") };
    });
    return { expediente_ref: EXPEDIENTE, comunicaciones,
      siguiente_cursor: `${comunicaciones.at(-1).comunicacion_ref}#${"a".repeat(64)}` };
  } }, { contexto });
  await esperar();
  assert.equal(paginas, 10);
  assert.match(raiz.innerHTML, /No se pudo comprobar la lista completa/u);
  assert.doesNotMatch(raiz.innerHTML, /Llamamiento 1 de|data-ct-comunicacion-indice|data-ct-llamamiento-form=/u);
  cerrar();
});

test("CT140 y GET200 muestran respuesta existente sin formulario ni nuevo POST", async () => {
  const raiz = raizPrueba(); let escrituras = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] }),
    consultarReciboRespuesta: async () => ({
      esquema: "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1",
      organizacion_ref: fila(1).organizacion_ref, comunicacion_ref: fila(1).comunicacion_ref,
      respuesta: "aceptacion", justificante_ref: "justificante:ct140:001",
      recibo_ref: "recibo:respuesta:ct140:001", auditoria_ref: "auditoria:ct140:001",
      registrada_en: "2026-09-24T11:00:00Z", estado: "registrada_por_rrhh",
    }),
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  }, { contexto });
  await esperar();
  elegir(raiz, 0);
  await esperar();
  assert.match(raiz.innerHTML, /Llamamiento 1 de 1 · 24\/09\/2026 · Respuesta registrada/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="consultaRespuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  cerrar();
});

test("CT140 con antecedente de selección habilita respuesta original tras GET404", async () => {
  const raiz = raizPrueba(), escrituras = [];
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] }),
    consultarReciboRespuesta: async () => { throw sinRespuesta(); },
    registrarRespuestaRecibida: async (solicitud) => {
      escrituras.push(solicitud); return justificante(solicitud);
    },
  }, { contexto });
  await esperar();
  elegir(raiz, 0);
  await esperar();
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion"/u);
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras.length, 1);
  assert.equal(escrituras[0].comunicacion_ref, fila(1).comunicacion_ref);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
  cerrar();
});

test("al cambiar de fila ignora respuesta tardía de la comunicación anterior", async () => {
  const raiz = raizPrueba(); let resolverAnterior, signalAnterior;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE,
      comunicaciones: [fila(1), fila(2)] }),
    consultarReciboRespuesta: (consulta, { signal }) => consulta.comunicacion_ref === fila(1).comunicacion_ref
      ? new Promise((resolve) => { resolverAnterior = resolve; signalAnterior = signal; })
      : Promise.reject(sinRespuesta()),
  }, { contexto });
  await esperar();
  elegir(raiz, 0);
  await esperar();
  elegir(raiz, 1);
  await esperar();
  assert.equal(signalAnterior.aborted, true);
  resolverAnterior({ esquema: "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1",
    organizacion_ref: fila(1).organizacion_ref, comunicacion_ref: fila(1).comunicacion_ref,
    respuesta: "aceptacion", justificante_ref: "justificante:ct140:001",
    recibo_ref: "recibo:respuesta:ct140:001", auditoria_ref: "auditoria:ct140:001",
    registrada_en: "2026-09-24T11:00:00Z", estado: "registrada_por_rrhh" });
  await esperar();
  assert.match(raiz.innerHTML, /Llamamiento 2 de 2 · 24\/09\/2026 · Respuesta pendiente/u);
  assert.doesNotMatch(raiz.innerHTML, /recibo:respuesta:ct140:001/u);
  cerrar();
});

test("CT140 deniega la lista sin filtrar filas y aborta al cambiar de expediente", async () => {
  const denegada = raizPrueba();
  const cerrarDenegada = montar(denegada, { consultarComunicacionesExpediente: async () => {
    throw Object.assign(new Error(), { estado: 403, codigo: "acceso_denegado", envelopeValido: true });
  } }, { contexto });
  await esperar();
  assert.match(denegada.innerHTML, /No puede consultar estos llamamientos/u);
  assert.doesNotMatch(denegada.innerHTML, /data-ct-comunicacion-indice|data-ct-llamamiento-form=/u);
  cerrarDenegada();

  const raiz = raizPrueba(); let signal, resolver;
  const cerrar = montar(raiz, { consultarComunicacionesExpediente: (_consulta, opciones) => {
    signal = opciones.signal;
    return new Promise((resolve) => { resolver = resolve; });
  } }, { contexto });
  await esperar();
  assert.match(raiz.innerHTML, /Comprobando todos los llamamientos/u);
  cerrar.actualizarContexto({ expediente_ref: "expediente:otro:002", version_esperada: 6 });
  assert.equal(signal.aborted, true);
  resolver({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] });
  await esperar();
  assert.equal(raiz.innerHTML, "");
});
