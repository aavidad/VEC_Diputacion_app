import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal, RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js";
import { validarSolicitudResolucionFormalizacion, validarReciboResolucionFormalizacion, validarPreparacionResolucionFormalizacion } from "./contrato-resolucion-formalizacion.js";
import { montarFormularioResolucionFormalizacion } from "./formulario-resolucion-formalizacion.js";

function errorHTTPResolucion(estado) {
  const codigo = { 400: "peticion_no_valida", 403: "acceso_denegado",
    409: "clave_idempotencia_reutilizada", 422: "contenido_no_valido", 503: "servicio_no_disponible" }[estado];
  return new Response(JSON.stringify({ error: { codigo,
    clave_i18n: "api.contratacion_temporal.resolucion_formalizacion.error." + codigo,
    correlacion_ref: "corr_0123456789abcdef0123456789abcdef" } }),
  { status: estado, headers: { "content-type": "application/json; charset=utf-8" } });
}
const jsonPreparacion = (data) => new Response(JSON.stringify({ data }), {
  status: 200, headers: { "content-type": "application/json; charset=utf-8" },
});

test("P1 cliente: 409 nominal conserva incertidumbre; 400/403/422 siguen determinados", async () => {
  for (const estado of [400, 403, 409, 422]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => errorHTTPResolucion(estado) });
    await assert.rejects(cliente.registrarResolucionFormalizacion(solicitud), (error) =>
      error.estado === estado && error.envelopeValido === true
      && error.resultadoIndeterminado === (estado === 409));
  }
});

test("P1 formulario y cliente: 409→GET v8 muestra historia, sin otro POST ni confirmación de material divergente", async () => {
  const raiz = raizFormulario(), llamadas = [];
  const historico = { ...preparacionFormulario, version_actual: 8,
    recibo: { ...recibo, recibo_ref: "recibo:original:historico", estado: "replay_registrada" } };
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, metodo: opciones.method, cuerpo: opciones.body });
    return opciones.method === "POST" ? errorHTTPResolucion(409) : jsonPreparacion(historico);
  } });
  montarFormularioResolucionFormalizacion({ raiz, cliente, preparacion: preparacionFormulario,
    generarClaveIdempotencia: () => solicitud.clave_idempotencia, confirmarOperacion: () => true });
  await raiz.enviar({ ...valoresFormulario, motivo: "Material divergente" });
  assert.deepEqual(llamadas.map((x) => x.metodo), ["POST", "GET"]);
  assert.equal(llamadas[1].ruta, RUTAS_HTTP_CONTRATACION_TEMPORAL.resolucionFormalizacion
    + "?expediente_ref=" + encodeURIComponent(solicitud.expediente_ref));
  assert.equal(llamadas[1].cuerpo, undefined);
  assert.match(raiz.innerHTML, /recibo:original:historico/u);
  assert.match(raiz.innerHTML, /No confirma que coincida/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-resolucion-formalizacion-form/u);
  await raiz.enviar(valoresFormulario);
  assert.equal(llamadas.length, 2);
});

