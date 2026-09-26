import test from "node:test";
import assert from "node:assert/strict";
import {
  validarSolicitudEventoPlazo, validarReciboEventoPlazo, validarSolicitudResolucionLlamamiento,
  validarReciboResolucionLlamamiento, situacionPlazoRespuesta, respuestaFueraDePlazo,
} from "./contrato-llamamiento.js";
import { MENSAJES_LLAMAMIENTO_ES } from "./i18n-llamamiento.js";
import {
  CLAVE, EXPEDIENTE, recibo, raizPrueba, montar, seleccion, comunicacionRegistrada, declaracion,
  justificante, archivoCorreo, revisionManual,
} from "./formulario-llamamiento-pruebas.js";

const CLAVE_CONTACTO = "123e4567-e89b-42d3-a456-426614174011";
const CLAVE_CAUSA = "123e4567-e89b-42d3-a456-426614174012";
const CLAVE_EXPIRACION = "123e4567-e89b-42d3-a456-426614174013";
const PLAZO = Object.freeze({
  respuesta_hasta: "2026-09-06T22:00:00Z", ultimo_dia: "2026-09-06",
  politica_ref: "vec.bolsa.reglas:1:b05.plazo_respuesta", tratamiento_fuera_de_plazo: "exige_causa_justificada",
  confirmacion_expiracion: "rrhh", criterio_respuesta_ref: "vec.bolsa.reglas:1:b07.fuera_de_plazo",
  criterio_expiracion_ref: "vec.bolsa.reglas:1:b08.sin_respuesta_baja", regla_ejemplo: true,
});
const reciboEvento = (s, plazo = PLAZO) => ({ ...s, esquema: "vec.contratacion-temporal.evento-plazo-llamamiento.v1",
  evento_ref: `evento:${s.tipo}`, recibo_ref: `recibo:${s.tipo}`, auditoria_ref: `auditoria:${s.tipo}`,
  registrado_en: "2026-09-07T09:00:00.5Z", estado: "registrado", ...(s.tipo === "contacto_efectivo" ? { plazo } : {}) });
const reciboExpiracion = {
  esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1", respuesta: "expiracion_gobernada",
  estado_plazo: "expirado", estado_local: "confirmado", resolucion_ref: "resolucion:expiracion:001",
  recibo_local_ref: "recibo:expiracion:001", auditoria_ref: "auditoria:expiracion:001", version_resultante: 3,
  resuelta_en: "2026-09-07T08:00:00Z",
  intencion_siguiente: { referencia: "intencion:siguiente:expiracion", estado_local: "pendiente", actualizada_en: "2026-09-07T08:00:00Z" },
};
const antes = Date.parse("2026-09-06T12:00:00Z"), despues = Date.parse("2026-09-07T08:00:00Z");

async function abrirPlazo(raiz, cliente = {}, reloj = () => antes, extras = {}) {
  const cerrar = montar(raiz, {
    registrarComunicacionLlamamiento: async () => comunicacionRegistrada,
    registrarRespuestaRecibida: async (s) => justificante(s),
    registrarEventoPlazoLlamamiento: async (s) => reciboEvento(s), ...cliente,
  }, { reloj, ...extras });
  await raiz.enviar("seleccion", { ...seleccion(), version_esperada: 6 });
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  return cerrar;
}

test("contrato del plazo: el vencimiento y las reglas solo vienen del servidor", () => {
  const s = validarSolicitudEventoPlazo({ clave_idempotencia: CLAVE_CONTACTO, organizacion_ref: "organizacion:x",
    expediente_ref: EXPEDIENTE, llamamiento_ref: "llamamiento:x", comunicacion_ref: "comunicacion:x",
    version_comunicacion_esperada: 2, tipo: "contacto_efectivo", instante_en: "2026-09-05T08:30:00Z", prueba_ref: "prueba:llamada" });
  assert.equal(validarReciboEventoPlazo(reciboEvento(s), s).plazo.respuesta_hasta, PLAZO.respuesta_hasta);
  for (const cambio of [{ tipo: "correo" }, { version_comunicacion_esperada: 1 }, { instante_en: "ayer" },
    { respuesta_hasta: "2026-09-06T22:00:00Z" }, { prueba_ref: "" }]) {
    assert.throws(() => validarSolicitudEventoPlazo({ ...s, ...cambio }), TypeError);
  }
  for (const plazo of [{ ...PLAZO, respuesta_hasta: "2026-09-05T08:00:00Z" }, { ...PLAZO, tratamiento_fuera_de_plazo: "libre" },
    { ...PLAZO, confirmacion_expiracion: "automatica" }, { ...PLAZO, extra: 1 }]) {
    assert.throws(() => validarReciboEventoPlazo(reciboEvento(s, plazo), s), TypeError);
  }
  const causa = { ...s, tipo: "causa_justificada" };
  assert.throws(() => validarReciboEventoPlazo({ ...reciboEvento(causa), plazo: PLAZO }, causa), TypeError);
  assert.equal(situacionPlazoRespuesta(PLAZO, Date.parse(PLAZO.respuesta_hasta) - 1), "en_plazo");
  assert.equal(situacionPlazoRespuesta(PLAZO, Date.parse(PLAZO.respuesta_hasta)), "vencido");
  assert.equal(respuestaFueraDePlazo(PLAZO, "2026-09-06T21:59:59.999999Z"), false);
  assert.equal(respuestaFueraDePlazo(PLAZO, "2026-09-06T22:00:00Z"), true);
});

