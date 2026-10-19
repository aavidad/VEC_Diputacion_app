import assert from "node:assert/strict";
import test from "node:test";
import {
  CONFLICTOS_SEGUIMIENTO_CESE, RUTA_NO_INCORPORACIONES, RUTAS_SEGUIMIENTO_CESE, crearClienteSeguimientoCeseHTTP,
  validarConsultaSeguimientoCese, validarSolicitudNoIncorporacion,
} from "./cliente-http-seguimiento-cese.js";
import { montarPanelSeguimientoCese } from "./seguimiento-cese.js";
import { presentarEtiquetasHitoRRHH } from "./adaptador-http-expedientes.js?v=VEC_WEB_VERSION";
import { ofrecerNoIncorporacion, ofrecerPropuestaNoIncorporacion, solicitudNoIncorporacion, solicitudResolucionPropuestaNoIncorporacion } from "./seguimiento-no-incorporacion.js";

const EXP = "expediente:ni:001";
const reglaNo = { motivos: [{ clave: "no_presentado", etiqueta: "No se presenta el día y en el lugar indicados" }], segunda_persona: true };
const opciones = { causas_cese: [{ clave: "fin_sustitucion", etiqueta: "Fin de la sustitución", clave_i18n: "x.y", justificante_tipo: "comunicacion_reincorporacion" }],
  condiciones_cierre: ["cese_registrado", "ginpix_confirmado"], fase_retorno_modificacion: "fiscalizacion",
  motivos_modificacion: [{ clave: "cambio_jornada", etiqueta: "Cambio de jornada", clave_i18n: "x.z" }], confirmacion_ginpix: true, no_incorporacion: reglaNo };
const noInc = { motivo_clave: "no_presentado", resolucion_ref: "resolucion:rrhh:2026/0142", resolucion_sha256: "b".repeat(64), resuelta_por: "per_segunda",
  fecha_notificacion: "2026-09-20", recibo_ref: "recibo:ni:1", registrada_en: "2026-09-21T09:00:00.123456Z" };
const consulta = (estado = {}) => ({ esquema: "vec.contratacion-temporal.seguimiento-cese.v1", opciones,
  estado: { expediente_ref: EXP, incorporacion: null, cese: null, cierre: null, ginpix: null, confirmacion_centro: null, ...estado } });
const contexto = Object.freeze({ expediente_ref: EXP, version: 7, fase_clave: "nombramiento", estado_clave: "en_curso" });
const esperar = () => new Promise((r) => setImmediate(r));
function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}
const solicitud = { expediente_ref: EXP, version_esperada: 7, clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000", paso: "proponer", propuesta_ref: "",
  motivo_clave: "no_presentado", resolucion_ref: "resolucion:rrhh:2026/0142", resolucion_sha256: "b".repeat(64), resuelta_por: "", fecha_notificacion: "2026-09-20", observaciones: "" };
const propuesta = { propuesta_ref: "recibo:ni:propuesta", motivo_clave: "no_presentado", resolucion_ref: "resolucion:rrhh:2026/0142", resolucion_sha256: "b".repeat(64),
  fecha_notificacion: "2026-09-20", registrada_en: "2026-09-21T08:00:00.123456Z" };

