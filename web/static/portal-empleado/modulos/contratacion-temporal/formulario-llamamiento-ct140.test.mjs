import assert from "node:assert/strict";
import test from "node:test";
import {
  EXPEDIENTE, recibo, raizPrueba, montar, archivoCorreo, declaracion, justificante,
  CLAVE_RESOLUCION, revisionManual, reciboResolucion, continuacionConfirmada,
  PUBLICACIONES_PROPUESTA,
} from "./formulario-llamamiento-pruebas.js?v=20261001-ct-firma-verificador-v2";
import { cargarCatalogosContratacionEnIdioma } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";

const MENSAJES_LLAMAMIENTO_EN = (await cargarCatalogosContratacionEnIdioma(
  "contratacion-temporal-llamamiento", "en",
)).actual;

const fila = (n) => ({
  organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
  llamamiento_ref: `llamamiento:ct140:${String(n).padStart(2, "0")}`,
  comunicacion_ref: `comunicacion:ct140:${String(n).padStart(2, "0")}`,
  version: 2, estado: "registrada_localmente",
  registrada_en: `2026-09-24T10:${String(n).padStart(2, "0")}:00Z`,
  recibo_comunicacion_ref: `recibo:comunicacion:${n}`,
  antecedente_tipo: n === 12 ? "continuacion_confirmada" : "seleccion_confirmada",
  recibo_antecedente_ref: `recibo:antecedente:${n}`,
  estado_respuesta: "sin_respuesta",
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

for (const ingles of [false, true]) test(`CT140 sin_respuesta en 12 filas, ${ingles ? "EN" : "ES"}, GET404 y POST`, async () => {
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
    ? /Call-up 2 of 12 · 24\/09\/2026 · Reply not checked/u
    : /Llamamiento 2 de 12 · 24\/09\/2026 · Respuesta sin comprobar/u);
  assert.doesNotMatch(raiz.innerHTML, /comunicacion:ct140|llamamiento:ct140|recibo:antecedente/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  elegir(raiz, 11);
  await esperar();
  assert.match(raiz.innerHTML, ingles ? /No reply/u : /Sin respuesta/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta_siguiente"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion_siguiente"/u);
  await raiz.archivo(archivoCorreo(), "respuesta_siguiente");
  await raiz.enviar("respuesta_siguiente", declaracion());
  assert.equal(escrituras.length, 1);
  assert.equal(escrituras[0].comunicacion_ref, fila(12).comunicacion_ref);
  assert.equal(escrituras[0].llamamiento_ref, fila(12).llamamiento_ref);
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
  assert.match(raiz.innerHTML, /No se han podido cargar los llamamientos\. Pulse «Volver a ca/u);
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
  assert.match(raiz.innerHTML, /No se han podido cargar los llamamientos\. Pulse «Volver a ca/u);
  assert.doesNotMatch(raiz.innerHTML, /Llamamiento 1 de|data-ct-comunicacion-indice|data-ct-llamamiento-form=/u);
  cerrar();
});

test("CT140 y GET200 muestran respuesta existente sin formulario ni nuevo POST", async () => {
  const raiz = raizPrueba(); let escrituras = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] }),
    consultarReciboRespuesta: async () => ({
      esquema: "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1",
      organizacion_ref: fila(1).organizacion_ref, expediente_ref: fila(1).expediente_ref,
      comunicacion_ref: fila(1).comunicacion_ref,
      respuesta: "aceptacion", justificante_ref: "justificante:ct140:001",
      recibo_ref: "recibo:respuesta:ct140:001", auditoria_ref: "auditoria:ct140:001",
      registrada_en: "2026-09-24T11:00:00Z", estado: "registrada_por_rrhh",
    }),
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  }, { contexto });
  await esperar();
  elegir(raiz, 0);
  await esperar();
  assert.match(raiz.innerHTML, /Llamamiento 1 de 1 · 24\/09\/2026 · Respuesta anotada/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="consultaRespuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  cerrar();
});

test("CT140 sin_respuesta con selección permite POST tras GET404 sin fabricar recibo", async () => {
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
  assert.match(raiz.innerHTML, /Todavía no hay ninguna respuesta anotada\. Si ya la tiene, an/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion"/u);
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras.length, 1);
  assert.equal(escrituras[0].comunicacion_ref, fila(1).comunicacion_ref);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
  cerrar();
});

