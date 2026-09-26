import assert from "node:assert/strict";
import test from "node:test";
import {
  CONFLICTOS_CANCELACION_EXPEDIENTE, RUTA_CANCELACIONES_EXPEDIENTE, RUTA_CANCELACION_EXPEDIENTE, crearClienteCancelacionHTTP,
  validarConsultaCancelacion, validarReciboCancelacion, validarSolicitudCancelacion,
} from "./cliente-http-cancelacion.js";
import { contextoCancelacionDesdeEstado, montarPanelCancelacion, rutaCancelacionNoMontada } from "./cancelacion-expediente.js";
import { codigoValidoParaRuta, claveI18nValida } from "./cliente-http-transporte.js";
import { presentarEtiquetasHitoRRHH } from "./adaptador-http-expedientes.js";

const EXP = "expediente:ct:cancelacion:001";
const UUID = "123e4567-e89b-42d3-a456-426614174000";
const solicitud = Object.freeze({ expediente_ref: EXP, version_esperada: 3, clave_idempotencia: UUID, motivo_clave: "falta_credito", observaciones: "" });
const recibo = Object.freeze({ esquema: "vec.contratacion-temporal.recibo-cancelacion.v1", operacion: "cancelar_expediente", expediente_ref: EXP,
  version_anterior: 3, version_resultante: 4, fase_resultante: "asignacion_unidad", estado_resultante: "cancelado", motivo_clave: "falta_credito",
  recibo_ref: "recibo:ct-seguimiento:1", auditoria_ref: "aud_v3_0123456789abcdef0123456789abcdef", evento_ref: "evento:ct-seguimiento:1",
  registrada_en: "2026-09-26T10:00:00.123456Z" });
const motivos = [{ clave: "necesidad_desaparecida", etiqueta: "Ha desaparecido la necesidad", clave_i18n: "contratacion_temporal.cancelacion.motivo.necesidad_desaparecida" },
  { clave: "falta_credito", etiqueta: "Sin crédito disponible", clave_i18n: "" }];
const consulta = (cancelacion = null) => ({ esquema: "vec.contratacion-temporal.cancelacion-expediente.v1", expediente_ref: EXP,
  fases_admitidas: ["solicitud", "asignacion_unidad", "informe_juridico"], motivos, cancelacion });
const registrada = Object.freeze({ canal: "centro", motivo_clave: "desistimiento_centro", motivo_etiqueta: "El centro retira su petición", motivo_clave_i18n: "",
  fase_previa: "solicitud", observaciones: "Se reincorpora la titular", recibo_ref: "recibo:ct-seguimiento:9", registrada_en: "2026-09-26T08:30:00Z" });

test("la solicitud se valida antes de salir: motivo obligatorio, observación opcional y sin campos ajenos", () => {
  assert.deepEqual(validarSolicitudCancelacion({ ...solicitud }), solicitud);
  assert.throws(() => validarSolicitudCancelacion({ ...solicitud, motivo_clave: "" }), TypeError);
  assert.throws(() => validarSolicitudCancelacion({ ...solicitud, observaciones: " con borde" }), TypeError);
  assert.throws(() => validarSolicitudCancelacion({ ...solicitud, actor_ref: "per_x" }), TypeError);
  assert.throws(() => validarSolicitudCancelacion({ ...solicitud, clave_idempotencia: "no-uuid" }), TypeError);
});

test("el recibo debe dejar el expediente cancelado en la versión siguiente con el mismo motivo", () => {
  assert.equal(validarReciboCancelacion({ ...recibo }, solicitud).estado_resultante, "cancelado");
  assert.throws(() => validarReciboCancelacion({ ...recibo, estado_resultante: "en_curso" }, solicitud), TypeError);
  assert.throws(() => validarReciboCancelacion({ ...recibo, version_resultante: 5 }, solicitud), TypeError);
  assert.throws(() => validarReciboCancelacion({ ...recibo, motivo_clave: "error_solicitud" }, solicitud), TypeError);
});