test("la no incorporación se valida y viaja por su ruta, con sus conflictos conocidos", async () => {
  assert.deepEqual(validarSolicitudNoIncorporacion({ ...solicitud }), solicitud);
  assert.equal(validarSolicitudNoIncorporacion({ ...solicitud, paso: "registrar", resuelta_por: "per_declarada" }).resuelta_por, "per_declarada");
  assert.equal(validarSolicitudNoIncorporacion({ ...solicitud, paso: "confirmar", propuesta_ref: "recibo:ni:propuesta" }).paso, "confirmar");
  for (const cambio of [{ motivo_clave: "No" }, { resolucion_sha256: "x" }, { fecha_notificacion: "2026-02-30" }, { resuelta_por: "per_segunda" },
    { paso: "registrar" }, { paso: "confirmar" }, { paso: "rechazar", propuesta_ref: "recibo:ni:propuesta", resuelta_por: "per_otra" },
    { propuesta_ref: "recibo:ni:propuesta" }, { paso: "firmar" }, { extra: 1 }]) {
    assert.throws(() => validarSolicitudNoIncorporacion({ ...solicitud, ...cambio }), TypeError, JSON.stringify(cambio));
  }
  assert.ok(RUTAS_SEGUIMIENTO_CESE.includes(RUTA_NO_INCORPORACIONES));
  for (const c of ["sin_aceptacion", "incorporacion_existente", "no_incorporacion_existente", "fecha_no_admitida", "propuesta_pendiente",
    "propuesta_no_valida", "misma_persona"]) assert.ok(CONFLICTOS_SEGUIMIENTO_CESE.includes(c));
  const llamadas = [];
  const cliente = crearClienteSeguimientoCeseHTTP({ ejecutar: async (p) => { llamadas.push(p); return p.validarRespuesta({ esquema: "vec.contratacion-temporal.recibo-seguimiento.v1",
    operacion: "registrar_no_incorporacion", expediente_ref: EXP, version_anterior: 7, version_resultante: 8, fase_resultante: "fiscalizacion", estado_resultante: "en_curso",
    recibo_ref: "recibo:ni:1", auditoria_ref: "auditoria:ni:1", evento_ref: "evento:ni:1", registrada_en: "2026-09-26T10:00:00Z", causa_clave: "no_presentado" }); },
  validarOpciones: (o) => ({ signal: o?.signal }), serializarAcotado: (v) => JSON.stringify(v) });
  assert.equal((await cliente.registrarNoIncorporacion({ ...solicitud })).fase_resultante, "fiscalizacion");
  assert.equal(llamadas[0].ruta, RUTA_NO_INCORPORACIONES);
});

test("la consulta valida las opciones y la no incorporación registrada", () => {
  assert.equal(validarConsultaSeguimientoCese(consulta({ no_incorporacion: noInc }), EXP).estado.no_incorporacion.motivo_clave, "no_presentado");
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ no_incorporacion: { ...noInc, resolucion_sha256: "x" } }), EXP), TypeError);
  assert.equal(validarConsultaSeguimientoCese(consulta({ no_incorporacion_propuesta: propuesta }), EXP).estado.no_incorporacion_propuesta.propuesta_ref, "recibo:ni:propuesta");
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ no_incorporacion_propuesta: { ...propuesta, actor_ref: "per_actor" } }), EXP), TypeError);
  const sinMotivos = consulta();
  sinMotivos.opciones = { ...opciones, no_incorporacion: { motivos: [], segunda_persona: true } };
  assert.throws(() => validarConsultaSeguimientoCese(sinMotivos, EXP), TypeError);
});

test("el panel ofrece registrar que no se incorpora sin incorporación y lo resume después", async () => {
  const c = contenedorFalso();
  const desmontar = montarPanelSeguimientoCese({ contenedor: c, cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta(), EXP) }, contexto });
  await esperar();
  assert.match(c.innerHTML, /data-ct-seg-form="no_incorporacion"/u);
  assert.match(c.innerHTML, /Proponer que no se incorpora/u);
  assert.doesNotMatch(c.innerHTML, /name="resuelta_por"/u, "con segunda persona nadie declara quién resuelve");
  assert.match(c.innerHTML, /name="resolucion_sha256"/u);
  assert.doesNotMatch(c.innerHTML, /b24\.sancion/u, "la consecuencia interna no se muestra");
  desmontar();
  const d = contenedorFalso();
  const quitar = montarPanelSeguimientoCese({ contenedor: d, contexto,
    cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta({ no_incorporacion: noInc }), EXP) } });
  await esperar();
  assert.doesNotMatch(d.innerHTML, /data-ct-seg-form="no_incorporacion"/u);
  assert.match(d.innerHTML, /No se incorpora: No se presenta el día y en el lugar indicados\. Resolución notificada el 20 de septiembre de 2026\./u);
  assert.doesNotMatch(d.innerHTML, /resolucion:rrhh|recibo:ni/u, "sin referencias internas en pantalla");
  quitar();
});

test("con incorporación o sin la regla no se ofrece; la solicitud sale del formulario", () => {
  assert.equal(ofrecerNoIncorporacion({ incorporacion: { inicio: "2026-10-01" } }, opciones), false);
  assert.equal(ofrecerNoIncorporacion({ incorporacion: null }, { ...opciones, no_incorporacion: undefined }), false);
  assert.equal(ofrecerNoIncorporacion({ incorporacion: null, cese: null }, opciones), true);
  const [metodo, s] = solicitudNoIncorporacion({ expediente_ref: EXP }, { motivo_clave: "no_presentado", resolucion_sha256: "B".repeat(64) });
  assert.equal(metodo, "registrarNoIncorporacion");
  assert.equal(s.resolucion_sha256, "b".repeat(64));
});

