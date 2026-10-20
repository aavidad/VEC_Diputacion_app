import assert from "node:assert/strict";
import test from "node:test";
import { raizPrueba, montar, EXPEDIENTE } from "./formulario-llamamiento-pruebas.js?v=VEC_WEB_VERSION";
import { validarResultadoBolsaCT } from "./contrato-resultado-bolsa.js?v=VEC_WEB_VERSION";
import { validarSolicitudResolucionPortal } from "./contrato-llamamiento.js?v=VEC_WEB_VERSION";
import { crearLlamamientoClienteHTTP, RUTAS_LLAMAMIENTO } from "./cliente-http-llamamiento.js?v=VEC_WEB_VERSION";

const LLAMAMIENTO = `llamamiento:${"a".repeat(64)}`;
const CLAVE = "123e4567-e89b-42d3-a456-426614174001";
const respuesta = "recibo:respuesta:portal:001";
const persona = (disponible = true) => ({ participacion_ref: "participacion:portal:001",
  respuesta: "acepta", modo: "firme", recibo_respuesta_ref: respuesta,
  respondida_en: "2026-10-20T09:05:00Z", justificante_ref: null,
  contacto_resultado: null, recibo_contacto_ref: null, contacto_en: null,
  situacion_actual: "disponible", recibo_situacion_ref: null, situacion_desde: null,
  origen_ct_confirmado: true, resolucion_portal_disponible: disponible });
const resultado = (participaciones = [persona()]) => ({ vinculos: [{ bolsa_ref: "bolsa:portal:001",
  llamamiento_ref: LLAMAMIENTO, recibo_emision_ref: `recibo:${LLAMAMIENTO}`,
  recibo_vinculo_ref: "recibo:vinculo:portal:001", vinculado_en: "2026-10-20T09:00:00Z",
  emitido_en: "2026-10-20T09:00:00Z", participaciones }], emisiones_vinculables: [],
  siguiente_cursor: null, total_vinculos: 1, personas_solicitadas: 1, aceptaciones_firmes: 1 });
const contexto = (bolsa) => ({ expediente_ref: EXPEDIENTE, version_esperada: 6,
  numero_personas: 1, resultado_bolsa: bolsa });
