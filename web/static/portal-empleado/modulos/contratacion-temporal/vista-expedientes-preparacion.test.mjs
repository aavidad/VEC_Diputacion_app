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
  const zona = { innerHTML: "", eventos, enfocado: null,
    addEventListener(tipo, funcion) { eventos.set(tipo, funcion); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) { return { focus: () => { zona.enfocado = selector; } }; },
  };
  const alta = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const raiz = { querySelector(selector) {
    if (selector === "[data-ct-exp-preparacion]") return zona;
    if (selector === "[data-ct-exp-alta]") return alta;
    return null;
  } };
  return { zona, alta, raiz };
}

function gestorPara(superficieActual, catalogos) {
  return crearGestorTramitacion({
    raiz: superficieActual.raiz,
    presentador: { obtenerEstado: () => ({ vista: "alta" }) },
    altaDisponible: true,
    // Sin presentador de alta válido el formulario cae a su aviso propio: la
    // relación por vía no depende de él.
    alta: { catalogos, ejecutor: async () => {} },
  });
}

function pulsar(zona, via, type = "click", key) {
  zona.eventos.get(type)({ type, key, preventDefault() {},
    target: { closest: (selector) => (selector === "[data-ct-preparacion-pestana]"
      ? { dataset: { ctPreparacionPestana: via } } : null) } });
}

test("la nueva petición muestra la relación por vía que trae el catálogo del alta, sin otra consulta", () => {
  const actual = superficie();
  const gestor = gestorPara(actual, { preparacion_vias: catalogo() });
  gestor.montarAltaSiProcede();
  assert.match(actual.zona.innerHTML, /role="tablist"/u);
  assert.match(actual.zona.innerHTML, /id="ct-preparacion-alta-pestana-bolsa_vigente"[^>]*aria-selected="true"/u);
  assert.match(actual.zona.innerHTML, /No hace falta nada en este apartado/u);
  pulsar(actual.zona, "oferta_sae");
  assert.match(actual.zona.innerHTML, /id="ct-preparacion-alta-pestana-oferta_sae"[^>]*aria-selected="true"/u);
  assert.equal(actual.zona.enfocado, '[data-ct-preparacion-pestana="oferta_sae"]');
  pulsar(actual.zona, "oferta_sae", "keydown", "Home");
  assert.match(actual.zona.innerHTML, /id="ct-preparacion-alta-pestana-bolsa_vigente"[^>]*aria-selected="true"/u);
  gestor.retirarComponentes();
  assert.equal(actual.zona.innerHTML, "");
  assert.equal(actual.zona.eventos.size, 0);
});

test("sin relación en el catálogo del alta la zona queda vacía y volver a montar no duplica escuchas", () => {
  const actual = superficie();
  const gestor = gestorPara(actual, {});
  gestor.montarAltaSiProcede();
  assert.equal(actual.zona.innerHTML, "");
  assert.equal(actual.zona.eventos.size, 0);
  const con = superficie();
  const otro = gestorPara(con, { preparacion_vias: catalogo() });
  otro.montarAltaSiProcede();
  otro.montarAltaSiProcede();
  assert.equal(con.zona.eventos.size, 2);
  assert.match(con.zona.innerHTML, /data-ct-preparacion-vias/u);
  otro.retirarComponentes();
});