test("P1 formulario y cliente: 409→GET v7 sin recibo conserva clave y cuerpo en reintento exacto", async () => {
  const raiz = raizFormulario(), posts = []; let claves = 0, gets = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (_ruta, opciones) => {
    if (opciones.method === "GET") { gets += 1; return jsonPreparacion(preparacionFormulario); }
    posts.push(opciones.body);
    return posts.length === 1 ? errorHTTPResolucion(409) : new Response(JSON.stringify({ data: recibo }),
      { status: 201, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  montarFormularioResolucionFormalizacion({ raiz, cliente, preparacion: preparacionFormulario,
    generarClaveIdempotencia: () => { claves += 1; return solicitud.clave_idempotencia; }, confirmarOperacion: () => true });
  await raiz.enviar(valoresFormulario);
  assert.match(raiz.innerHTML, /<fieldset disabled/u);
  assert.match(raiz.innerHTML, /La clave puede tener uso previo/u);
  assert.doesNotMatch(raiz.innerHTML, /type="submit" disabled/u);
  await raiz.enviar({ ...valoresFormulario, motivo: "No debe enviarse", numero_resolucion: "Cambio prohibido" });
  assert.equal(posts.length, 2); assert.equal(posts[1], posts[0]);
  assert.equal(claves, 1); assert.equal(gets, 1);
  assert.match(raiz.innerHTML, /recibo:ct:001/u);
});

for (const caso of ["red", 403, 503, "expediente ajeno", "propuesta ajena"]) {
  test(`P1 recuperación GET ${caso}: no libera solicitud ni clave`, async () => {
    const raiz = raizFormulario(), posts = []; let claves = 0;
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (_ruta, opciones) => {
      if (opciones.method === "POST") { posts.push(opciones.body); return errorHTTPResolucion(409); }
      if (caso === "red") throw new TypeError("detalle privado");
      if (typeof caso === "number") return errorHTTPResolucion(caso);
      const otro = { ...preparacionFormulario, version_actual: 8, recibo: { ...recibo } };
      if (caso === "expediente ajeno") {
        otro.expediente_ref = otro.recibo.expediente_ref = "expediente:ajeno";
      } else otro.propuesta_ref = otro.recibo.propuesta_ref = "propuesta:ajena";
      return jsonPreparacion(otro);
    } });
    montarFormularioResolucionFormalizacion({ raiz, cliente, preparacion: preparacionFormulario,
      generarClaveIdempotencia: () => { claves += 1; return solicitud.clave_idempotencia; }, confirmarOperacion: () => true });
    await raiz.enviar(valoresFormulario);
    assert.match(raiz.innerHTML, /<fieldset disabled/u);
    assert.match(raiz.innerHTML, /No se pudo verificar/u);
    assert.doesNotMatch(raiz.innerHTML, /detalle privado|recibo:ct:001/u);
    await raiz.enviar({ ...valoresFormulario, motivo: "No se permite editar" });
    assert.equal(posts.length, 2); assert.equal(posts[0], posts[1]); assert.equal(claves, 1);
  });
}

test("P1 GET de recuperación conserva aborto y descarta resolución tras desmontaje", async () => {
  const raiz = raizFormulario(); let resolver, signalGET;
  const cliente = {
    async registrarResolucionFormalizacion() { throw Object.assign(new Error("409"), { estado: 409,
      envelopeValido: true, resultadoIndeterminado: true }); },
    prepararResolucionFormalizacion(_ref, opciones) {
      signalGET = opciones.signal; return new Promise((r) => { resolver = r; });
    },
  };
  const desmontar = montarFormularioResolucionFormalizacion({ raiz, cliente, preparacion: preparacionFormulario,
    generarClaveIdempotencia: () => solicitud.clave_idempotencia, confirmarOperacion: () => true });
  const vuelo = raiz.enviar(valoresFormulario);
  await new Promise((r) => setImmediate(r));
  assert.equal(signalGET.aborted, false);
  desmontar(); const html = raiz.innerHTML;
  assert.equal(signalGET.aborted, true);
  resolver({ ...preparacionFormulario, version_actual: 8, recibo });
  await vuelo; assert.equal(raiz.innerHTML, html);
});


const solicitud = Object.freeze({ expediente_ref: "expediente:ct:001", version_esperada: 7,
  propuesta_ref: "propuesta:ct:001", clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
  numero_resolucion: "R-001/2026", fecha_resolucion: "2026-09-06", motivo: "Ejercicio revisado.",
  confirma_revision_propuesta: true, confirma_ejercicio_manual: true });
const recibo = Object.freeze({ esquema: "vec.contratacion-temporal.resolucion-formalizacion.v1", estado: "registrada",
  expediente_ref: solicitud.expediente_ref, version_resultante: 8, propuesta_ref: solicitud.propuesta_ref,
  resolucion_formalizacion_ref: "resolucion:ct:001", documento_resolucion_ref: "documento:ct:001",
  documento_resolucion_version: 1, documento_resolucion_sha256: "a".repeat(64), actuacion_ref: "actuacion:ct:001",
  auditoria_ref: "auditoria:ct:001", outbox_ref: "outbox:ct:001", recibo_ref: "recibo:ct:001",
  registrada_en: "2026-09-06T12:00:00Z", tipo_validacion: "manual_de_ejercicio",
  firma_oficial: false, eficacia_administrativa: false });

test("resolución de formalización: contrato cerrado v7→v8 y límites explícitos", () => {
  assert.deepEqual(validarSolicitudResolucionFormalizacion(solicitud), solicitud);
  assert.deepEqual(validarReciboResolucionFormalizacion(recibo, solicitud), recibo);
  assert.throws(() => validarSolicitudResolucionFormalizacion({ ...solicitud, firma_oficial: false }), TypeError);
  assert.throws(() => validarSolicitudResolucionFormalizacion({ ...solicitud, confirma_ejercicio_manual: false }), TypeError);
  assert.throws(() => validarReciboResolucionFormalizacion({ ...recibo, eficacia_administrativa: true }, solicitud), TypeError);
});

test("resolución de formalización: admite UUID v4 real y rechaza versión o variante ajenas", () => {
  const claveReal = randomUUID();
  assert.match(claveReal, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u);
  assert.equal(validarSolicitudResolucionFormalizacion({ ...solicitud, clave_idempotencia: claveReal }).clave_idempotencia, claveReal);
  const versionAjena = claveReal.split("-"); versionAjena[2] = "5" + versionAjena[2].slice(1);
  assert.throws(() => validarSolicitudResolucionFormalizacion({ ...solicitud, clave_idempotencia: versionAjena.join("-") }), TypeError);
  const grupos = claveReal.split("-"); grupos[3] = `7${grupos[3].slice(1)}`;
  assert.throws(() => validarSolicitudResolucionFormalizacion({ ...solicitud, clave_idempotencia: grupos.join("-") }), TypeError);
});

test("resolución de formalización: cliente envía solo el contrato y distingue registro", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return new Response(JSON.stringify({ data: recibo }), { status: 201, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  assert.deepEqual(await cliente.registrarResolucionFormalizacion(solicitud), recibo);
  assert.equal(llamadas[0].ruta, RUTAS_HTTP_CONTRATACION_TEMPORAL.resolucionFormalizacion);
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), solicitud);
});

