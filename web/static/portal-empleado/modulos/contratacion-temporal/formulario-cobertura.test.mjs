import assert from "node:assert/strict";
import test from "node:test";

import { montarFormularioCobertura } from "./formulario-cobertura.js";

const EXPEDIENTE = "expediente:ct:prueba:cobertura:001";
const CLAVE = "11111111-1111-4111-8111-111111111111";
const HUELLA = "a".repeat(64);

function propuesta() {
  return {
    esquema: "vec.contratacion-temporal.propuesta-cobertura.v1",
    estado: "viable",
    via_recomendada: "bolsa_vigente",
    evaluaciones: [{
      via_clave: "bolsa_vigente", prioridad: 1, estado: "viable",
      resultados_omitidos: [], ausencias_bloqueantes: [],
      ausencias_admitidas: [], no_habilitantes: [], conflictos: [],
    }],
    identidad_semantica: {
      referencia: `propuesta-cobertura-semantica:sha256:${HUELLA}`,
      huella_sha256: HUELLA,
      canon: {
        dominio: "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica",
        version_esquema: 1,
        algoritmo: "sha-256",
      },
    },
  };
}

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-cobertura.v1",
    recibo_ref: "recibo:ct:cobertura:prueba:001",
    estado: "aplicada",
    decision_cobertura_ref: "decision-cobertura:sha256:" + "b".repeat(64),
    version_resultante: 3,
    confirmada_en: "2026-09-04T10:49:22.962178Z",
  };
}

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    eventos,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    enviar() {
      const formulario = {
        closest(selector) {
          return selector === "[data-ct-cobertura-form]" ? this : null;
        },
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
    recuperar() {
      const control = {
        dataset: { ctCoberturaAccion: "consultar-resultado" },
        closest(selector) {
          return selector === "[data-ct-cobertura-accion]" ? this : null;
        },
      };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

async function estabilizar() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

test("propuesta y decisión usan el recibo de Análisis y una sola clave", async () => {
  const raiz = raizFalsa();
  const propuestas = [];
  const decisiones = [];
  const continuaciones = [];
  let confirmaciones = 0;
  const cliente = {
    async proponerCobertura(solicitud) {
      propuestas.push(solicitud);
      return propuesta();
    },
    async decidirCobertura(solicitud) {
      decisiones.push(solicitud);
      return recibo();
    },
    async consultarResultadoCobertura() {
      throw new Error("consulta inesperada");
    },
  };
  const desmontar = montarFormularioCobertura({
    raiz,
    cliente,
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 2 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion() { confirmaciones += 1; return true; },
    alConfirmar(confirmado) { continuaciones.push(confirmado); return true; },
  });
  await estabilizar();
  assert.deepEqual(propuestas, [{ expediente_ref: EXPEDIENTE, version_esperada: 2 }]);
  assert.match(raiz.innerHTML, /data-ct-cobertura-via="bolsa_vigente"/);

  await Promise.all([raiz.enviar(), raiz.enviar()]);
  assert.equal(confirmaciones, 1);
  assert.equal(decisiones.length, 1);
  assert.deepEqual(decisiones[0], {
    expediente_ref: EXPEDIENTE,
    version_esperada: 2,
    clave_idempotencia: CLAVE,
    identidad_semantica: propuesta().identidad_semantica,
    via_elegida: "bolsa_vigente",
    motivo_clave: "",
  });
  assert.match(raiz.innerHTML, /data-ct-cobertura-recibo/);
  assert.match(raiz.innerHTML, /recibo:ct:cobertura:prueba:001/);
  assert.deepEqual(continuaciones, [recibo()]);
  desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("cancelar no decide y el resultado indeterminado solo se consulta", async () => {
  const raizCancelada = raizFalsa();
  let decisionesCanceladas = 0;
  montarFormularioCobertura({
    raiz: raizCancelada,
    cliente: {
      async proponerCobertura() { return propuesta(); },
      async decidirCobertura() { decisionesCanceladas += 1; return recibo(); },
      async consultarResultadoCobertura() { return { esquema: "", estado: "" }; },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 2 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion: () => false,
  });
  await estabilizar();
  await raizCancelada.enviar();
  assert.equal(decisionesCanceladas, 0);

  const raiz = raizFalsa();
  let decisiones = 0;
  const consultas = [];
  const continuaciones = [];
  const error = new Error("resultado privado indeterminado");
  error.resultadoIndeterminado = true;
  montarFormularioCobertura({
    raiz,
    cliente: {
      async proponerCobertura() { return propuesta(); },
      async decidirCobertura() { decisiones += 1; throw error; },
      async consultarResultadoCobertura(solicitud) {
        consultas.push(solicitud);
        return {
          esquema: "vec.contratacion-temporal.resultado-consulta-cobertura.v1",
          estado: "confirmado",
          recibo: recibo(),
        };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 2 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion: () => true,
    alConfirmar(confirmado) { continuaciones.push(confirmado); return true; },
  });
  await estabilizar();
  await raiz.enviar();
  assert.equal(decisiones, 1);
  assert.match(raiz.innerHTML, /data-ct-cobertura-accion="consultar-resultado"/);
  await raiz.recuperar();
  assert.equal(decisiones, 1);
  assert.deepEqual(consultas, [{
    expediente_ref: EXPEDIENTE,
    clave_idempotencia: CLAVE,
  }]);
  assert.match(raiz.innerHTML, /data-ct-cobertura-recibo/);
  assert.deepEqual(continuaciones, [recibo()]);
});

test("conserva el recibo y avisa si la asignación no puede montarse", async () => {
  const raiz = raizFalsa();
  montarFormularioCobertura({
    raiz,
    cliente: {
      async proponerCobertura() { return propuesta(); },
      async decidirCobertura() { return recibo(); },
      async consultarResultadoCobertura() { throw new Error("consulta inesperada"); },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 2 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion: () => true,
    alConfirmar: () => false,
  });
  await estabilizar();
  await raiz.enviar();

  assert.match(raiz.innerHTML, /data-ct-cobertura-recibo/u);
  assert.match(raiz.innerHTML, /La cobertura está confirmada, pero la asignación no está disponible/u);
});
