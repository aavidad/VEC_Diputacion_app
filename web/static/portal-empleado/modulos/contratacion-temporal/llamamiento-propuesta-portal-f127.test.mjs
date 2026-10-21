import assert from "node:assert/strict";
import test from "node:test";
import { raizPrueba, montar, EXPEDIENTE, PUBLICACIONES_PROPUESTA } from "./formulario-llamamiento-pruebas.js?v=VEC_WEB_VERSION";
import { validarResultadoBolsaCT } from "./contrato-resultado-bolsa.js?v=VEC_WEB_VERSION";
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


const publicaciones = async () => new Response(PUBLICACIONES_PROPUESTA,
  { status: 200, headers: { "Content-Type": "application/json" } });
const proyeccion = () => ({ resolucion_ref: recibo().resolucion_ref,
  recibo_ref: recibo().recibo_local_ref, resuelta_en: recibo().resuelta_en, respuesta: "aceptacion" });
const recuperado = () => resultado([{ ...persona(false), resolucion_ct: proyeccion() }]);
const esperar = () => new Promise(setImmediate);

for (const replay of [false, true]) test(`resolución portal${replay ? " repetida" : ""} prepara una propuesta explícita con el recibo real`, async () => {
  const raiz = raizPrueba(), peticiones = [];
  const cerrar = montar(raiz, { resolverLlamamiento: async () => recibo(replay ? "replay_confirmado" : "confirmado"),
    prepararPropuestaFormalizacion: async (entrada) => { peticiones.push(entrada); throw new Error("respuesta perdida"); } },
  { contexto: contexto(resultado()), fetchPublicaciones: publicaciones });
  await enviar(raiz, formulario());
  assert.equal(peticiones.length, 0);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-correo|data-ct-llamamiento-form="seleccion"/u);
  await raiz.enviar("propuesta", { clave_idempotencia: CLAVE });
  assert.equal(peticiones.length, 1);
  const entrada = peticiones[0];
  assert.equal(entrada.expediente_ref, EXPEDIENTE);
  assert.equal(entrada.llamamiento_ref, LLAMAMIENTO);
  assert.equal(entrada.version_esperada, 6);
  assert.equal(entrada.resolucion_llamamiento_aceptada_ref, recibo().resolucion_ref);
  assert.equal(entrada.recibo_resolucion_aceptada_ref, recibo().recibo_local_ref);
  for (const campo of ["tipo_formalizacion", "plantilla", "politica_firma", "plan_firma"]) {
    assert.ok(Object.isFrozen(entrada[campo]));
    assert.match(entrada[campo].huella_sha256, /^[0-9a-f]{64}$/u);
  }
  await raiz.enviar("propuesta", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174099" });
  assert.equal(peticiones.length, 2);
  assert.strictEqual(peticiones[0], peticiones[1]);
  cerrar();
});

test("recargar recupera la pareja real sin repetir resolución ni propuesta", async () => {
  const raiz = raizPrueba(); let actos = 0;
  const cerrar = montar(raiz, { resolverLlamamiento: async () => { actos++; },
    prepararPropuestaFormalizacion: async () => { actos++; } },
  { contexto: contexto(recuperado()), fetchPublicaciones: publicaciones });
  await esperar();
  assert.equal(actos, 0);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  assert.match(raiz.innerHTML, /recibo:ct:portal:001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-resolucion=/u);
  cerrar();
});

test("versión avanzada no prepara otra propuesta y un contexto ajeno aborta publicaciones", async () => {
  let lecturas = 0, liberar, signal;
  const raiz = raizPrueba();
  const cerrar = montar(raiz, { prepararPropuestaFormalizacion: async () => assert.fail("sin envío") },
    { contexto: { ...contexto(recuperado()), version_esperada: 7 }, fetchPublicaciones: async () => { lecturas++; } });
  assert.equal(lecturas, 0);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  cerrar();
  const otra = raizPrueba();
  const desmontar = montar(otra, { prepararPropuestaFormalizacion: async () => {} }, {
    contexto: contexto(recuperado()), fetchPublicaciones: async (_, opciones) => {
      signal = opciones.signal;
      return new Promise((resolver) => { liberar = resolver; });
    },
  });
  assert.equal(desmontar.actualizarContexto({ ...contexto(recuperado()), expediente_ref: "expediente:otro" }), false);
  assert.equal(signal.aborted, true);
  liberar(await publicaciones());
  await esperar();
  assert.equal(otra.innerHTML, "");
});

test("la proyección rechaza datos extra, renuncia y disponibilidad contradictoria", () => {
  for (const cambio of [{ secreto: "no" }, { respuesta: "renuncia" }]) {
    assert.throws(() => validarResultadoBolsaCT(resultado([{ ...persona(false), resolucion_ct: { ...proyeccion(), ...cambio } }])));
  }
  assert.throws(() => validarResultadoBolsaCT(resultado([{ ...persona(), resolucion_ct: proyeccion() }])));
});

test("avanzar el mismo expediente cancela la preparación pendiente sin otra propuesta", async () => {
  const raiz = raizPrueba(); let signal, liberar;
  const cerrar = montar(raiz, { prepararPropuestaFormalizacion: async () => assert.fail("sin acto") }, {
    contexto: contexto(recuperado()), fetchPublicaciones: async (_, opciones) => {
      signal = opciones.signal;
      return new Promise((resolver) => { liberar = resolver; });
    },
  });
  assert.equal(cerrar.actualizarContexto({ ...contexto(recuperado()), version_esperada: 7 }), true);
  assert.equal(signal.aborted, true);
  liberar(await publicaciones());
  await esperar();
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  cerrar();
});