test("resolución de formalización: preparación GET nominal no envía cuerpo", async () => {
  const preparacion = { esquema: "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1", expediente_ref: solicitud.expediente_ref,
    propuesta_ref: solicitud.propuesta_ref, version_esperada: 7, version_actual: 7, recibo: null };
  assert.deepEqual(validarPreparacionResolucionFormalizacion(preparacion, solicitud.expediente_ref), preparacion);
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones }); return new Response(JSON.stringify({ data: preparacion }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  assert.deepEqual(await cliente.prepararResolucionFormalizacion(solicitud.expediente_ref), preparacion);
  assert.equal(llamadas[0].ruta, `${RUTAS_HTTP_CONTRATACION_TEMPORAL.resolucionFormalizacion}?expediente_ref=${encodeURIComponent(solicitud.expediente_ref)}`);
  assert.equal(llamadas[0].opciones.method, "GET"); assert.equal(llamadas[0].opciones.body, undefined);
});

test("preparación v8 recupera recibo autorizado sin construir un POST", () => {
  const preparada = { esquema: "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1", expediente_ref: solicitud.expediente_ref,
    propuesta_ref: solicitud.propuesta_ref, version_esperada: 7, version_actual: 8, recibo };
  assert.equal(validarPreparacionResolucionFormalizacion(preparada, solicitud.expediente_ref).recibo.recibo_ref, recibo.recibo_ref);
});

