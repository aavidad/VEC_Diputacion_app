import assert from "node:assert/strict";
import test from "node:test";
import { raizPrueba, montar, EXPEDIENTE } from "./formulario-llamamiento-pruebas.js?v=VEC_WEB_VERSION";
import { validarSolicitudResolucionPortal, validarReciboResolucionPortal } from "./contrato-llamamiento.js?v=VEC_WEB_VERSION";
import { validarResultadoBolsaCT } from "./contrato-resultado-bolsa.js?v=VEC_WEB_VERSION";
import { crearLlamamientoClienteHTTP, RUTAS_LLAMAMIENTO } from "./cliente-http-llamamiento.js?v=VEC_WEB_VERSION";
import { contextoLlamamientoDesdeEstado } from "./vista-expedientes-render.js?v=VEC_WEB_VERSION";

const LLAMAMIENTO = `llamamiento:${"a".repeat(64)}`;
const EMISION = `recibo:${LLAMAMIENTO}`;
const RESPUESTA = "recibo:respuesta:portal:001";
const CLAVE = "123e4567-e89b-42d3-a456-426614174001";
const PARTICIPACION = "participacion:portal:001";
const momento = "2026-10-20T09:00:00Z";
const persona = () => ({ participacion_ref: PARTICIPACION, respuesta: "acepta", modo: "firme",
  resolucion_portal_disponible: true,
  recibo_respuesta_ref: RESPUESTA, respondida_en: momento, justificante_ref: null,
  contacto_resultado: null, recibo_contacto_ref: null, contacto_en: null,
  situacion_actual: "disponible", recibo_situacion_ref: null, situacion_desde: null });
const resultado = (participaciones = [persona()]) => ({ vinculos: [{ bolsa_ref: "bolsa:portal:001",
  llamamiento_ref: LLAMAMIENTO, recibo_emision_ref: EMISION,
  recibo_vinculo_ref: "recibo:vinculo:portal:001", vinculado_en: momento, emitido_en: momento,
  participaciones }], emisiones_vinculables: [], siguiente_cursor: null, total_vinculos: 1,
  personas_solicitadas: 1, aceptaciones_firmes: 1 });
