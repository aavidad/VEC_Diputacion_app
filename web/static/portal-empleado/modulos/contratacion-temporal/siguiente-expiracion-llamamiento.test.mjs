import test from "node:test";
import assert from "node:assert/strict";
import { MENSAJES_LLAMAMIENTO_ES } from "./i18n-llamamiento.js";
import {
  CLAVE, EXPEDIENTE, recibo, raizPrueba, montar, seleccion, comunicacionRegistrada, justificante, revisionManual,
} from "./formulario-llamamiento-pruebas.js";

// Tras confirmar RRHH la expiración (sin respuesta en plazo), el paso ofrece
// abrir el siguiente llamamiento igual que tras una renuncia.
const CLAVE_CONTACTO = "123e4567-e89b-42d3-a456-426614174021";
const CLAVE_EXPIRACION = "123e4567-e89b-42d3-a456-426614174023";
const CLAVE_SIGUIENTE = "123e4567-e89b-42d3-a456-426614174024";
const PLAZO = Object.freeze({
  respuesta_hasta: "2026-09-06T22:00:00Z", ultimo_dia: "2026-09-06",
  politica_ref: "vec.bolsa.reglas:1:b05.plazo_respuesta", tratamiento_fuera_de_plazo: "exige_causa_justificada",
  confirmacion_expiracion: "rrhh", criterio_respuesta_ref: "vec.bolsa.reglas:1:b07.fuera_de_plazo",
  criterio_expiracion_ref: "vec.bolsa.reglas:1:b08.sin_respuesta_baja", regla_ejemplo: true,
});
const reciboContacto = (s) => ({ ...s, esquema: "vec.contratacion-temporal.evento-plazo-llamamiento.v1",
  evento_ref: "evento:contacto", recibo_ref: "recibo:contacto", auditoria_ref: "auditoria:contacto",
  registrado_en: "2026-09-05T09:00:00.5Z", estado: "registrado", plazo: PLAZO });
const reciboExpiracion = {
  esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1", respuesta: "expiracion_gobernada",
  estado_plazo: "expirado", estado_local: "confirmado", resolucion_ref: "resolucion:expiracion:021",
  recibo_local_ref: "recibo:expiracion:021", auditoria_ref: "auditoria:expiracion:021", version_resultante: 3,
  resuelta_en: "2026-09-07T08:00:00Z",
  intencion_siguiente: { referencia: "intencion:siguiente:expiracion:021", estado_local: "pendiente", actualizada_en: "2026-09-07T08:00:00Z" },
};
const continuacion = (s) => ({ esquema: "vec.contratacion-temporal.continuacion-llamamiento.v1",
  organizacion_ref: s.organizacion_ref, expediente_ref: s.expediente_ref, resolucion_ref: s.resolucion_ref,
  intencion_ref: s.intencion_ref, llamamiento_anterior_ref: recibo.llamamiento_ref, llamamiento_ref: "llamamiento:siguiente:021",
  version_llamamiento: 1, recibo_bolsa_ref: "recibo:bolsa:021", recibo_ref: "recibo:ct:021", auditoria_ref: "auditoria:siguiente:021",
  confirmada_en: "2026-09-07T08:05:00.123456Z", estado_intencion: "despachada", estado_local: "confirmado" });

async function confirmarExpiracion(raiz, continuar) {
  let ahora = Date.parse("2026-09-06T12:00:00Z");
  const cerrar = montar(raiz, {
    registrarComunicacionLlamamiento: async () => comunicacionRegistrada,
    registrarRespuestaRecibida: async (s) => justificante(s),
    registrarEventoPlazoLlamamiento: async (s) => reciboContacto(s),
    resolverLlamamiento: async () => reciboExpiracion,
    continuarLlamamiento: continuar,
  }, { reloj: () => ahora, confirmarOperacion: () => true });
  await raiz.enviar("seleccion", { ...seleccion(), version_esperada: 6 });
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  await raiz.enviar("contacto", { clave_idempotencia: CLAVE_CONTACTO, instante_en: "2026-09-05T08:30", prueba_ref: "prueba:llamada" });
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="siguiente"/u, "sin confirmación no hay siguiente");
  ahora = Date.parse("2026-09-07T08:00:00Z");
  cerrar.revisarPlazo();
  await raiz.enviar("expiracion", { clave_idempotencia: CLAVE_EXPIRACION, ...revisionManual });
  return cerrar;
}

test("tras confirmar la expiración se abre el siguiente llamamiento como tras una renuncia", async () => {
  const raiz = raizPrueba(), enviadas = [];
  const cerrar = await confirmarExpiracion(raiz, async (s) => { enviadas.push(s); return continuacion(s); });
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="siguiente"/u);
  assert.match(raiz.innerHTML, new RegExp(MENSAJES_LLAMAMIENTO_ES.llamamiento_resultado_siguiente_continuacion.slice(0, 30), "u"));
  await raiz.enviar("siguiente", { clave_idempotencia: CLAVE_EXPIRACION });
  assert.equal(enviadas.length, 0, "la continuación necesita su propia clave");
  await raiz.enviar("siguiente", { clave_idempotencia: CLAVE_SIGUIENTE, resolucion_ref: "resolucion:inventada",
    intencion_ref: "intencion:inventada" });
  assert.deepEqual(enviadas, [{ clave_idempotencia: CLAVE_SIGUIENTE, organizacion_ref: recibo.organizacion_ref,
    expediente_ref: EXPEDIENTE, resolucion_ref: reciboExpiracion.resolucion_ref,
    intencion_ref: reciboExpiracion.intencion_siguiente.referencia }]);
  assert.match(raiz.innerHTML, /llamamiento:siguiente:021/u);
  assert.match(raiz.innerHTML, new RegExp(MENSAJES_LLAMAMIENTO_ES.llamamiento_siguiente_recibo, "u"));
  // El aviso al sucesor tras una expiración queda fuera de este paso.
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion_siguiente"/u);
  cerrar();
});

test("un recibo de continuación de otro llamamiento anterior se rechaza", async () => {
  const raiz = raizPrueba();
  const cerrar = await confirmarExpiracion(raiz, async (s) => ({ ...continuacion(s), llamamiento_anterior_ref: "llamamiento:ajeno" }));
  await raiz.enviar("siguiente", { clave_idempotencia: CLAVE_SIGUIENTE });
  assert.doesNotMatch(raiz.innerHTML, /llamamiento:siguiente:021/u);
  cerrar();
});