test("renuncia restaurada permite resolución y siguiente llamamiento sin saltar guardas", async () => {
  const raiz = raizPrueba(); let resoluciones = 0, siguientes = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] }),
    consultarReciboRespuesta: async () => { throw sinRespuesta(); },
    registrarRespuestaRecibida: async (solicitud) => ({ ...justificante(solicitud),
      registrada_en: "2026-09-24T10:30:00Z" }),
    resolverLlamamiento: async () => {
      resoluciones += 1;
      return { ...reciboResolucion("renuncia"), resuelta_en: "2026-09-24T11:00:00Z",
        intencion_siguiente: { referencia: "intencion:siguiente:001", estado_local: "pendiente",
          actualizada_en: "2026-09-24T11:00:00Z" } };
    },
    continuarLlamamiento: async () => {
      siguientes += 1;
      return { ...continuacionConfirmada,
        llamamiento_anterior_ref: fila(1).llamamiento_ref,
        confirmada_en: "2026-09-24T11:10:00Z" };
    },
  }, { contexto });
  await esperar(); elegir(raiz, 0); await esperar();
  await raiz.archivo(archivoCorreo("renuncia"));
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: "renuncia", recibida_en: "2026-09-24T11:00" });
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  assert.equal(resoluciones, 1);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="siguiente"/u);
  await raiz.enviar("siguiente", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174004" });
  assert.equal(siguientes, 1);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="siguiente"/u);
  cerrar();
});

test("aceptación restaurada permite propuesta solo con resolución y publicaciones", async () => {
  const raiz = raizPrueba(); let propuestas = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] }),
    consultarReciboRespuesta: async () => { throw sinRespuesta(); },
    registrarRespuestaRecibida: async (solicitud) => ({ ...justificante(solicitud),
      registrada_en: "2026-09-24T10:30:00Z" }),
    resolverLlamamiento: async () => ({ ...reciboResolucion("aceptacion"),
      resuelta_en: "2026-09-24T11:00:00Z" }),
    prepararPropuestaFormalizacion: async () => {
      propuestas += 1;
      return { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1",
        estado_local: "confirmado", propuesta_ref: "propuesta:ct140:001",
        recibo_local_ref: "recibo:propuesta:ct140:001", version_resultante: 7,
        confirmada_en: "2026-09-24T11:10:00Z" };
    },
  }, { contexto, fetchPublicaciones: async () => new Response(PUBLICACIONES_PROPUESTA,
    { status: 200, headers: { "Content-Type": "application/json" } }) });
  await esperar(); elegir(raiz, 0); await esperar();
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", { ...declaracion(), recibida_en: "2026-09-24T11:00" });
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  await raiz.enviar("propuesta", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174005" });
  assert.equal(propuestas, 1);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="propuesta"/u);
  cerrar();
});

for (const opcion of ["renuncia", "aceptacion"]) test(`cambiar de fila limpia ${opcion === "renuncia" ? "siguiente" : "propuesta"} de la fila anterior`, async () => {
  const raiz = raizPrueba(); let efectos = 0;
  const accion = opcion === "renuncia" ? "siguiente" : "propuesta";
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE,
      comunicaciones: [fila(1), fila(2)] }),
    consultarReciboRespuesta: async () => { throw sinRespuesta(); },
    registrarRespuestaRecibida: async (solicitud) => ({ ...justificante(solicitud),
      registrada_en: "2026-09-24T10:30:00Z" }),
    resolverLlamamiento: async () => ({ ...reciboResolucion(opcion),
      resuelta_en: "2026-09-24T11:00:00Z",
      ...(opcion === "renuncia" ? { intencion_siguiente: { referencia: "intencion:siguiente:001",
        estado_local: "pendiente", actualizada_en: "2026-09-24T11:00:00Z" } } : {}) }),
    continuarLlamamiento: async () => { efectos += 1; return continuacionConfirmada; },
    prepararPropuestaFormalizacion: async () => {
      efectos += 1;
      return { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1",
        estado_local: "confirmado", propuesta_ref: "propuesta:ct140:001",
        recibo_local_ref: "recibo:propuesta:ct140:001", version_resultante: 7,
        confirmada_en: "2026-09-24T11:10:00Z" };
    },
  }, { contexto, fetchPublicaciones: async () => new Response(PUBLICACIONES_PROPUESTA,
    { status: 200, headers: { "Content-Type": "application/json" } }) });
  await esperar(); elegir(raiz, 0); await esperar();
  await raiz.archivo(archivoCorreo(opcion));
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: opcion, recibida_en: "2026-09-24T11:00" });
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  assert.match(raiz.innerHTML, new RegExp(`data-ct-llamamiento-form="${accion}"`, "u"));
  elegir(raiz, 1);
  await esperar();
  assert.doesNotMatch(raiz.innerHTML, new RegExp(`data-ct-llamamiento-form="${accion}"`, "u"));
  await raiz.enviar(accion, { clave_idempotencia: accion === "siguiente"
    ? "123e4567-e89b-42d3-a456-426614174004" : "123e4567-e89b-42d3-a456-426614174005" });
  assert.equal(efectos, 0, "no reutiliza el efecto del llamamiento anterior");
  cerrar();
});

