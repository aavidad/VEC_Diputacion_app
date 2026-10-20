import assert from "node:assert/strict";
import test from "node:test";

import { presentarEtiquetasHitoRRHH } from "./adaptador-http-expedientes.js?v=VEC_WEB_VERSION";
import { montarFormularioInformeJuridico } from "./formulario-informe-juridico.js?v=VEC_WEB_VERSION";
import { cargarMensajesContratacionTemporalEnIdioma } from "./i18n.js?v=VEC_WEB_VERSION";
import { crearTraductorExpedientesContratacion, cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js?v=VEC_WEB_VERSION";
import { renderizarBorradoresPublicados } from "./vista-borradores-publicados.js?v=VEC_WEB_VERSION";
import { renderizarSiguientePasoFicha } from "./vista-expedientes-ficha.js?v=VEC_WEB_VERSION";

const expedienteRef = "expediente:ct:sintetico:informe-001";
const recibo = Object.freeze({
  esquema: "vec.contratacion-temporal.recibo-informe-juridico.v1",
  operacion: "preparar", expediente_ref: expedienteRef, version_resultante: 5,
  informe_ref: "informe:ct:sintetico-001", documento_ref: "documento:ct:sintetico-001", version_documento: 1,
  formato: "text/plain; charset=utf-8", nombre: "informe-juridico-desarrollo.txt",
  huella_documento_sha256: "a".repeat(64), recibo_ref: "recibo:ct:informe:sintetico-001",
  auditoria_ref: "auditoria:ct:informe:sintetico-001", evento_ref: "evento:ct:informe:sintetico-001",
  contenido_desarrollo: "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA\n", confirmada_en: "2026-09-17T00:03:32Z",
});
const hito = Object.freeze({
  secuencia: 5, version_expediente: 5, accion_clave: "contratacion_temporal.informe_juridico.generar",
  fase_origen: "asignacion_unidad", fase_destino: "informe_juridico",
  estado_origen: "en_curso", estado_destino: "en_curso", realizada_en: recibo.confirmada_en,
});

function crearRaiz() {
  const eventos = new Map();
  return {
    innerHTML: "", ownerDocument: { activeElement: null },
    addEventListener(nombre, funcion) { eventos.set(nombre, funcion); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    enviar() {
      const formulario = { closest() { return this; }, checkValidity() { return true; } };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

function montar(raiz, { mensajes = {}, locale = "es-ES", confirmaciones = [] } = {}) {
  return montarFormularioInformeJuridico({
    raiz, mensajes, locale,
    cliente: {
      async prepararInformeJuridico() { return recibo; },
      async consultarDetalleRRHH() { return { resumen: { expediente_ref: expedienteRef, version: 5 }, hitos: [hito] }; },
    },
    contexto: { expediente_ref: expedienteRef, version_esperada: 4, numero_visible: "2026/94010" },
    generarClaveIdempotencia: () => "123e4567-e89b-42d3-a456-426614174000",
    confirmarOperacion: (descriptor) => { confirmaciones.push(descriptor); return true; },
  });
}

test("la confirmación del informe y su diálogo nombran el expediente por su número, sin referencia interna", async () => {
  const raiz = crearRaiz();
  const confirmaciones = [];
  const desmontar = montar(raiz, { confirmaciones });
  // F1-88: la confirmación nombra solo el número; ni versión ni referencia interna.
  assert.match(raiz.innerHTML, /Expediente 2026\/94010\./u);
  assert.doesNotMatch(raiz.innerHTML, /expediente:ct:|versión 4/u);
  await raiz.enviar();
  assert.equal(confirmaciones[0].referencia, "2026/94010");
  // Sin tipo MIME a la vista; la fecha, como en la ficha; la fase, con su paso.
  assert.doesNotMatch(raiz.innerHTML, /text\/plain|charset/u);
  assert.match(raiz.innerHTML, /17 sept 2026, 2:03:32/u);
  assert.match(raiz.innerHTML, /Gestión de bolsa · Asignación de unidad → Gestión de bolsa · Informe jurídico/u);
  desmontar();
});

test("en inglés la confirmación usa el número del expediente", async () => {
  const raiz = crearRaiz();
  const confirmaciones = [];
  const desmontar = montar(raiz, {
    mensajes: await cargarMensajesContratacionTemporalEnIdioma("en"), locale: "en-GB", confirmaciones,
  });
  assert.match(raiz.innerHTML, /Case 2026\/94010\./u);
  assert.doesNotMatch(raiz.innerHTML, /expediente:ct:|version 4/u);
  await raiz.enviar();
  assert.equal(confirmaciones[0].referencia, "2026/94010");
  desmontar();
});

test("sin número visible la confirmación dice «Expediente seleccionado» sin referencia interna", () => {
  const raiz = crearRaiz();
  const desmontar = montarFormularioInformeJuridico({
    raiz, cliente: { async prepararInformeJuridico() {}, async consultarDetalleRRHH() {} },
    contexto: { expediente_ref: expedienteRef, version_esperada: 4 },
  });
  assert.match(raiz.innerHTML, /Expediente seleccionado\./u);
  assert.doesNotMatch(raiz.innerHTML, /expediente:ct:|versión asignada/u);
  desmontar();
  assert.throws(() => montarFormularioInformeJuridico({
    raiz: crearRaiz(), cliente: { async prepararInformeJuridico() {}, async consultarDetalleRRHH() {} },
    contexto: { expediente_ref: expedienteRef, version_esperada: 4, numero_visible: " " },
  }), TypeError);
});

test("los dos historiales nombran la fase igual que la cabecera, sin «Fase N de 8»", async () => {
  assert.deepEqual(
    [presentarEtiquetasHitoRRHH(hito).faseOrigen, presentarEtiquetasHitoRRHH(hito).faseDestino],
    ["Gestión de bolsa · Asignación de unidad", "Gestión de bolsa · Informe jurídico"],
  );
  const en = presentarEtiquetasHitoRRHH(hito, await cargarMensajesExpedientesContratacionEnIdioma("en"));
  assert.equal(en.faseDestino, "Recruitment pool · Legal report");
  assert.equal(presentarEtiquetasHitoRRHH({ ...hito, fase_destino: "fiscalizacion" }).faseDestino, "Fiscalización");
});

const expediente = {
  expediente_ref: expedienteRef, version: 4, tareas: [],
  fases: [{ etiqueta: "Gestión de bolsa", estado_clave: "en_curso" }],
  historial: [{ accion_clave: "contratacion_temporal.unidad.asignar", version_expediente: 4 }],
};
const estado = {
  expediente_ref: expedienteRef, carga: "listo",
  cuadro: { expedientes: [{ expediente_ref: expedienteRef, version: 4, fase_clave: "asignacion_unidad", estado_clave: "en_curso" }] },
};

test("asignada la unidad, el siguiente paso es preparar el informe jurídico", async () => {
  const html = renderizarSiguientePasoFicha(expediente, estado, crearTraductorExpedientesContratacion(), false,
    { informePendiente: true });
  assert.match(html, /Siguiente paso: preparar el informe jurídico/u);
  assert.match(html, /apartado «Preparar informe jurídico»/u);
  assert.match(html, /data-ct-exp-accion="ir-tramite"/u);
  const en = renderizarSiguientePasoFicha(expediente, estado,
    crearTraductorExpedientesContratacion(await cargarMensajesExpedientesContratacionEnIdioma("en")), false,
    { informePendiente: true });
  assert.match(en, /Next step: prepare the legal report/u);
  assert.match(en, /the “Prepare legal report” section/u);
  assert.doesNotMatch(renderizarSiguientePasoFicha(expediente, { ...estado, ocupado: true },
    crearTraductorExpedientesContratacion(), false, { informePendiente: true }), /preparar el informe jurídico/u);
});

test("con el recibo del informe el siguiente paso cambia en el momento, aunque el cuadro siga en v4", () => {
  const html = renderizarSiguientePasoFicha(expediente, estado, crearTraductorExpedientesContratacion(), false,
    { informePendiente: true, informeConfirmado: true });
  assert.match(html, /Siguiente paso: revisar las firmas del informe/u);
  assert.doesNotMatch(html, /preparar el informe jurídico|data-ct-exp-accion="ir-tramite"/u);
});

test("los documentos generados se nombran con el catálogo del idioma y un tipo nuevo conserva el del servidor", () => {
  const html = renderizarBorradoresPublicados({ estado: "lista", catalogo: {
    procedencia_ref: "recibo:f196", tipos: [
      { clave: "informe_definitivo", etiqueta: "Del servidor", formatos: ["pdf"], disponible: true },
      { clave: "tipo_nuevo", etiqueta: "Tipo publicado nuevo", formatos: ["pdf"], disponible: true },
    ] } });
  assert.match(html, />Informe definitivo</u);
  assert.match(html, />Tipo publicado nuevo</u);
  assert.doesNotMatch(html, /Del servidor/u);
});