function raizFormulario() { const eventos = new Map(); const raiz = { innerHTML: "", addEventListener: (t, h) => eventos.set(t, h), removeEventListener() {}, replaceChildren() {},
  enviar(valores) { const e = Object.fromEntries(Object.entries(valores).map(([k, v]) => [k, typeof v === "boolean" ? { checked: v } : { value: v }])); return eventos.get("submit")({ preventDefault() {}, target: { elements: e } }); } }; return raiz; }
const preparacionFormulario = { esquema: "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1", expediente_ref: solicitud.expediente_ref, propuesta_ref: solicitud.propuesta_ref, version_esperada: 7, version_actual: 7, recibo: null };
const valoresFormulario = { numero_resolucion: "R-2", fecha_resolucion: "2026-09-06", motivo: "Motivo", confirma_revision_propuesta: true, confirma_ejercicio_manual: true, clave_idempotencia: "" };
test("formulario: validación local y cancelación no envían ni congelan el borrador", async () => {
  const raiz = raizFormulario(); let llamadas = 0; let confirmar = false; let enviada;
  montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario, generarClaveIdempotencia: () => solicitud.clave_idempotencia, confirmarOperacion: () => confirmar, cliente: { registrarResolucionFormalizacion(s) { llamadas += 1; enviada = s; return Promise.resolve(recibo); } } });
  assert.doesNotMatch(raiz.innerHTML, / checked/u);
  await raiz.enviar({ ...valoresFormulario, confirma_ejercicio_manual: false }); assert.equal(llamadas, 0); assert.match(raiz.innerHTML, /Revise número/u);
  await raiz.enviar(valoresFormulario); assert.equal(llamadas, 0); assert.match(raiz.innerHTML, /cancelada/u);
  confirmar = true; await raiz.enviar({ ...valoresFormulario, motivo: "Corregido" }); assert.equal(llamadas, 1);
  assert.equal(enviada.motivo, "Corregido");
  assert.match(raiz.innerHTML, /recibo:ct:001/u);
});

test("formulario: cancelar recuperación incierta conserva cuerpo y clave exactos", async () => {
  const raiz = raizFormulario(); const enviados = []; const confirmaciones = [true, false, true]; let claves = 0;
  montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario, generarClaveIdempotencia: () => { claves += 1; return solicitud.clave_idempotencia; },
    confirmarOperacion: () => confirmaciones.shift(), cliente: { registrarResolucionFormalizacion(x) { enviados.push(x); return enviados.length === 1 ? Promise.reject(new TypeError("red")) : Promise.resolve({ ...recibo, propuesta_ref: x.propuesta_ref, expediente_ref: x.expediente_ref }); } } });
  await raiz.enviar(valoresFormulario);
  assert.match(raiz.innerHTML, /<fieldset disabled/u);
  assert.doesNotMatch(raiz.innerHTML, /type="submit" disabled/u);
  await raiz.enviar({ ...valoresFormulario, motivo: "Modificado" });
  assert.match(raiz.innerHTML, /<fieldset disabled/u);
  await raiz.enviar({ ...valoresFormulario, motivo: "Modificado" });
  assert.equal(enviados.length, 2); assert.deepEqual(enviados[1], enviados[0]);
  assert.equal(enviados[0].clave_idempotencia, solicitud.clave_idempotencia);
  assert.equal(claves, 1); assert.match(raiz.innerHTML, /recibo:ct:001/u);
});

test("formulario: 201 muestra recibo válido y sus límites", async () => {
  const raiz = raizFormulario(); let llamadas = 0;
  montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario, generarClaveIdempotencia: () => solicitud.clave_idempotencia,
    confirmarOperacion: () => true, cliente: { registrarResolucionFormalizacion(x) { llamadas += 1; return Promise.resolve({ ...recibo, expediente_ref: x.expediente_ref, propuesta_ref: x.propuesta_ref }); } } });
  await raiz.enviar(valoresFormulario);
  assert.equal(llamadas, 1); assert.match(raiz.innerHTML, /recibo:ct:001/u); assert.match(raiz.innerHTML, /Firma oficial: no/u);
});