test("la segunda persona ve la propuesta pendiente y la confirma o rechaza con sus datos", async () => {
  assert.equal(ofrecerNoIncorporacion({ incorporacion: null, no_incorporacion_propuesta: propuesta }, opciones), false);
  assert.equal(ofrecerPropuestaNoIncorporacion({ incorporacion: null, no_incorporacion_propuesta: propuesta }, opciones), true);
  assert.equal(ofrecerPropuestaNoIncorporacion({ incorporacion: null, no_incorporacion_propuesta: propuesta },
    { ...opciones, no_incorporacion: { ...reglaNo, segunda_persona: false } }), false);
  const c = contenedorFalso();
  const quitar = montarPanelSeguimientoCese({ contenedor: c, contexto,
    cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta({ no_incorporacion_propuesta: propuesta }), EXP) } });
  await esperar();
  assert.match(c.innerHTML, /Propuesta de no incorporación pendiente/u);
  assert.match(c.innerHTML, /data-ct-seg-form="no_incorporacion_confirmar"/u);
  assert.match(c.innerHTML, /data-ct-seg-form="no_incorporacion_rechazar"/u);
  assert.match(c.innerHTML, /Pendiente de revisión por otra persona de RRHH: No se presenta/u);
  assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="no_incorporacion"[ >]/u);
  assert.doesNotMatch(c.innerHTML, /recibo:ni:propuesta/u, "la referencia interna de la propuesta no se expone");
  assert.match(c.innerHTML, /resolucion:rrhh:2026\/0142/u, "la resolución se muestra para revisarla");
  assert.match(c.innerHTML, /<input type="text" readonly value="[^"]*revision-no-incorporacion\.html\?expediente_ref=/u);
  assert.match(c.innerHTML, /Copiar enlace de revisión/u);
  assert.match(c.innerHTML, /Envíe este enlace a otra persona de RRHH\. No recibe aviso automático\./u);
  quitar();
  // Quien hizo la propuesta no ve Confirmar ni Rechazar: se le explica.
  const propia = contenedorFalso();
  const quitarPropia = montarPanelSeguimientoCese({ contenedor: propia, contexto,
    cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta({ no_incorporacion_propuesta: { ...propuesta, propuesta_propia: true } }), EXP) } });
  await esperar();
  assert.doesNotMatch(propia.innerHTML, /data-ct-seg-form="no_incorporacion_(?:confirmar|rechazar)"/u);
  assert.match(propia.innerHTML, /Esta propuesta la ha hecho usted\. Otra persona de RRHH tiene que confirmarla o rechazarla\./u);
  assert.match(propia.innerHTML, /Copiar enlace de revisión/u, "quien propone necesita el enlace para enviarlo");
  quitarPropia();
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ no_incorporacion_propuesta: { ...propuesta, propuesta_propia: "si" } }), EXP), TypeError);
  const [metodo, s] = solicitudResolucionPropuestaNoIncorporacion({ expediente_ref: EXP, version_esperada: 8, clave_idempotencia: "123e4567-e89b-42d3-a456-426614174001" },
    propuesta, "confirmar", { observaciones: "Revisada" });
  assert.equal(metodo, "registrarNoIncorporacion");
  assert.deepEqual(validarSolicitudNoIncorporacion(s), { expediente_ref: EXP, version_esperada: 8, clave_idempotencia: "123e4567-e89b-42d3-a456-426614174001",
    paso: "confirmar", propuesta_ref: "recibo:ni:propuesta", motivo_clave: "no_presentado", resolucion_ref: "resolucion:rrhh:2026/0142",
    resolucion_sha256: "b".repeat(64), resuelta_por: "", fecha_notificacion: "2026-09-20", observaciones: "Revisada" });
  const [, registro] = solicitudNoIncorporacion({ expediente_ref: EXP }, { motivo_clave: "no_presentado", resuelta_por: "per_declarada" }, { segunda_persona: false });
  assert.equal(registro.paso, "registrar");
  assert.equal(registro.resuelta_por, "per_declarada");
});