const solicitud = () => ({ fuente_respuesta: "portal", clave_idempotencia: CLAVE,
  expediente_ref: EXPEDIENTE, bolsa_ref: "bolsa:portal:001", llamamiento_ref: LLAMAMIENTO,
  recibo_emision_ref: EMISION, participacion_ref: PARTICIPACION, recibo_respuesta_ref: RESPUESTA,
  respuesta: "aceptacion", revision_respuesta_rrhh: true, revision_plazo_rrhh: true,
  criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906" });
const recibo = (estado_local = "confirmado") => ({ esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1",
  fuente_respuesta: "portal", respuesta: "aceptacion", estado_plazo: "revision_manual_portal",
  estado_local, resolucion_ref: "resolucion:portal:001", recibo_local_ref: "recibo:ct:portal:001",
  auditoria_ref: "auditoria:portal:001", version_resultante: 1, resuelta_en: "2026-10-20T10:00:00Z" });
const formulario = (revisionRespuesta = true, revisionPlazo = true) => ({
  dataset: { ctPortalResolucion: "0" },
  elements: { namedItem: (nombre) => ({ checked: nombre === "revision_respuesta_rrhh"
    ? revisionRespuesta : revisionPlazo }) },
  closest(selector) { return selector === "[data-ct-portal-resolucion]" ? this : null; },
});
const enviar = (raiz, form) => raiz.eventos.get("submit")({ target: form, preventDefault() {} });

test("contrato portal cerrado conserva origen, revisión y recibo de resolución", () => {
  assert.deepEqual(validarSolicitudResolucionPortal(solicitud()), solicitud());
  assert.equal(validarReciboResolucionPortal(recibo(), solicitud()).estado_plazo, "revision_manual_portal");
  assert.throws(() => validarSolicitudResolucionPortal({ ...solicitud(), comunicacion_ref: "comunicacion:falsa" }));
  assert.throws(() => validarSolicitudResolucionPortal({ ...solicitud(), revision_plazo_rrhh: false }));
  assert.throws(() => validarReciboResolucionPortal({ ...recibo(), estado_plazo: "vigente" }, solicitud()));
});

test("cliente HTTP envía el cuerpo canónico y distingue confirmación de recuperación", async () => {
  const peticiones = [];
  const cliente = crearLlamamientoClienteHTTP({ validarOpciones: (opciones) => opciones,
    ejecutar: async (peticion) => {
      peticiones.push(peticion);
      const status = peticiones.length === 1 ? 201 : 200;
      return peticion.validarRespuesta(recibo(status === 201 ? "confirmado" : "replay_confirmado"), status);
    } });
  assert.equal((await cliente.resolverLlamamiento(solicitud(), {})).estado_local, "confirmado");
  assert.equal((await cliente.resolverLlamamiento(solicitud(), {})).estado_local, "replay_confirmado");
  assert.deepEqual(peticiones.map((peticion) => peticion.ruta),
    [RUTAS_LLAMAMIENTO.resolucionLlamamiento, RUTAS_LLAMAMIENTO.resolucionLlamamiento]);
  assert.deepEqual(Object.keys(peticiones[0].entrada), Object.keys(solicitud()));
  assert.deepEqual(peticiones[0].entrada, peticiones[1].entrada);
});

test("aceptación firme singular de Mi Bolsa se confirma sin correo y reintenta con la misma petición", async () => {
  const raiz = raizPrueba(), peticiones = [];
  const cerrar = montar(raiz, { resolverLlamamiento: async (entrada) => {
    peticiones.push(entrada);
    if (peticiones.length === 1) throw new TypeError("respuesta perdida");
    return recibo("replay_confirmado");
  } }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
    numero_personas: 1, resultado_bolsa: resultado() }, generarClaveIdempotencia: () => CLAVE });
  assert.match(raiz.innerHTML, /Aceptación desde Mi Bolsa/u);
  assert.match(raiz.innerHTML, /recibo:respuesta:portal:001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
  assert.doesNotMatch(raiz.innerHTML.match(/<section class="ct-alta ct-llamamiento" aria-labelledby="ct-portal-resoluciones-titulo">[\s\S]*$/u)?.[0] ?? "", /\.eml|correo_sha256|correo_ref/u);
  await enviar(raiz, formulario(false));
  assert.equal(peticiones.length, 0);
  await enviar(raiz, formulario());
  assert.equal(peticiones.length, 1);
  assert.deepEqual(peticiones[0], solicitud());
  await enviar(raiz, formulario());
  assert.equal(peticiones.length, 2);
  assert.deepEqual(peticiones[1], peticiones[0]);
  assert.match(raiz.innerHTML, /Aceptación confirmada en el expediente/u);
  assert.match(raiz.innerHTML, /recibo:ct:portal:001/u);
  cerrar();
});

test("solo una participación firme habilita la acción; la ficha recupera recibos distintos", () => {
  const varias = resultado([{ ...persona(), resolucion_portal_disponible: false },
    { ...persona(), participacion_ref: "participacion:portal:002", resolucion_portal_disponible: false }]);
  const raiz = raizPrueba();
  const cerrar = montar(raiz, {}, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
    numero_personas: 1, resultado_bolsa: varias } });
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-resolucion/u);
  cerrar();
  const resuelta = resultado([{ ...persona(), resolucion_portal_disponible: false,
    resolucion_ct: { resolucion_ref: "resolucion:portal:001",
    recibo_ref: "recibo:ct:portal:001", resuelta_en: "2026-10-20T10:00:00.000000Z", respuesta: "aceptacion" } }]);
  assert.equal(validarResultadoBolsaCT(resuelta).vinculos[0].participaciones[0].resolucion_ct.recibo_ref,
    "recibo:ct:portal:001");
  const recuperada = raizPrueba();
  const desmontar = montar(recuperada, {}, { contexto: { expediente_ref: EXPEDIENTE,
    version_esperada: 6, numero_personas: 1, resultado_bolsa: resuelta } });
  assert.match(recuperada.innerHTML, /recibo:respuesta:portal:001/u);
  assert.match(recuperada.innerHTML, /recibo:ct:portal:001/u);
  assert.doesNotMatch(recuperada.innerHTML, /data-ct-portal-resolucion/u);
  desmontar();
  assert.throws(() => validarResultadoBolsaCT(resultado([{ ...persona(), modo: "propuesta_rrhh",
    resolucion_ct: resuelta.vinculos[0].participaciones[0].resolucion_ct }])));
});

test("una aceptación firme sin origen O6 nativo no ofrece resolución portal", () => {
  const raiz = raizPrueba();
  const cerrar = montar(raiz, {}, { contexto: { expediente_ref: EXPEDIENTE,
    version_esperada: 6, numero_personas: 1,
    resultado_bolsa: resultado([{ ...persona(), resolucion_portal_disponible: false }]) } });
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-resolucion/u);
  cerrar();
});

test("una denegación no se convierte en recibo ni se reintenta desde la ficha", async () => {
  const raiz = raizPrueba(); let intentos = 0;
  const cerrar = montar(raiz, { resolverLlamamiento: async () => {
    intentos += 1;
    throw Object.assign(new Error("denegado"), { estado: 403, envelopeValido: true,
      resultadoIndeterminado: false });
  } }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
    numero_personas: 1, resultado_bolsa: resultado() }, generarClaveIdempotencia: () => CLAVE });
  await enviar(raiz, formulario());
  assert.equal(intentos, 1);
  assert.match(raiz.innerHTML, /No tiene permiso para confirmar esta aceptación/u);
  assert.doesNotMatch(raiz.innerHTML, /recibo:ct:portal:001/u);
  await enviar(raiz, formulario());
  assert.equal(intentos, 1);
  cerrar();
});

test("la ficha entrega al formulario el resultado de Bolsa de su expediente y versión", () => {
  const bolsa = validarResultadoBolsaCT(resultado());
  const estado = { vista: "expediente", carga: "listo", ocupado: false,
    actualizacion_pendiente: false, resultado_indeterminado: false,
    expediente_ref: EXPEDIENTE,
    expediente: { demostracion: false, expediente_ref: EXPEDIENTE, version: 6,
      numero_personas: 1, resultado_bolsa: bolsa },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: EXPEDIENTE,
      version: 6, fase_clave: "llamamiento" }] } };
  assert.equal(contextoLlamamientoDesdeEstado(estado).resultado_bolsa, bolsa);
  estado.cuadro.expedientes[0].version = 7;
  assert.equal(contextoLlamamientoDesdeEstado(estado), null);
});