test("la expiración no lleva justificante y su recibo exige plazo expirado e intención pendiente", () => {
  const s = validarSolicitudResolucionLlamamiento({ clave_idempotencia: CLAVE_EXPIRACION, organizacion_ref: "organizacion:x",
    expediente_ref: EXPEDIENTE, llamamiento_ref: "llamamiento:x", comunicacion_ref: "comunicacion:x", version_esperada: 2,
    respuesta: "expiracion_gobernada", prueba_respuesta_ref: "", ...revisionManual, criterio_validacion_ref: PLAZO.criterio_expiracion_ref });
  assert.equal(s.prueba_respuesta_ref, "");
  assert.throws(() => validarSolicitudResolucionLlamamiento({ ...s, prueba_respuesta_ref: "justificante:x" }), TypeError);
  assert.throws(() => validarSolicitudResolucionLlamamiento({ ...s, revision_plazo_rrhh: false }), TypeError);
  assert.equal(validarReciboResolucionLlamamiento(reciboExpiracion, s).estado_plazo, "expirado");
  assert.throws(() => validarReciboResolucionLlamamiento({ ...reciboExpiracion, estado_plazo: "vigente" }, s), TypeError);
  const { intencion_siguiente: _, ...sinIntencion } = reciboExpiracion;
  assert.throws(() => validarReciboResolucionLlamamiento(sinIntencion, s), TypeError);
});

test("contacto efectivo muestra vencimiento y estado sin rotular la regla; sin texto de ayuda", async () => {
  const raiz = raizPrueba(), enviadas = [];
  const cerrar = await abrirPlazo(raiz, { registrarEventoPlazoLlamamiento: async (s) => { enviadas.push(s); return reciboEvento(s); } });
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="contacto"/u);
  await raiz.enviar("contacto", { clave_idempotencia: CLAVE, instante_en: "2026-09-05T08:30", prueba_ref: "prueba:llamada" });
  assert.equal(enviadas.length, 0, "necesita una clave propia");
  await raiz.enviar("contacto", { clave_idempotencia: CLAVE_CONTACTO, instante_en: "2026-09-05T08:30",
    prueba_ref: "prueba:llamada", respuesta_hasta: "2027-01-01T00:00:00Z", tipo: "causa_justificada" });
  assert.deepEqual(enviadas, [{ clave_idempotencia: CLAVE_CONTACTO, organizacion_ref: recibo.organizacion_ref,
    expediente_ref: EXPEDIENTE, llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: comunicacionRegistrada.comunicacion_ref,
    version_comunicacion_esperada: 2, tipo: "contacto_efectivo", instante_en: "2026-09-05T08:30:00Z", prueba_ref: "prueba:llamada" }]);
  const plazo = raiz.innerHTML.match(/<section class="ct-llamamiento-paso ct-llamamiento-plazo"[\s\S]*?<\/section>\s*<\/section>|<section class="ct-llamamiento-paso ct-llamamiento-plazo"[\s\S]*$/u)[0];
  assert.match(plazo, /datetime="2026-09-06T22:00:00Z"/u);
  assert.match(plazo, /data-ct-llamamiento-plazo-situacion="en_plazo"/u);
  assert.match(plazo, /En plazo/u);
  assert.doesNotMatch(plazo, /Regla de ejemplo/u);
  assert.doesNotMatch(plazo, /data-ct-llamamiento-propuesta-expiracion/u);
  assert.doesNotMatch(plazo, /class="ct-ayuda"|id="ct-llamamiento-contacto-ayuda"/u);
  assert.equal(raiz.foco.at(-1), '[data-ct-llamamiento-recibo="contacto"]');
  cerrar();
});