const comunicacionCT = () => ({ organizacion_ref: "organizacion:ct:001", expediente_ref: EXPEDIENTE,
  llamamiento_ref: "llamamiento:otra-comunicacion", comunicacion_ref: "comunicacion:ct:001", version: 2,
  estado: "registrada_localmente", registrada_en: "2026-10-20T09:00:00Z",
  recibo_comunicacion_ref: "recibo:comunicacion:ct:001", antecedente_tipo: "seleccion_confirmada",
  recibo_antecedente_ref: `recibo:${LLAMAMIENTO}`, estado_respuesta: "sin_respuesta" });
const elegirComunicacion = (raiz) => raiz.eventos.get("click")({ preventDefault() {}, target: {
  closest: (selector) => selector === "[data-ct-comunicacion-indice]"
    ? { dataset: { ctComunicacionIndice: "0" } } : null,
} });

for (const seleccionar of [false, true]) for (const rechazar of [false, true]) test(`propuesta portal con lista CT${seleccionar ? " seleccionada" : " sin seleccionar"}: ${rechazar ? "rechazo" : "confirmación"} del servidor`, async () => {
  const raiz = raizPrueba(), peticiones = [];
  const cerrar = montar(raiz, {
    resolverLlamamiento: async () => recibo(),
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [comunicacionCT()] }),
    prepararPropuestaFormalizacion: async (s) => {
      peticiones.push(s);
      if (rechazar) throw Object.assign(new Error("sin permiso"), { estado: 403, codigo: "acceso_denegado",
        envelopeValido: true, resultadoIndeterminado: false });
      return { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1", estado_local: "confirmado",
        propuesta_ref: "propuesta:portal:001", recibo_local_ref: "recibo:propuesta:portal:001",
        version_resultante: 7, confirmada_en: "2026-10-20T10:05:00Z" };
    },
  }, { contexto: contexto(resultado()), fetchPublicaciones: publicaciones });
  await esperar();
  await enviar(raiz, formulario());
  if (seleccionar) { elegirComunicacion(raiz); await esperar(); }
  assert.equal((raiz.innerHTML.match(/data-ct-llamamiento-form="propuesta"/gu) ?? []).length, 1);
  assert.equal(peticiones.length, 0);
  await raiz.enviar("propuesta", { clave_idempotencia: CLAVE,
    resolucion_llamamiento_aceptada_ref: "resolucion:ajena", recibo_resolucion_aceptada_ref: "recibo:ajeno" });
  assert.equal(peticiones.length, 1,
    raiz.innerHTML.match(/data-ct-llamamiento-estado="propuesta"[\s\S]*?<\/div>/u)?.[0]);
  assert.equal(peticiones[0].resolucion_llamamiento_aceptada_ref, recibo().resolucion_ref);
  assert.equal(peticiones[0].recibo_resolucion_aceptada_ref, recibo().recibo_local_ref);
  if (rechazar) {
    assert.match(raiz.innerHTML, /No se ha guardado\. Revise los datos y vuelva a intentarlo\./u);
    assert.doesNotMatch(raiz.innerHTML, /Propuesta guardada/u);
  } else {
    assert.match(raiz.innerHTML, /Propuesta guardada/u);
    await raiz.enviar("propuesta", { clave_idempotencia: "123e4567-e89b-42d3-a456-426614174099" });
    assert.equal(peticiones.length, 1);
  }
  cerrar();
});

test("una aceptación portal sin resolución no habilita la propuesta al haber comunicaciones CT", async () => {
  const raiz = raizPrueba(); let peticiones = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [comunicacionCT()] }),
    prepararPropuestaFormalizacion: async () => { peticiones++; },
  }, { contexto: contexto(resultado()), fetchPublicaciones: publicaciones });
  await esperar();
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  await raiz.enviar("propuesta", { clave_idempotencia: CLAVE });
  assert.equal(peticiones, 0);
  cerrar();
});

for (const estado of [403, 503]) test(`la propuesta portal resuelta no depende de una lectura CT ${estado}`, async () => {
  const raiz = raizPrueba(), peticiones = [];
  const cerrar = montar(raiz, {
    resolverLlamamiento: async () => recibo(),
    consultarComunicacionesExpediente: async () => { throw Object.assign(new Error("lectura CT no disponible"),
      { estado, codigo: estado === 403 ? "acceso_denegado" : "servicio_no_disponible", envelopeValido: true }); },
    prepararPropuestaFormalizacion: async (s) => {
      peticiones.push(s);
      return { esquema: "vec.contratacion-temporal.propuesta-formalizacion-local.v1", estado_local: "confirmado",
        propuesta_ref: "propuesta:portal:001", recibo_local_ref: "recibo:propuesta:portal:001",
        version_resultante: 7, confirmada_en: "2026-10-20T10:05:00Z" };
    },
  }, { contexto: contexto(resultado()), fetchPublicaciones: publicaciones });
  await esperar();
  await enviar(raiz, formulario());
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="propuesta"/u);
  await raiz.enviar("propuesta", { clave_idempotencia: CLAVE });
  assert.equal(peticiones.length, 1);
  assert.equal(peticiones[0].recibo_resolucion_aceptada_ref, recibo().recibo_local_ref);
  assert.match(raiz.innerHTML, /Propuesta guardada/u);
  cerrar();
});
