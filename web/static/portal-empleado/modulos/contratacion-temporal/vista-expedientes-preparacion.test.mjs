import assert from "node:assert/strict";
import test from "node:test";

import { crearGestorTramitacion } from "./vista-expedientes-tramitacion.js?v=20261001-ct-a-i18n-v1";

const HUELLA = "b".repeat(64);

function catalogo() {
  return { referencia: "catalogo:ct:preparacion:v3", version: 3, huella_sha256: HUELLA, es_ejemplo: true,
    vias: [
      { clave: "bolsa_vigente", orden: 1, documentos: [], datos: [
        { clave: "categoria", orden: 1, clave_i18n: "contratacion_temporal.cobertura.dato.categoria" }] },
      { clave: "oferta_sae", orden: 2, documentos: [
        { clave: "nota_sae", orden: 1, clave_i18n: "contratacion_temporal.cobertura.doc.nota_informativa_sae" }], datos: [] },
    ] };
}

function superficie() {
  const eventos = new Map();
  const consultas = [];
  const zona = { innerHTML: "", eventos,
    addEventListener(tipo, funcion) { eventos.set(tipo, funcion); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
  };
  const alta = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const raiz = { querySelector(selector) {
    consultas.push(selector);
    if (selector === "[data-ct-exp-preparacion]") return zona;
    if (selector === "[data-ct-exp-alta]") return alta;
    return null;
  } };
  return { zona, alta, raiz, consultas };
}

function gestorPara(superficieActual, catalogos) {
  return crearGestorTramitacion({
    raiz: superficieActual.raiz,
    presentador: { obtenerEstado: () => ({ vista: "alta" }) },
    altaDisponible: true,
    alta: { catalogos, ejecutor: async () => {} },
  });
}

test("el alta no monta la relación de cobertura de ejemplo ni sus escuchas", () => {
  const actual = superficie();
  const gestor = gestorPara(actual, { preparacion_vias: catalogo() });
  gestor.montarAltaSiProcede();
  assert.ok(actual.consultas.includes("[data-ct-exp-alta]"));
  assert.ok(!actual.consultas.includes("[data-ct-exp-preparacion]"));
  assert.equal(actual.zona.innerHTML, "");
  assert.equal(actual.zona.eventos.size, 0);
  gestor.montarAltaSiProcede();
  assert.equal(actual.zona.eventos.size, 0);
  gestor.retirarComponentes();
  assert.equal(actual.zona.eventos.size, 0);
});

test("sin relación por vía, el error del formulario de alta sigue visible y recuperable", () => {
  const actual = superficie();
  const gestor = gestorPara(actual, {});
  gestor.montarAltaSiProcede();
  assert.equal(actual.zona.innerHTML, "");
  assert.equal(actual.zona.eventos.size, 0);
  assert.match(actual.alta.innerHTML, /data-ct-exp-accion="reintentar"/u);
  gestor.retirarComponentes();
});