test("al vencer sin respuesta VEC propone y solo la confirmación de RRHH la registra", async () => {
  const raiz = raizPrueba(), resoluciones = [], confirmaciones = [];
  let ahora = antes;
  const cerrar = await abrirPlazo(raiz, { resolverLlamamiento: async (s) => { resoluciones.push(s); return reciboExpiracion; } },
    () => ahora, { confirmarOperacion: (d) => { confirmaciones.push(d); return true; } });
  await raiz.enviar("contacto", { clave_idempotencia: CLAVE_CONTACTO, instante_en: "2026-09-05T08:30", prueba_ref: "prueba:llamada" });
  await raiz.enviar("expiracion", { clave_idempotencia: CLAVE_EXPIRACION, ...revisionManual });
  assert.equal(resoluciones.length, 0, "no se propone antes del vencimiento");
  ahora = despues;
  cerrar.revisarPlazo();
  assert.match(raiz.innerHTML, /data-ct-llamamiento-plazo-situacion="vencido"/u);
  assert.match(raiz.innerHTML, /No aceptación por falta de respuesta y llamamiento al siguiente candidato/u);
  await raiz.enviar("expiracion", { clave_idempotencia: CLAVE_EXPIRACION, revision_respuesta_rrhh: true, revision_plazo_rrhh: false });
  assert.equal(resoluciones.length, 0, "RRHH debe marcar ambas comprobaciones");
  await raiz.enviar("expiracion", { clave_idempotencia: CLAVE_EXPIRACION, ...revisionManual, respuesta: "aceptacion",
    prueba_respuesta_ref: "justificante:inventado", criterio_validacion_ref: "politica:inventada" });
  assert.deepEqual(resoluciones, [{ clave_idempotencia: CLAVE_EXPIRACION, organizacion_ref: recibo.organizacion_ref,
    expediente_ref: EXPEDIENTE, llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: comunicacionRegistrada.comunicacion_ref,
    version_esperada: 2, respuesta: "expiracion_gobernada", prueba_respuesta_ref: "", ...revisionManual,
    criterio_validacion_ref: PLAZO.criterio_expiracion_ref }]);
  assert.match(confirmaciones.at(-1).advertencia, /vec\.bolsa\.reglas:1:b08\.sin_respuesta_baja/u);
  assert.match(raiz.innerHTML, /Propuesta confirmada: no aceptación registrada/u);
  assert.match(raiz.innerHTML, /Expirado/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  cerrar();
});

test("respuesta fuera de plazo exige la causa acreditada antes de resolver", async () => {
  const raiz = raizPrueba(), eventos = [], resoluciones = [];
  const cerrar = await abrirPlazo(raiz, {
    registrarEventoPlazoLlamamiento: async (s) => { eventos.push(s); return reciboEvento(s); },
    registrarRespuestaRecibida: async (s) => ({ ...justificante(s), registrada_en: "2026-09-07T07:10:00Z" }),
    resolverLlamamiento: async (s) => { resoluciones.push(s); return { esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1",
      respuesta: "aceptacion", estado_plazo: "vigente", estado_local: "confirmado", resolucion_ref: "resolucion:x",
      recibo_local_ref: "recibo:x", auditoria_ref: "auditoria:x", version_resultante: 3, resuelta_en: "2026-09-07T09:00:00Z" }; },
  }, () => despues);
  await raiz.enviar("contacto", { clave_idempotencia: CLAVE_CONTACTO, instante_en: "2026-09-05T08:30", prueba_ref: "prueba:llamada" });
  await raiz.archivo(archivoCorreo("aceptacion"));
  await raiz.enviar("respuesta", { ...declaracion(), recibida_en: "2026-09-07T07:00" });
  assert.match(raiz.innerHTML, /Respuesta recibida fuera de plazo/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="causa"/u);
  await raiz.enviar("resolucion", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174003", ...revisionManual });
  assert.equal(resoluciones.length, 0, "sin causa acreditada no se resuelve");
  await raiz.enviar("causa", { clave_idempotencia: CLAVE_CAUSA, instante_en: "2026-09-07T07:30", prueba_ref: "prueba:causa" });
  assert.equal(eventos.at(-1).tipo, "causa_justificada");
  assert.match(raiz.innerHTML, /Causa justificada acreditada/u);
  await raiz.enviar("resolucion", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174003", ...revisionManual });
  assert.equal(resoluciones.length, 1);
  assert.equal(resoluciones[0].criterio_validacion_ref, PLAZO.criterio_respuesta_ref);
  cerrar();
});

test("textos del plazo por i18n", () => {
  for (const clave of ["llamamiento_plazo_titulo", "llamamiento_plazo_en_plazo", "llamamiento_plazo_vencido",
    "llamamiento_plazo_expirado", "llamamiento_propuesta_expiracion", "llamamiento_confirmar_expiracion",
    "llamamiento_contacto_recibo", "llamamiento_causa_recibo", "llamamiento_expiracion_recibo"]) {
    assert.equal(typeof MENSAJES_LLAMAMIENTO_ES[clave], "string", clave);
  }
});
