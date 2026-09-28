import assert from "node:assert/strict";
import test from "node:test";
import {
  aplicarCatalogoAcceso,
  iniciarI18nAcceso,
  montarAyudaAcceso,
  rutaCatalogoAcceso,
  seleccionarIdiomaAcceso,
} from "./acceso-i18n.js";

test("elige catálogos empaquetados por URL y preferencia del navegador", () => {
  assert.equal(seleccionarIdiomaAcceso(["fr-FR", "es-ES"]), "es");
  assert.equal(seleccionarIdiomaAcceso(["../../privado"]), "es");
  assert.equal(rutaCatalogoAcceso(["en-US"]), "/acceso/locales/en.json");
  assert.equal(rutaCatalogoAcceso(["en-US"], "es"), "/acceso/locales/es.json");
  assert.equal(rutaCatalogoAcceso(["es-ES"], "en"), "/acceso/locales/en.json");
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

test("el catálogo inglés cambia el idioma del documento solo tras cargarlo", async () => {
  const documento = { documentElement: { lang: "es" }, querySelectorAll: () => [] };
  const ubicacion = { href: "https://vec.example/acceso/?lang=en" };
  const cargado = await iniciarI18nAcceso(documento, async (ruta) => {
    assert.equal(ruta, "/acceso/locales/en.json");
    return { ok: true, json: async () => ({ "acceso.titulo": "Human Resources portal" }) };
  }, ["es-ES"], ubicacion);
  assert.equal(cargado, "en");
  assert.equal(documento.documentElement.lang, "en");
});

test("la ayuda empieza oculta, se abre con ?, y Escape la cierra devolviendo el foco", () => {
  const eventos = new Map();
  const atributos = new Map([["aria-controls", "acceso-autorizacion"]]);
  let foco = null;
  const boton = {
    getAttribute: (nombre) => atributos.get(nombre),
    setAttribute: (nombre, valor) => atributos.set(nombre, valor),
    addEventListener: (nombre, funcion) => eventos.set(`boton:${nombre}`, funcion),
    focus: () => { foco = "boton"; },
  };
  const contenido = {
    id: "acceso-autorizacion",
    hidden: false,
    focus: () => { foco = "contenido"; },
  };
  const documento = {
    getElementById: (id) => id === "boton-ayuda-acceso" ? boton : id === contenido.id ? contenido : null,
    addEventListener: (nombre, funcion) => eventos.set(`documento:${nombre}`, funcion),
  };

  assert.equal(montarAyudaAcceso(documento), true);
  assert.equal(contenido.hidden, true);
  assert.equal(atributos.get("aria-expanded"), "false");

  eventos.get("boton:click")();
  assert.equal(contenido.hidden, false);
  assert.equal(atributos.get("aria-expanded"), "true");
  assert.equal(foco, "contenido");

  let prevenido = false;
  eventos.get("documento:keydown")({ key: "Escape", preventDefault: () => { prevenido = true; } });
  assert.equal(prevenido, true);
  assert.equal(contenido.hidden, true);
  assert.equal(atributos.get("aria-expanded"), "false");
  assert.equal(foco, "boton");

  eventos.get("boton:click")();
  eventos.get("boton:click")();
  assert.equal(contenido.hidden, true);
  assert.equal(atributos.get("aria-expanded"), "false");
  assert.equal(foco, "boton");
});