for (const ingles of [false, true]) test(`CT140 registrada y GET404 ${ingles ? "EN" : "ES"} oculta POST y explica recibo de otra comunicación`, async () => {
  const raiz = raizPrueba(); let escrituras = 0;
  const registrada = { ...fila(1), estado_respuesta: "registrada" };
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [registrada] }),
    consultarReciboRespuesta: async () => { throw sinRespuesta(); },
    registrarRespuestaRecibida: async () => { escrituras += 1; },
  }, { contexto, ...(ingles ? { mensajes: MENSAJES_LLAMAMIENTO_EN, locale: "en-GB" } : {}) });
  await esperar();
  assert.match(raiz.innerHTML, ingles ? /Reply recorded/u : /Respuesta anotada/u);
  elegir(raiz, 0);
  await esperar();
  assert.match(raiz.innerHTML, ingles
    ? /A reply is already recorded, but its receipt cannot be shown/u
    : /Ya hay una respuesta anotada, pero no se puede mostrar su ju/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 0);
  cerrar();
});

for (const estado of [403, 409]) test(`POST CT140 rechazado ${estado} bloquea lista anterior hasta nueva consulta`, async () => {
  const raiz = raizPrueba(); let escrituras = 0, listas = 0, recibos = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => {
      listas += 1;
      return { expediente_ref: EXPEDIENTE, comunicaciones: [fila(1), fila(2)] };
    },
    consultarReciboRespuesta: async () => { recibos += 1; throw sinRespuesta(); },
    registrarRespuestaRecibida: async () => {
      escrituras += 1;
      throw Object.assign(new Error(), { estado,
        codigo: estado === 403 ? "acceso_denegado" : "contenido_respuesta_en_conflicto",
        envelopeValido: true, resultadoIndeterminado: false });
    },
  }, { contexto });
  await esperar();
  elegir(raiz, 0);
  await esperar();
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 1);
  assert.match(raiz.innerHTML, /No se ha podido confirmar la respuesta\. Vuelva a cargar la l/u);
  assert.match(raiz.innerHTML, /No anote otra respuesta hasta que Informática revise la ante/u);
  assert.doesNotMatch(raiz.innerHTML, /Respuesta pendiente según la lista|data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 1);
  raiz.eventos.get("click")({ preventDefault() {}, target: { closest: (selector) =>
    selector === "[data-ct-comunicaciones-reintentar]"
      ? { dataset: { ctComunicacionesReintentar: "" } } : null } });
  await esperar();
  assert.equal(listas, 2);
  elegir(raiz, 1);
  await esperar();
  assert.equal(recibos, 2, "puede consultar otra comunicación después del rechazo");
  assert.match(raiz.innerHTML, /Hay una respuesta enviada que no se ha podido confirmar\. No/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(escrituras, 1, "no reenvía la declaración ni admite otra escritura");
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
    organizacion_ref: fila(1).organizacion_ref, expediente_ref: fila(1).expediente_ref,
    comunicacion_ref: fila(1).comunicacion_ref,
    respuesta: "aceptacion", justificante_ref: "justificante:ct140:001",
    recibo_ref: "recibo:respuesta:ct140:001", auditoria_ref: "auditoria:ct140:001",
    registrada_en: "2026-09-24T11:00:00Z", estado: "registrada_por_rrhh" });
  await esperar();
  assert.match(raiz.innerHTML, /Llamamiento 2 de 2 · 24\/09\/2026 · Sin respuesta/u);
  assert.doesNotMatch(raiz.innerHTML, /recibo:respuesta:ct140:001/u);
  cerrar();
});

test("CT140 deniega la lista sin filtrar filas y aborta al cambiar de expediente", async () => {
  const denegada = raizPrueba();
  const cerrarDenegada = montar(denegada, { consultarComunicacionesExpediente: async () => {
    throw Object.assign(new Error(), { estado: 403, codigo: "acceso_denegado", envelopeValido: true });
  } }, { contexto });
  await esperar();
  assert.match(denegada.innerHTML, /No tiene permiso para ver estos llamamientos\./u);
  assert.doesNotMatch(denegada.innerHTML, /data-ct-comunicacion-indice|data-ct-llamamiento-form=/u);
  cerrarDenegada();

  const raiz = raizPrueba(); let signal, resolver;
  const cerrar = montar(raiz, { consultarComunicacionesExpediente: (_consulta, opciones) => {
    signal = opciones.signal;
    return new Promise((resolve) => { resolver = resolve; });
  } }, { contexto });
  await esperar();
  assert.match(raiz.innerHTML, /Cargando los llamamientos…/u);
  cerrar.actualizarContexto({ expediente_ref: "expediente:otro:002", version_esperada: 6 });
  assert.equal(signal.aborted, true);
  resolver({ expediente_ref: EXPEDIENTE, comunicaciones: [fila(1)] });
  await esperar();
  assert.equal(raiz.innerHTML, "");
});