test("la consulta exige el esquema, el expediente pedido y una cancelación registrada bien formada", () => {
  assert.equal(validarConsultaCancelacion(consulta(), EXP).cancelacion, null);
  assert.equal(validarConsultaCancelacion(consulta({ ...registrada }), EXP).cancelacion.canal, "centro");
  assert.throws(() => validarConsultaCancelacion(consulta(), "expediente:otro:1"), TypeError);
  assert.throws(() => validarConsultaCancelacion({ ...consulta(), fases_admitidas: [] }, EXP), TypeError);
  assert.throws(() => validarConsultaCancelacion(consulta({ ...registrada, canal: "intervencion" }), EXP), TypeError);
});

test("el cliente usa sus rutas, espera 201 y trata como rechazo cierto solo los conflictos conocidos", async () => {
  const llamadas = [];
  const cliente = crearClienteCancelacionHTTP({ ejecutar: async (p) => { llamadas.push(p); return p.validarRespuesta(p.efecto ? { ...recibo } : consulta()); },
    validarOpciones: (o) => ({ signal: o?.signal }), serializarAcotado: (v) => JSON.stringify(v) });
  assert.equal((await cliente.cancelarExpediente({ ...solicitud })).estado_resultante, "cancelado");
  assert.equal(llamadas[0].ruta, RUTA_CANCELACIONES_EXPEDIENTE);
  assert.equal(llamadas[0].estadoEsperado, 201);
  assert.equal(llamadas[0].efecto, true);
  for (const codigo of CONFLICTOS_CANCELACION_EXPEDIENTE) assert.equal(llamadas[0].rechazoDeterminado({ envelopeValido: true, estado: 409, codigo }), true);
  assert.equal(llamadas[0].rechazoDeterminado({ envelopeValido: true, estado: 409, codigo: "otro" }), false);
  await cliente.consultarCancelacion(EXP);
  assert.equal(llamadas[1].ruta, RUTA_CANCELACION_EXPEDIENTE);
  assert.equal(llamadas[1].efecto, false);
  assert.throws(() => cliente.cancelarExpediente({ ...solicitud, motivo_clave: "" }), TypeError);
  assert.equal(llamadas.length, 2);
});

test("el transporte admite los rechazos propios con su clave i18n", () => {
  assert.equal(codigoValidoParaRuta(RUTA_CANCELACIONES_EXPEDIENTE, 409, "tras_fiscalizacion", {}), true);
  assert.equal(codigoValidoParaRuta(RUTA_CANCELACIONES_EXPEDIENTE, 409, "sin_incorporacion", {}), false);
  assert.equal(claveI18nValida(RUTA_CANCELACIONES_EXPEDIENTE, "tras_fiscalizacion", "api.contratacion_temporal.cancelacion.error.tras_fiscalizacion", {}), true);
  assert.equal(claveI18nValida(RUTA_CANCELACIONES_EXPEDIENTE, "tras_fiscalizacion", "api.contratacion_temporal.seguimiento.error.tras_fiscalizacion", {}), false);
});

test("el historial nombra la actuación de cancelación", () => {
  assert.equal(presentarEtiquetasHitoRRHH({ accion_clave: "contratacion_temporal.expediente.cancelar", fase_origen: "solicitud", fase_destino: "solicitud",
    estado_origen: "en_curso", estado_destino: "cancelado" }).accion, "Expediente cancelado");
  assert.equal(presentarEtiquetasHitoRRHH({ accion_clave: "contratacion_temporal.expediente.cancelar", fase_origen: "solicitud", fase_destino: "solicitud",
    estado_origen: "en_curso", estado_destino: "cancelado" }).estadoDestino, "Cancelado");
});

test("el contexto sale del detalle vigente y de su fila, en cualquier fase", () => {
  const estado = (fase, version = 3, estadoClave = "en_curso") => ({ vista: "expediente", carga: "listo", expediente: { demostracion: false, expediente_ref: EXP, version: 3 },
    cuadro: { expedientes: [{ expediente_ref: EXP, version, fase_clave: fase, estado_clave: estadoClave }] } });
  assert.deepEqual({ ...contextoCancelacionDesdeEstado(estado("asignacion_unidad")) }, { expediente_ref: EXP, version: 3, fase_clave: "asignacion_unidad", estado_clave: "en_curso" });
  assert.equal(contextoCancelacionDesdeEstado(estado("asignacion_unidad", 2)), null, "fila desfasada");
  assert.equal(contextoCancelacionDesdeEstado({ ...estado("solicitud"), vista: "cuadro" }), null);
});