test("la revisión acotada usa versión de la respuesta, solo permite resolver y conserva el recibo tras releer", async () => {
  const c = contenedorFalso();
  const metadata = { version_expediente: 9, fase_clave: "nombramiento", estado_clave: "en_curso", numero_expediente: "CT-2026-9" };
  let pendiente = true, consultas = 0, envios = 0;
  const recibo = { recibo_ref: "recibo:ni:confirmado", version_resultante: 10 };
  const cliente = { consultarSeguimientoCese: async () => { consultas++; return validarConsultaSeguimientoCese(
    consulta({ ...metadata, ...(pendiente ? { no_incorporacion_propuesta: propuesta } : {}) }), EXP); },
    registrarNoIncorporacion: async (s) => { envios++; assert.equal(s.version_esperada, 9); assert.equal(s.propuesta_ref, propuesta.propuesta_ref);
      pendiente = false; return recibo; } };
  const previo = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.f = f; } entries() { return Object.entries(this.f.campos); } };
  try {
    const quitar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto: { expediente_ref: EXP, version: 99 }, soloNoIncorporacion: true,
      generarClave: () => solicitud.clave_idempotencia, confirmarOperacion: () => true });
    await esperar();
    assert.match(c.innerHTML, /data-ct-seg-form="no_incorporacion_confirmar"/u);
    assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="(?:cese|cierre|modificacion|ginpix|no_incorporacion)"/u);
    c.eventos.get("submit")({ target: { closest: () => ({ dataset: { ctSegForm: "cese" }, campos: {} }) }, preventDefault() {} });
    await esperar();
    assert.equal(envios, 0, "un formulario ajeno inyectado no envía nada");
    c.eventos.get("submit")({ target: { closest: () => ({ dataset: { ctSegForm: "no_incorporacion_confirmar" }, campos: { observaciones: "" } }) }, preventDefault() {} });
    await esperar();
    assert.equal(envios, 1);
    assert.equal(consultas, 2, "tras el recibo se recupera el estado sin otro POST");
    assert.match(c.innerHTML, /No incorporación confirmada\./u);
    assert.doesNotMatch(c.innerHTML, /recibo:ni:confirmado|Versión 10/u, "la revisión no enseña identificadores técnicos");
    assert.match(c.innerHTML, /No hay una propuesta pendiente/u);
    assert.doesNotMatch(c.innerHTML, /data-ct-seg-form="no_incorporacion_confirmar"/u);
    quitar();
  } finally { globalThis.FormData = previo; }
});

test("tras rechazar la revisión dice qué ha pasado sin volver a leer", async () => {
  const c = contenedorFalso();
  const metadata = { version_expediente: 9, fase_clave: "nombramiento", estado_clave: "en_curso", numero_expediente: "CT-2026-9" };
  let consultas = 0;
  const cliente = { consultarSeguimientoCese: async () => {
    if (++consultas > 1) throw Object.assign(new Error("acceso denegado"), { estado: 403, envelopeValido: true });
    return validarConsultaSeguimientoCese(consulta({ ...metadata, no_incorporacion_propuesta: propuesta }), EXP);
  }, registrarNoIncorporacion: async (s) => {
    assert.equal(s.paso, "rechazar");
    return { recibo_ref: "recibo:ni:rechazo", version_resultante: 10 };
  } };
  const previo = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.f = f; } entries() { return Object.entries(this.f.campos); } };
  try {
    const quitar = montarPanelSeguimientoCese({ contenedor: c, cliente, contexto: { expediente_ref: EXP }, soloNoIncorporacion: true,
      generarClave: () => solicitud.clave_idempotencia, confirmarOperacion: () => true });
    await esperar();
    c.eventos.get("submit")({ target: { closest: () => ({ dataset: { ctSegForm: "no_incorporacion_rechazar" }, campos: { observaciones: "" } }) }, preventDefault() {} });
    await esperar();
    assert.equal(consultas, 1, "tras rechazar ya no hay nada que esta persona pueda leer");
    assert.match(c.innerHTML, /Propuesta rechazada\./u);
    assert.match(c.innerHTML, /Ha rechazado la propuesta\. El expediente sigue en nombramiento, sin cambios\./u);
    assert.match(c.innerHTML, /href="\/portal-empleado\/\?lang=[a-z]{2}#contratacion-temporal"/u);
    assert.doesNotMatch(c.innerHTML, /recibo:ni:rechazo|data-ct-seg-form=/u);
    quitar();
  } finally { globalThis.FormData = previo; }
});