const solicitud = () => ({ fuente_respuesta: "portal", clave_idempotencia: CLAVE,
  expediente_ref: EXPEDIENTE, bolsa_ref: "bolsa:portal:001", llamamiento_ref: LLAMAMIENTO,
  recibo_emision_ref: `recibo:${LLAMAMIENTO}`, participacion_ref: "participacion:portal:001",
  recibo_respuesta_ref: respuesta, respuesta: "aceptacion", revision_respuesta_rrhh: true,
  revision_plazo_rrhh: true, criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906" });
const recibo = (estado_local = "confirmado") => ({ esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1",
  fuente_respuesta: "portal", respuesta: "aceptacion", estado_plazo: "revision_manual_portal",
  estado_local, resolucion_ref: "resolucion:portal:001", recibo_local_ref: "recibo:ct:portal:001",
  auditoria_ref: "auditoria:portal:001", version_resultante: 1,
  resuelta_en: "2026-10-20T10:00:00Z" });
const formulario = (respuestaRevisada = true, plazoRevisado = true) => ({
  dataset: { ctPortalResolucion: "0" },
  elements: { namedItem: (nombre) => ({ checked: nombre === "revision_respuesta_rrhh"
    ? respuestaRevisada : plazoRevisado }) },
  closest(selector) { return selector === "[data-ct-portal-resolucion]" ? this : null; },
});
const enviar = (raiz, form) => raiz.eventos.get("submit")({ target: form, preventDefault() {} });

test("O6 ordinaria conserva lectura sin ofrecer resolución nativa", () => {
  const raiz = raizPrueba();
  const cerrar = montar(raiz, {}, { contexto: contexto(resultado([persona(false)])) });
  assert.match(raiz.innerHTML, /Iniciado desde este expediente/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-resolucion/u);
  cerrar();
  assert.throws(() => validarResultadoBolsaCT(resultado([{ ...persona(),
    origen_ct_confirmado: false }])));
});

test("solo elegibilidad nativa singular permite revisar y confirmar sin correo", async () => {
  const raiz = raizPrueba(), peticiones = [];
  const cerrar = montar(raiz, { resolverLlamamiento: async (entrada) => {
    peticiones.push(entrada);
    if (peticiones.length === 1) throw new TypeError("respuesta perdida");
    return recibo("replay_confirmado");
  } }, { contexto: contexto(resultado()), generarClaveIdempotencia: () => CLAVE });
  assert.match(raiz.innerHTML, /data-ct-portal-resolucion="0"/u);
  assert.match(raiz.innerHTML, /recibo:respuesta:portal:001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
  await enviar(raiz, formulario(false));
  assert.equal(peticiones.length, 0);
  await enviar(raiz, formulario());
  assert.deepEqual(peticiones, [solicitud()]);
  await enviar(raiz, formulario());
  assert.deepEqual(peticiones, [solicitud(), solicitud()]);
  assert.match(raiz.innerHTML, /Aceptación confirmada en el expediente/u);
  assert.match(raiz.innerHTML, /recibo:ct:portal:001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-resolucion=/u);
  cerrar();
});

test("HTTP mantiene cuerpo canónico y separa 201 de replay 200", async () => {
  const peticiones = [];
  const cliente = crearLlamamientoClienteHTTP({ validarOpciones: (opciones) => opciones,
    ejecutar: async (peticion) => {
      peticiones.push(peticion);
      const status = peticiones.length === 1 ? 201 : 200;
      return peticion.validarRespuesta(recibo(status === 201 ? "confirmado" : "replay_confirmado"), status);
    } });
  assert.deepEqual(validarSolicitudResolucionPortal(solicitud()), solicitud());
  assert.equal((await cliente.resolverLlamamiento(solicitud(), {})).estado_local, "confirmado");
  assert.equal((await cliente.resolverLlamamiento(solicitud(), {})).estado_local, "replay_confirmado");
  assert.equal(peticiones[0].ruta, RUTAS_LLAMAMIENTO.resolucionLlamamiento);
  assert.deepEqual(Object.keys(peticiones[0].entrada), Object.keys(solicitud()));
  assert.throws(() => validarSolicitudResolucionPortal({ ...solicitud(), comunicacion_ref: "comunicacion:falsa" }));
});

test("tras recargar queda la aceptación de Bolsa sin ofrecer otra resolución", () => {
  const resuelta = resultado([persona(false)]);
  validarResultadoBolsaCT(resuelta);
  const raiz = raizPrueba();
  const cerrar = montar(raiz, {}, { contexto: contexto(resuelta) });
  assert.match(raiz.innerHTML, /recibo:respuesta:portal:001/u);
  assert.doesNotMatch(raiz.innerHTML, /recibo:ct:portal:001|No se ha podido comprobar si se guardó/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-resolucion=/u);
  cerrar();
});

test("una denegación bloquea otro envío y no muestra recibo CT", async () => {
  const raiz = raizPrueba(); let intentos = 0;
  const cerrar = montar(raiz, { resolverLlamamiento: async () => {
    intentos += 1;
    throw Object.assign(new Error("denegado"), { estado: 403, envelopeValido: true,
      resultadoIndeterminado: false });
  } }, { contexto: contexto(resultado()), generarClaveIdempotencia: () => CLAVE });
  await enviar(raiz, formulario());
  assert.equal(intentos, 1);
  assert.match(raiz.innerHTML, /No tiene permiso para confirmar esta aceptación/u);
  assert.doesNotMatch(raiz.innerHTML, /recibo:ct:portal:001/u);
  await enviar(raiz, formulario());
  assert.equal(intentos, 1);
  cerrar();
});

test("CT140 pendiente conserva su estado sin abrir un formulario legado", async () => {
  const raiz = raizPrueba(); let liberar;
  const pendiente = new Promise((resolver) => { liberar = resolver; });
  const cerrar = montar(raiz, { consultarComunicacionesExpediente: async () => pendiente },
    { contexto: contexto(resultado()) });
  await new Promise(setImmediate);
  assert.match(raiz.innerHTML, /Cargando los llamamientos/u);
  assert.match(raiz.innerHTML, /data-ct-portal-resolucion="0"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  cerrar();
  liberar({ expediente_ref: EXPEDIENTE, comunicaciones: [] });
});
