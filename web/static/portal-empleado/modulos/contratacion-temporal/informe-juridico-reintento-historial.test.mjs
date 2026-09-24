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
  return {
    innerHTML: "",
    addEventListener(nombre, funcion) { eventos.set(nombre, funcion); },
    removeEventListener(nombre, funcion) {
      if (eventos.get(nombre) === funcion) eventos.delete(nombre);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
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
  assert.match(raiz.innerHTML, /<tbody>[\s\S]*Informe jurídico generado[\s\S]*<\/tbody>/u);
  assert.match(raiz.innerHTML, /recibo:ct:informe:sintetico-001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-historial-error/u);
  assert.doesNotMatch(raiz.innerHTML, /Recuperando el historial actualizado/u);
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