test("al reabrir la revisión se ve la no incorporación ya confirmada y qué sigue", async () => {
  const c = contenedorFalso();
  const quitar = montarPanelSeguimientoCese({ contenedor: c, contexto: { expediente_ref: EXP }, soloNoIncorporacion: true,
    cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta({
      version_expediente: 10, fase_clave: "fiscalizacion", estado_clave: "en_curso", numero_expediente: "CT-2026-9", no_incorporacion: noInc,
    }), EXP) } });
  await esperar();
  assert.match(c.innerHTML, /La no incorporación está confirmada\. El expediente vuelve a fiscalización/u);
  assert.match(c.innerHTML, /Ir a Peticiones de personal temporal/u);
  assert.match(c.innerHTML, /No se incorpora: No se presenta/u);
  assert.doesNotMatch(c.innerHTML, /recibo:ni:1/u, "sin recibo técnico en pantalla");
  assert.doesNotMatch(c.innerHTML, /data-ct-seg-form=/u);
  quitar();
});

test("el aviso de archivo de la no incorporación usa el idioma del panel y enfoca el campo", async () => {
  const { readFile } = await import("node:fs/promises");
  const { general } = JSON.parse(await readFile(new URL("../../../textos/en/contratacion-temporal-seguimiento-cese.json", import.meta.url), "utf8"));
  const mensajes = Object.fromEntries(Object.entries(general).map(([clave, valor]) => [`seguimiento_cese.${clave}`, valor]));
  const c = contenedorFalso();
  c.addEventListener = (nombre, escucha, captura) => c.eventos.set(`${nombre}${captura ? ":captura" : ""}`, escucha);
  const quitar = montarPanelSeguimientoCese({ contenedor: c, contexto, mensajes,
    cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta(), EXP) } });
  await esperar();
  let enfocado = false;
  let bloqueado = false;
  const estado = { textContent: "" };
  const campo = { querySelector: (selector) => selector === "[data-huella-valor]" ? { value: "" } : estado };
  const control = { dataset: {}, hasAttribute: () => true, setAttribute() {}, closest: () => campo, focus() { enfocado = true; } };
  try {
    c.eventos.get("submit:captura")({ target: { querySelectorAll: () => [control] },
      preventDefault() { bloqueado = true; }, stopImmediatePropagation() {} });
    assert.equal(estado.textContent, "Choose the supporting document file to continue.");
    assert.equal(enfocado, true);
    assert.equal(bloqueado, true);
    assert.match(c.innerHTML, /Propose failure to take up post/u);
  } finally { quitar(); }
});


test("la revisión inglesa traduce el motivo recibido en castellano", async () => {
  const { readFile } = await import("node:fs/promises");
  const { general } = JSON.parse(await readFile(new URL("../../../textos/en/contratacion-temporal-seguimiento-cese.json", import.meta.url), "utf8"));
  const mensajes = Object.fromEntries(Object.entries(general).map(([clave, valor]) => [`seguimiento_cese.${clave}`, valor]));
  const c = contenedorFalso();
  const quitar = montarPanelSeguimientoCese({ contenedor: c, contexto: { expediente_ref: EXP }, mensajes, soloNoIncorporacion: true,
    cliente: { consultarSeguimientoCese: async () => validarConsultaSeguimientoCese(consulta({
      version_expediente: 9, fase_clave: "nombramiento", estado_clave: "en_curso", numero_expediente: "CT-2026-9", no_incorporacion_propuesta: propuesta,
    }), EXP) } });
  await esperar();
  assert.match(c.innerHTML, /Does not attend at the specified time and place/u);
  assert.doesNotMatch(c.innerHTML, /No se presenta/u);
  quitar();
});

test("el historial nombra propuesta, confirmación y rechazo de la no incorporación", () => {
  const accion = (accion_clave) => presentarEtiquetasHitoRRHH({ accion_clave, fase_origen: "nombramiento", fase_destino: "nombramiento",
    estado_origen: "en_curso", estado_destino: "en_curso" }).accion;
  assert.equal(accion("contratacion_temporal.incorporacion.no_incorporacion_propuesta"), "Propuesta de no incorporación registrada");
  assert.equal(accion("contratacion_temporal.incorporacion.no_incorporacion"), "No incorporación confirmada");
  assert.equal(accion("contratacion_temporal.incorporacion.no_incorporacion_rechazo"), "Propuesta de no incorporación rechazada");
});