function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", hidden: false, eventos, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}
const esperar = () => new Promise((r) => setImmediate(r));
const contextoPanel = (fase = "asignacion_unidad", estado = "en_curso") => Object.freeze({ expediente_ref: EXP, version: 3, fase_clave: fase, estado_clave: estado });
const pulsar = (c, selector) => c.eventos.get("click")({ target: { closest: () => ({ matches: (s) => s === selector, setAttribute() {} }) } });

test("solo se ofrece en curso y en una fase admitida; el formulario se abre con el botón y pide motivo", async () => {
  const c = contenedorFalso();
  const cliente = { consultarCancelacion: async () => validarConsultaCancelacion(consulta(), EXP) };
  let desmontar = montarPanelCancelacion({ contenedor: c, cliente, contexto: contextoPanel() });
  assert.match(c.innerHTML, /aria-busy="true"/u);
  await esperar();
  assert.match(c.innerHTML, /data-ct-cancelacion-abrir aria-expanded="false"[^>]*>Cancelar expediente<\/button>/u);
  assert.doesNotMatch(c.innerHTML, /data-ct-cancelacion-form/u, "el formulario no se muestra hasta pulsar");
  assert.match(c.innerHTML, /aria-expanded="false"[^>]*>\?<\/button>/u, "la ayuda solo se abre con «?»");
  assert.match(c.innerHTML, /id="ct-cancelacion-ayuda"[^>]*hidden/u);
  pulsar(c, "[data-ct-cancelacion-abrir]");
  assert.match(c.innerHTML, /data-ct-cancelacion-form/u);
  assert.match(c.innerHTML, /<select name="motivo_clave" required\s*><option value=""><\/option><option value="necesidad_desaparecida">Ha desaparecido la necesidad/u);
  assert.match(c.innerHTML, /<textarea name="observaciones"[^>]*><\/textarea>/u);
  desmontar();
  assert.equal(c.eventos.size, 0);

  for (const [fase, estado] of [["fiscalizacion", "en_curso"], ["nombramiento", "en_curso"], ["solicitud", "incidencia"]]) {
    const otro = contenedorFalso();
    desmontar = montarPanelCancelacion({ contenedor: otro, cliente, contexto: contextoPanel(fase, estado) });
    await esperar();
    assert.equal(otro.innerHTML, "", `${fase}/${estado} no ofrece cancelar`);
    assert.equal(otro.hidden, true);
    desmontar();
  }
});

test("un expediente cancelado muestra quién, por qué y cuándo, sin ninguna acción ni la referencia a la vista", async () => {
  const c = contenedorFalso();
  const cliente = { consultarCancelacion: async () => validarConsultaCancelacion(consulta({ ...registrada }), EXP) };
  const desmontar = montarPanelCancelacion({ contenedor: c, cliente, contexto: contextoPanel("solicitud", "cancelado") });
  await esperar();
  assert.match(c.innerHTML, /Expediente cancelado por el centro solicitante: El centro retira su petición\./u);
  assert.match(c.innerHTML, /class="ct-exp-chip ct-fase-cancelado">Cancelado</u);
  assert.match(c.innerHTML, /Se reincorpora la titular/u);
  assert.match(c.innerHTML, /<dd>Solicitud<\/dd>/u);
  assert.doesNotMatch(c.innerHTML, /data-ct-cancelacion-abrir|data-ct-cancelacion-form/u);
  assert.doesNotMatch(c.innerHTML, />recibo:ct-seguimiento:9</u, "la referencia solo viaja en el botón de copiar");
  assert.match(c.innerHTML, /data-copiar-justificante="recibo:ct-seguimiento:9"/u);
  desmontar();
});

