import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { CATALOGO_PORTADA, aplicarIdiomaPortada, textoPortada } from "./portada-i18n.js";

test("la portada tiene paridad ES/EN y claves para su HTML", () => {
  assert.deepEqual(Object.keys(CATALOGO_PORTADA.es).sort(), Object.keys(CATALOGO_PORTADA.en).sort());
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  for (const [, clave] of html.matchAll(/data-i18n(?:-[\w-]+)?="([\w-]+)"/g)) {
    if (clave === "true") continue; // Identificadores de muestra, invariables.
    assert.ok(CATALOGO_PORTADA.es[clave], `falta español: ${clave}`);
    assert.ok(CATALOGO_PORTADA.en[clave], `falta inglés: ${clave}`);
  }
  assert.match(html, /<label[^>]+for="vec-language"/);
  assert.match(html, /<select[^>]+id="vec-language"/);
});

test("aplica inglés a texto, atributo y título sin insertar HTML", () => {
  const nodos = [{ nodeValue: "  Portal corporativo  " }, { nodeValue: "Preparando demo" }];
  const textos = nodos.map((nodo, indice) => ({
    dataset: { i18n: indice === 0 ? "portal_corporativo" : "preparando" },
    childNodes: [{ ...nodo, nodeType: 3 }],
  }));
  const atributos = new Map([["aria-label", "Bandeja principal"]]);
  const elemento = {
    getAttribute: (nombre) => nombre === "data-i18n-aria-label" ? "bandeja_principal" : atributos.get(nombre),
    setAttribute: (nombre, valor) => atributos.set(nombre, valor),
  };
  const selector = {
    value: "",
    addEventListener: () => {},
    setAttribute: () => {},
  };
  const raiz = { querySelectorAll: (consulta) => consulta === "[data-i18n]" ? textos : [elemento] };
  const documento = {
    documentElement: { lang: "es" },
    title: "",
    querySelector: (selectorCss) => selectorCss === ".workspace" ? raiz : selector,
  };
  aplicarIdiomaPortada(documento, "en");
  assert.equal(documento.documentElement.lang, "en");
  assert.equal(documento.title, "VEC Granada Provincial Council");
  assert.equal(textos[0].childNodes[0].nodeValue, "  Corporate portal  ");
  assert.equal(textos[1].childNodes[0].nodeValue, "Preparing demonstration");
  assert.equal(atributos.get("aria-label"), "Main work queue");
  assert.equal(textoPortada("puntos_61", "en"), "61.4 points");
});
