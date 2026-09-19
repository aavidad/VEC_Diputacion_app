import assert from "node:assert/strict";
import test from "node:test";

import { cerrarFase, construirPanelFase, instalarPantallasFase } from "./fases-expediente.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

test("construirPanelFase conserva la cabecera accesible del historial", () => {
  const originalDocument = globalThis.document;
  globalThis.document = {
    createElement() {
      return {
        className: "",
        innerHTML: "",
        setAttribute() {},
      };
    },
  };
  try {
    const boton = {
      dataset: { ctExpFaseVer: "solicitud" },
      closest(selector) {
        if (selector !== "li") return null;
        return {
          className: "ct-fase-en_curso",
          querySelector(selectorHijo) {
            if (selectorHijo === "span:not(.ct-exp-numero-fase)") return { textContent: "Solicitud" };
            if (selectorHijo === "small") return { textContent: "En curso" };
            if (selectorHijo === ".ct-exp-numero-fase") return { textContent: "1" };
            return null;
          },
        };
      },
    };
    const hito = {
      dataset: { ctExpHitoAccion: "alta", ctExpHitoFase: "Solicitud" },
      outerHTML: "<tr><td>1</td><td>2026-09-19</td><td>Alta</td><td>Solicitud</td><td>Registrado</td></tr>",
    };
    const contenido = {
      querySelectorAll(selector) {
        if (selector === '[data-ct-exp-campo-fase="solicitud"]') return [];
        if (selector === "[data-ct-exp-hito-fase]") return [hito];
        return [];
      },
    };
    const panel = construirPanelFase(contenido, boton, crearTraductorExpedientesContratacion());

    assert.match(panel.innerHTML, /tabindex="0" role="region" aria-label="Historial de actuaciones"/u);
    assert.match(panel.innerHTML, /<caption>Historial de actuaciones<\/caption>/u);
    assert.match(panel.innerHTML, /<thead><tr><th scope="col">Secuencia<\/th><th scope="col">Fecha y hora<\/th><th scope="col">Actuación<\/th><th scope="col">Fase registrada<\/th><th scope="col">Estado registrado<\/th><\/tr><\/thead>/u);
    assert.match(panel.innerHTML, /<tbody><tr><td>1<\/td>/u);
    const panelSinDocumentos = construirPanelFase(
      contenido, boton, crearTraductorExpedientesContratacion(), { documentos: false },
    );
    assert.doesNotMatch(panelSinDocumentos.innerHTML, /data-ct-exp-vista="documentos"/u);
  } finally {
    globalThis.document = originalDocument;
  }
});

test("cerrarFase devuelve el foco al botón de la fase abierta", () => {
  let foco = 0;
  let eliminado = false;
  const faseAbierta = {
    setAttribute(nombre, valor) { this[nombre] = valor; },
    focus() { foco += 1; },
  };
  const contenido = {
    querySelector(selector) {
      if (selector === '[data-ct-exp-fase-ver][aria-pressed="true"]') return faseAbierta;
      if (selector === ".ct-exp-fase-panel") return { remove() { eliminado = true; } };
      return null;
    },
    querySelectorAll(selector) {
      assert.equal(selector, "[data-ct-exp-fase-ver]");
      return [faseAbierta];
    },
    contains(elemento) { return elemento === faseAbierta; },
  };
  const cerrar = { closest(selector) { return selector === ".ct-exp-contenido" ? contenido : null; } };

  assert.equal(cerrarFase(cerrar), true);
  assert.equal(eliminado, true);
  assert.equal(faseAbierta["aria-pressed"], "false");
  assert.equal(foco, 1);
});

test("delegado de fase usa sobrescritura i18n al abrir con clic", () => {
  const originalDocument = globalThis.document;
  let manejarClick;
  const panel = {
    className: "", innerHTML: "", setAttribute() {},
    querySelector() { return { setAttribute() {}, focus() {} }; },
  };
  globalThis.document = { createElement() { return panel; } };
  const contenido = {
    querySelector() { return null; },
    querySelectorAll(selector) {
      if (selector === '[data-ct-exp-campo-fase="solicitud"]') return [];
      if (selector === "[data-ct-exp-hito-fase]") return [];
      return [];
    },
  };
  const rail = {
    querySelectorAll() { return [boton]; },
    insertAdjacentElement(posicion, elemento) {
      assert.equal(posicion, "afterend");
      assert.equal(elemento, panel);
    },
  };
  const boton = {
    dataset: { ctExpFaseVer: "solicitud" },
    setAttribute() {},
    closest(selector) {
      if (selector === ".ct-exp-contenido") return contenido;
      if (selector === ".ct-exp-progreso") return rail;
      if (selector === "li") return {
        className: "ct-fase-en_curso",
        querySelector(hijo) {
          if (hijo === "span:not(.ct-exp-numero-fase)") return { textContent: "Solicitud" };
          if (hijo === "small") return { textContent: "En curso" };
          if (hijo === ".ct-exp-numero-fase") return { textContent: "1" };
          return null;
        },
      };
      return null;
    },
  };
  try {
    instalarPantallasFase({
      addEventListener(tipo, manejador) {
        assert.equal(tipo, "click");
        manejarClick = manejador;
      },
    }, crearTraductorExpedientesContratacion({ nav_documentos: "Documentos <del catálogo>" }));
    let prevenido = false;
    manejarClick({
      target: { closest: (selector) => selector === "[data-ct-exp-fase-ver]" ? boton : null },
      preventDefault() { prevenido = true; },
    });
    assert.equal(prevenido, true);
    assert.match(panel.innerHTML, /Documentos &lt;del catálogo&gt;/u);
  } finally {
    globalThis.document = originalDocument;
  }
});