test("formulario: 422 determinado permite corregir motivo y conserva clave", async () => {
  const raiz = raizFormulario(); const enviados = []; let intento = 0;
  montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario, generarClaveIdempotencia: () => solicitud.clave_idempotencia,
    confirmarOperacion: () => true, cliente: { registrarResolucionFormalizacion(x) { enviados.push(x); intento += 1; if (intento === 1) { const e = new Error("422"); e.envelopeValido = true; e.resultadoIndeterminado = false; return Promise.reject(e); } return Promise.resolve({ ...recibo, expediente_ref: x.expediente_ref, propuesta_ref: x.propuesta_ref }); } } });
  await raiz.enviar(valoresFormulario);
  assert.doesNotMatch(raiz.innerHTML, /<fieldset disabled/u);
  assert.match(raiz.innerHTML, /R-2/u); assert.match(raiz.innerHTML, /Motivo/u);
  await raiz.enviar({ ...valoresFormulario, numero_resolucion: "R-3", motivo: "Motivo corregido" });
  assert.equal(enviados.length, 2); assert.equal(enviados[1].motivo, "Motivo corregido"); assert.equal(enviados[1].clave_idempotencia, enviados[0].clave_idempotencia);
  assert.equal(enviados[1].numero_resolucion, "R-3"); assert.match(raiz.innerHTML, /recibo:ct:001/u);
});

test("formulario: sobrescritura i18n se aplica al texto visible y queda escapada", () => {
  const raiz = raizFormulario();
  montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario, cliente: { registrarResolucionFormalizacion() {} },
    mensajes: { resolucion_formalizacion_titulo: "Resolución <revisada>" } });
  assert.match(raiz.innerHTML, /Resolución &lt;revisada&gt;/u);
  assert.doesNotMatch(raiz.innerHTML, /Resolución <revisada>/u);
});

test("formulario: i18n de estados y etiquetas, contexto visible y fecha localizada", async () => {
  const raiz = raizFormulario();
  const mensajes = { resolucion_formalizacion_validacion: "Revisión <necesaria>",
    resolucion_formalizacion_recibo_ref: "Justificante <propio>" };
  montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario,
    generarClaveIdempotencia: () => solicitud.clave_idempotencia, confirmarOperacion: () => true,
    mensajes, locale: "en-GB", zonaHoraria: "UTC",
    cliente: { registrarResolucionFormalizacion: async () => recibo } });
  assert.match(raiz.innerHTML, /expediente:ct:001/u);
  assert.match(raiz.innerHTML, /propuesta:ct:001/u);
  await raiz.enviar({ ...valoresFormulario, confirma_revision_propuesta: false });
  assert.match(raiz.innerHTML, /Revisión &lt;necesaria&gt;/u);
  await raiz.enviar(valoresFormulario);
  assert.match(raiz.innerHTML, /Justificante &lt;propio&gt;/u);
  const fecha = new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "medium", timeZone: "UTC" }).format(new Date(recibo.registrada_en));
  assert.ok(raiz.innerHTML.includes(fecha));
});

test("formulario: petición pendiente bloquea doble envío y no repinta después de desmontar", async () => {
  const raiz = raizFormulario(); let resolver, llamadas = 0;
  const desmontar = montarFormularioResolucionFormalizacion({ raiz, preparacion: preparacionFormulario,
    generarClaveIdempotencia: () => solicitud.clave_idempotencia, confirmarOperacion: () => true,
    cliente: { registrarResolucionFormalizacion() { llamadas += 1; return new Promise((r) => { resolver = r; }); } } });
  const vuelo = raiz.enviar(valoresFormulario);
  assert.match(raiz.innerHTML, /aria-busy="true"/u);
  await raiz.enviar(valoresFormulario); assert.equal(llamadas, 1);
  desmontar(); const html = raiz.innerHTML; resolver(recibo); await vuelo;
  assert.equal(raiz.innerHTML, html);
});
