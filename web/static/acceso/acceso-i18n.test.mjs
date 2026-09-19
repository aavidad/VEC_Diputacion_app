import assert from "node:assert/strict";
import test from "node:test";
import {
  aplicarCatalogoAcceso,
  iniciarI18nAcceso,
  rutaCatalogoAcceso,
  seleccionarIdiomaAcceso,
} from "./acceso-i18n.js";

test("elige exclusivamente el catálogo empaquetado español", () => {
  assert.equal(seleccionarIdiomaAcceso(["fr-FR", "es-ES"]), "es");
  assert.equal(seleccionarIdiomaAcceso(["../../privado"]), "es");
  assert.equal(rutaCatalogoAcceso(["en-US"]), "/acceso/locales/es.json");
});

test("traduce texto y atributos admitidos sin construir rutas", async () => {
  const texto = {
    getAttribute: (nombre) => nombre === "data-i18n" ? "acceso.titulo" : null,
    textContent: "",
  };
  const atributo = {
    getAttribute: (nombre) => nombre === "data-i18n-atributo" ? "aria-label:acceso.enlaces.etiqueta" : null,
    setAttribute: (nombre, valor) => { atributo[nombre] = valor; },
  };
  const documento = {
    documentElement: { lang: "en-US" },
    querySelectorAll: (selector) => selector === "[data-i18n]" ? [texto] : [atributo],
  };
  aplicarCatalogoAcceso(documento, {
    "acceso.titulo": "Acceso",
    "acceso.enlaces.etiqueta": "Consultas",
  });
  assert.equal(texto.textContent, "Acceso");
  assert.equal(atributo["aria-label"], "Consultas");

  const llamadas = [];
  await iniciarI18nAcceso(documento, async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return { ok: false, json: async () => ({}) };
  }, ["../../privado"]);
  assert.deepEqual(llamadas, [["/acceso/locales/es.json", { credentials: "omit" }]]);
});