test("tras cancelar pinta el justificante copiable y lo conserva al volver a montarse; los rechazos se explican", async () => {
  const c = contenedorFalso();
  let fallo = null;
  const enviadas = [];
  const cliente = { consultarCancelacion: async () => validarConsultaCancelacion(consulta(), EXP),
    cancelarExpediente: async (s) => { enviadas.push(s); if (fallo) throw fallo; return { ...recibo }; } };
  let recibido = null;
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.f = f; } entries() { return Object.entries(this.f.campos); } };
  try {
    const desmontar = montarPanelCancelacion({ contenedor: c, cliente, contexto: contextoPanel(), confirmarOperacion: () => true,
      generarClave: () => UUID, alConfirmar: (r, aviso) => { recibido = { r, aviso }; } });
    await esperar();
    const enviar = (campos) => c.eventos.get("submit")({ target: { closest: () => ({ campos }) }, preventDefault() {} });
    enviar({ motivo_clave: "", observaciones: "" });
    await esperar();
    assert.equal(enviadas.length, 0, "sin motivo no se envía");
    assert.match(c.innerHTML, /elija un motivo/u);
    fallo = { envelopeValido: true, estado: 409, codigo: "tras_fiscalizacion" };
    enviar({ motivo_clave: "falta_credito", observaciones: " Sin partida " });
    await esperar();
    assert.match(c.innerHTML, /ya pasó por fiscalización y no se puede cancelar/u);
    assert.equal(enviadas[0].observaciones, "Sin partida");
    fallo = null;
    enviar({ motivo_clave: "falta_credito", observaciones: "" });
    await esperar();
    assert.match(c.innerHTML, /data-ct-cancelacion-aviso[^>]*>Expediente cancelado\./u);
    assert.match(c.innerHTML, /data-copiar-justificante="recibo:ct-seguimiento:1"/u);
    assert.equal(recibido?.aviso.recibo, "recibo:ct-seguimiento:1");
    desmontar();
    const otro = contenedorFalso();
    const otroDesmontar = montarPanelCancelacion({ contenedor: otro, cliente: { consultarCancelacion: async () => validarConsultaCancelacion(consulta({ ...registrada, canal: "rrhh" }), EXP) },
      contexto: contextoPanel("asignacion_unidad", "cancelado"), avisoInicial: recibido.aviso });
    await esperar();
    assert.match(otro.innerHTML, /Expediente cancelado por Recursos Humanos/u);
    assert.match(otro.innerHTML, /data-copiar-justificante="recibo:ct-seguimiento:1"/u);
    otroDesmontar();
  } finally {
    globalThis.FormData = FormDataOriginal;
  }
});

test("si el servidor no compone la cancelación (404 de ruta) el panel no se monta", async () => {
  const c = contenedorFalso();
  const cliente = { consultarCancelacion: async () => { throw { estado: 404, codigo: "respuesta_error_no_valida", envelopeValido: false }; } };
  const desmontar = montarPanelCancelacion({ contenedor: c, cliente, contexto: contextoPanel() });
  await esperar();
  assert.equal(c.innerHTML, "");
  assert.equal(c.hidden, true);
  desmontar();
  assert.equal(rutaCancelacionNoMontada({ estado: 404, envelopeValido: true }), false);
});

test("la bandeja y el cuadro muestran el expediente cancelado con su estado legible", async () => {
  const { crearAdaptadorHTTPExpedientesContratacionTemporal } = await import("./adaptador-http-expedientes.js");
  const { renderizarModuloContratacionTemporal } = await import("./vista-expedientes.js");
  const fila = { expediente_ref: EXP, numero_visible: "2026/CT-0042", version: 4, flujo_ref: "flujo:ct:desarrollo", flujo_version: 1,
    flujo_huella_sha256: "a".repeat(64), fase_clave: "asignacion_unidad", estado_clave: "cancelado", centro_ref: "centro:desarrollo:001",
    categoria_ref: "categoria:desarrollo:c2", modalidad_clave: "sustitucion", unidad_ref: "unidad:desarrollo:rrhh",
    creado_en: "2026-09-20T08:00:00Z", actualizado_en: "2026-09-26T10:00:00Z" };
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: {
    consultarCuadroRRHH: async () => ({ esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-26T10:05:00Z", expedientes: [fila], hay_mas: false }),
    consultarDetalleRRHH: async () => { throw new Error("sin detalle"); } } });
  const cuadro = await adaptador.listar({ filtros: { texto: "", estado: "", fase: "" } });
  assert.equal(cuadro.expedientes[0].estado_clave, "cancelado");
  assert.equal(cuadro.expedientes[0].estado, "Cancelado");
  const html = renderizarModuloContratacionTemporal({ vista: "cuadro", carga: "listo", cuadro, filtros: { texto: "", estado: "", fase: "" } }, {});
  assert.match(html, /Cancelado/u);
});
