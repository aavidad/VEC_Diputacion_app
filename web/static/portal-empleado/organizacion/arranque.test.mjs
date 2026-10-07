import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearArranqueOrganizacion } from "./arranque.js";

function documentoFalso() {
  class Nodo {
    constructor(tagName, ownerDocument) {
      this.tagName = tagName;
      this.ownerDocument = ownerDocument;
      this.children = [];
      this.dataset = {};
      this.atributos = new Map();
      this.listeners = new Map();
    }
    append(...nodos) { for (const nodo of nodos) { nodo.parent = this; this.children.push(nodo); } }
    prepend(nodo) { nodo.parent = this; this.children.unshift(nodo); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((nodo) => nodo !== this); }
    setAttribute(clave, valor) { this.atributos.set(clave, String(valor)); }
    addEventListener(tipo, listener) { this.listeners.set(tipo, listener); }
    focus() { this.ownerDocument.activeElement = this; }
    querySelector(etiqueta) {
      if (this.tagName === etiqueta) return this;
      for (const hijo of this.children) {
        const encontrado = hijo.querySelector(etiqueta);
        if (encontrado) return encontrado;
      }
      return null;
    }
  }
  const documento = { documentElement: { lang: "es" }, title: "", createElement: (etiqueta) => new Nodo(etiqueta, documento) };
  const raiz = new Nodo("main", documento);
  documento.getElementById = (id) => id === "organizacion" ? raiz : null;
  return { documento, raiz };
}

function catalogoAviso(idioma = "es") {
  const mensajes = idioma === "en" ? {
    "general.organizacion_arranque_titulo": "This screen is not available",
    "general.organizacion_arranque_error": "Check the connection and try again.",
    "general.organizacion_arranque_reintentar": "Try again",
  } : {
    "general.organizacion_arranque_titulo": "Esta pantalla no está disponible",
    "general.organizacion_arranque_error": "Compruebe la conexión y pulse Reintentar.",
    "general.organizacion_arranque_reintentar": "Reintentar",
  };
  return { idioma, traducir: (ruta) => mensajes[ruta] };
}

test("prepara el mismo i18n antes de importar Organización y HTML apunta a la entrada", async () => {
  const { documento, raiz } = documentoFalso();
  let resolver;
  const orden = [];
  const arranque = crearArranqueOrganizacion({ documento,
    preparar: () => { orden.push("preparar"); return new Promise((resolve) => { resolver = resolve; }); },
    importar: async () => { orden.push("importar"); },
  });
  const inicio = arranque.iniciar();
  assert.deepEqual(orden, ["preparar"]);
  assert.equal(raiz.inert, true);
  assert.equal(raiz.atributos.get("aria-busy"), "true");
  resolver({ idioma: "en" });
  assert.equal(await inicio, true);
  assert.deepEqual(orden, ["preparar", "importar"]);
  assert.equal(documento.documentElement.lang, "en");
  assert.equal(raiz.atributos.get("aria-busy"), "false");
  const entrada = await readFile(new URL("./arranque.js", import.meta.url), "utf8");
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  assert.match(entrada, /personal\/i18n\.js\?v=20261007-pantallas-textos-final-v1/u);
  assert.match(entrada, /organizacion\.js\?v=20261007-pantallas-textos-final-v1/u);
  assert.match(html, /arranque\.js\?v=20261007-t-organizacion-arranque-v1/u);
});

test("fallo de preparación no importa consumidor; el botón reintenta desde la misma entrada", async () => {
  const { documento, raiz } = documentoFalso();
  const controlOriginal = documento.createElement("button");
  controlOriginal.hidden = false;
  const panelOcultoOriginal = documento.createElement("section");
  panelOcultoOriginal.hidden = true;
  raiz.append(controlOriginal, panelOcultoOriginal);
  let preparaciones = 0;
  let importaciones = 0;
  let resolver;
  const fallos = [];
  const arranque = crearArranqueOrganizacion({ documento,
    preparar: () => {
      preparaciones++;
      if (preparaciones === 1) throw new Error("catálogo no disponible");
      return new Promise((resolve) => { resolver = resolve; });
    },
    importar: async () => {
      assert.equal(controlOriginal.hidden, false);
      assert.equal(panelOcultoOriginal.hidden, true);
      importaciones++;
    },
    cargarAviso: async () => catalogoAviso("en"),
    registrarError: (error) => fallos.push(error),
  });
  assert.equal(await arranque.iniciar(), false);
  assert.equal(importaciones, 0);
  assert.equal(controlOriginal.hidden, true);
  assert.equal(panelOcultoOriginal.hidden, true);
  assert.equal(fallos[0].cause.message, "catálogo no disponible");
  const aviso = raiz.children[0];
  const boton = aviso.querySelector("button");
  assert.equal(aviso.atributos.get("role"), "alert");
  assert.equal(boton.textContent, "Try again");
  assert.equal(documento.documentElement.lang, "en");
  assert.equal(documento.activeElement, boton);
  boton.listeners.get("click")();
  boton.listeners.get("click")();
  assert.equal(preparaciones, 2);
  assert.equal(importaciones, 0);
  assert.equal(boton.disabled, true);
  resolver({ idioma: "es" });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(importaciones, 1);
  assert.equal(documento.documentElement.lang, "es");
  assert.equal(raiz.children.includes(aviso), false);
  assert.equal(raiz.atributos.get("aria-busy"), "false");
  assert.equal(documento.activeElement, raiz);
});

test("si falla la importación, Reintentar recarga el contexto de módulos", async () => {
  const { documento, raiz } = documentoFalso();
  const controlOriginal = documento.createElement("button");
  raiz.append(controlOriginal);
  let recargas = 0;
  let importaciones = 0;
  const arranque = crearArranqueOrganizacion({ documento,
    preparar: async () => ({ idioma: "es" }),
    importar: async () => { importaciones++; throw new Error("módulo no disponible"); },
    cargarAviso: async () => catalogoAviso(),
    registrarError: () => {},
    recargar: () => { recargas++; },
  });
  assert.equal(await arranque.iniciar(), false);
  assert.equal(controlOriginal.hidden, true);
  raiz.querySelector("button").listeners.get("click")();
  assert.equal(recargas, 1);
  assert.equal(importaciones, 1);
});

test("fallo del aviso conserva la causa técnica y no importa Organización", async () => {
  const { documento } = documentoFalso();
  const fallos = [];
  let importaciones = 0;
  const arranque = crearArranqueOrganizacion({ documento,
    preparar: async () => { throw new Error("catálogo principal"); },
    importar: async () => { importaciones++; },
    cargarAviso: async () => { throw new Error("aviso de respaldo"); },
    registrarError: (error) => fallos.push(error),
  });
  await assert.rejects(arranque.iniciar(), (error) =>
    error.codigo === "organizacion.arranque.aviso_no_disponible"
      && error.cause.errors[0].cause.message === "catálogo principal"
      && error.cause.errors[1].message === "aviso de respaldo");
  assert.equal(importaciones, 0);
  assert.equal(fallos.length, 2);
});
