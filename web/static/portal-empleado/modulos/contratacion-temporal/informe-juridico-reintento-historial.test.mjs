import assert from "node:assert/strict";
import test from "node:test";

import { montarFormularioInformeJuridico } from "./formulario-informe-juridico.js";

const expediente = "expediente:ct:sintetico:informe-001";
const recibo = Object.freeze({
  esquema: "vec.contratacion-temporal.recibo-informe-juridico.v1",
  operacion: "preparar",
  expediente_ref: expediente,
  version_resultante: 5,
  informe_ref: "informe:ct:sintetico-001",
  documento_ref: "documento:ct:sintetico-001",
  version_documento: 1,
  formato: "text/plain; charset=utf-8",
  nombre: "informe-juridico-desarrollo.txt",
  huella_documento_sha256: "a".repeat(64),
  recibo_ref: "recibo:ct:informe:sintetico-001",
  auditoria_ref: "auditoria:ct:informe:sintetico-001",
  evento_ref: "evento:ct:informe:sintetico-001",
  contenido_desarrollo: "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA\n",
  confirmada_en: "2026-09-04T18:00:00Z",
});

const detalle = Object.freeze({
  resumen: { expediente_ref: expediente, version: 5 },
  hitos: [{
    secuencia: 5, version_expediente: 5,
    accion_clave: "contratacion_temporal.informe_juridico.generar",
    fase_origen: "asignacion_unidad", fase_destino: "informe_juridico",
    estado_origen: "en_curso", estado_destino: "en_curso",
    realizada_en: recibo.confirmada_en,
  }],
});

function crearRaiz() {
  const eventos = new Map();
  const ownerDocument = { activeElement: null };
  const foco = { llamadas: 0, scrolls: 0 };
  const regionHistorial = {
    focus() { foco.llamadas += 1; ownerDocument.activeElement = this; },
    scrollIntoView() { foco.scrolls += 1; },
    contains(elemento) { return elemento === this; },
  };
  return {
    innerHTML: "",
    ownerDocument,
    foco,
    regionHistorial,
    addEventListener(nombre, funcion) { eventos.set(nombre, funcion); },
    removeEventListener(nombre, funcion) {
      if (eventos.get(nombre) === funcion) eventos.delete(nombre);
    },
    contains() { return true; },
    querySelector(selector) {
      return selector === "[data-ct-informe-historial]"
        ? regionHistorial : { focus() {}, scrollIntoView() {} };
    },
    replaceChildren() { this.innerHTML = ""; },
    enviar() {
      const formulario = {
        closest() { return this; },
        checkValidity() { return true; },
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
    reintentar() {
      const control = {
        dataset: { ctInformeAccion: "reintentar-historial" },
        closest() { return this; },
      };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
    eventos,
  };
}

function montar(raiz, cliente) {
  return montarFormularioInformeJuridico({
    raiz, cliente,
    contexto: { expediente_ref: expediente, version_esperada: 4 },
    generarClaveIdempotencia: () => "123e4567-e89b-42d3-a456-426614174000",
    confirmarOperacion: () => true,
  });
}

test("informe confirmado, consulta de historial fallida y reintento conserva recibo y documento", async () => {
  const raiz = crearRaiz();
  let posts = 0;
  const consultas = [];
  let resolverReintento;
  const desmontar = montar(raiz, {
    async prepararInformeJuridico() { posts += 1; return recibo; },
    consultarDetalleRRHH(solicitud, opciones) {
      consultas.push({ solicitud, signal: opciones.signal });
      if (consultas.length === 1) return Promise.reject(new Error("red temporal"));
      return new Promise((resolve) => { resolverReintento = resolve; });
    },
  });

  await raiz.enviar();
  assert.equal(posts, 1);
  assert.equal(consultas.length, 1);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.match(raiz.innerHTML, /data-ct-informe-documento/u);
  assert.match(raiz.innerHTML, /data-ct-informe-historial tabindex="-1"/u);
  assert.match(raiz.innerHTML, /data-ct-informe-accion="reintentar-historial"/u);
  assert.match(raiz.innerHTML, /recibo:ct:informe:sintetico-001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-form/u);

  const primero = raiz.reintentar();
  const segundo = raiz.reintentar();
  assert.equal(primero, segundo);
  assert.equal(posts, 1);
  assert.equal(consultas.length, 2);
  assert.deepEqual(consultas[1].solicitud, {
    expediente_ref: recibo.expediente_ref,
    version_observada: recibo.version_resultante,
  });
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.match(raiz.innerHTML, /data-ct-informe-documento/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-accion="reintentar-historial"/u);

  resolverReintento(detalle);
  await primero;
  assert.equal(posts, 1);
  assert.equal(consultas.length, 2);
  assert.match(raiz.innerHTML, /Historial persistido del expediente/u);
  assert.match(raiz.innerHTML, /data-ct-informe-historial tabindex="-1"/u);
  assert.match(raiz.innerHTML, /<tbody>[\s\S]*Informe jurídico generado[\s\S]*<\/tbody>/u);
  assert.match(raiz.innerHTML, /recibo:ct:informe:sintetico-001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-historial-error/u);
  assert.doesNotMatch(raiz.innerHTML, /Recuperando el historial actualizado/u);
  assert.equal(raiz.foco.llamadas, 2);
  assert.equal(raiz.foco.scrolls, 2);
  desmontar();
});

test("la consulta diferida no recupera foco ni desplaza si se pasó a otro expediente", async () => {
  const raiz = crearRaiz();
  let resolverReintento;
  let posts = 0;
  let consultas = 0;
  const desmontar = montar(raiz, {
    async prepararInformeJuridico() { posts += 1; return recibo; },
    consultarDetalleRRHH() {
      consultas += 1;
      if (consultas === 1) return Promise.reject(new Error("red temporal"));
      return new Promise((resolve) => { resolverReintento = resolve; });
    },
  });
  await raiz.enviar();
  const reintento = raiz.reintentar();
  assert.equal(raiz.ownerDocument.activeElement, raiz.regionHistorial);
  const focoAlIniciar = { ...raiz.foco };
  const otroExpediente = { id: "otro-expediente" };
  raiz.ownerDocument.activeElement = otroExpediente;

  resolverReintento(detalle);
  await reintento;
  assert.equal(raiz.ownerDocument.activeElement, otroExpediente);
  assert.deepEqual(raiz.foco, focoAlIniciar);
  assert.match(raiz.innerHTML, /Historial persistido del expediente/u);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.equal(posts, 1);
  assert.equal(consultas, 2);
  desmontar();
});

test("desmontar durante el reintento aborta la consulta e ignora la respuesta tardía", async () => {
  const raiz = crearRaiz();
  let posts = 0;
  let getSignal;
  let resolverReintento;
  let gets = 0;
  const desmontar = montar(raiz, {
    async prepararInformeJuridico() { posts += 1; return recibo; },
    consultarDetalleRRHH(_solicitud, opciones) {
      gets += 1;
      if (gets === 1) return Promise.reject(new Error("red temporal"));
      getSignal = opciones.signal;
      return new Promise((resolve) => { resolverReintento = resolve; });
    },
  });
  await raiz.enviar();
  const reintento = raiz.reintentar();
  assert.equal(getSignal.aborted, false);
  desmontar();
  assert.equal(getSignal.aborted, true);
  assert.equal(raiz.eventos.size, 0);
  resolverReintento(detalle);
  await reintento;
  assert.equal(raiz.innerHTML, "");
  assert.equal(posts, 1);
  assert.equal(gets, 2);
});
