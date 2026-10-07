import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { prepararTextosPersonal } from "./i18n.js?v=20261007-pantallas-textos-final-v1";

test.before(async () => { await prepararTextosPersonal(); });
import { exigirVersiones, posterior } from "../../versiones-cache.test-helper.mjs";
import { montarModuloEstructuraOrganizativaPublica } from "./vista-estructura-organizativa-publica.js";

test("estructura carga el i18n actualizado del corte F2", () => {
  const codigo = readFileSync(new URL("./vista-estructura-organizativa-publica.js", import.meta.url), "utf8");
  // i18n de Personal renovado (organización histórica): nunca la URL immutable previa.
  exigirVersiones(codigo, "./i18n.js", posterior("20260924-f2-web2"));
});

function raiz() {
  class Nodo {
    constructor(documento, etiqueta = "div") {
      this.ownerDocument = documento;
      this.tagName = etiqueta;
      this.children = [];
      this.dataset = {};
      this.parent = null;
      this.textContent = "";
      this.atributos = new Map();
      this.listeners = {};
    }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    removeChild(n) { this.children = this.children.filter((h) => h !== n); n.parent = null; }
    remove() { this.parent?.removeChild(this); }
    setAttribute(k, v) { this.atributos.set(k, String(v)); }
    addEventListener(tipo, listener) { this.listeners[tipo] = listener; }
    focus() { this.ownerDocument.activeElement = this; }
    matches(selector) {
      const clave = selector.match(/^\[data-([a-z-]+)\]$/u)?.[1]?.replace(/-([a-z])/gu, (_m, l) => l.toUpperCase());
      return clave ? this.dataset[clave] !== undefined : this.tagName === selector;
    }
    querySelector(selector) {
      if (this.matches(selector)) return this;
      for (const hijo of this.children) {
        const encontrado = hijo.querySelector(selector);
        if (encontrado) return encontrado;
      }
      return null;
    }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) };
  return new Nodo(documento, "root");
}
function textoVisible(nodo) {
  return nodo.hidden ? "" : [nodo.textContent, ...nodo.children.map(textoVisible)].join(" ");
}
function estructura() {
  const unidades = [];
  for (let i = 0; i < 14; i += 1) unidades.push({ clave: `delegacion-${i}`, etiqueta: "Delegación", tipo: "delegacion" });
  for (let i = 0; i < 41; i += 1) unidades.push({ clave: `centro-${i}`, etiqueta: "Centro", tipo: "centro", adscripcion_clave: "delegacion-0" });
  for (let i = 0; i < 11; i += 1) unidades.push({ clave: `puesto-${i}`, etiqueta: "Jefatura", tipo: "puesto_responsabilidad", adscripcion_clave: "centro-0" });
  return {
    unidades,
    fuente: {
      revision: "demo-v1", actualizada_en: "2026-09-06T00:00:00Z",
      demostracion: true, aviso: "Demo sin vigencia",
      huella_sha256: "0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c",
    },
  };
}

test("la ayuda y la advertencia quedan detrás de ?; en pantalla solo la pastilla «En preparación»", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  const boton = vista.querySelector("[data-personal-estructura-ayuda]");
  const contenido = vista.querySelector("[data-personal-estructura-ayuda-contenido]");
  assert.equal(boton.tagName, "button");
  assert.equal(boton.textContent, "?");
  assert.match(boton.atributos.get("aria-label"), /Consulte las unidades y su adscripción/u);
  assert.equal(boton.atributos.get("aria-expanded"), "false");
  assert.equal(contenido.hidden, true);
  assert.equal(contenido.children.at(-1).className, "rpt-huella");
  const css = readFileSync(new URL("../../portal-modulos.css", import.meta.url), "utf8");
  assert.match(css, /\.modulo-personal \.rpt-huella\s*\{[^}]*overflow-wrap:\s*anywhere/u);
  assert.match(textoVisible(vista), /En preparación/u);
  assert.doesNotMatch(textoVisible(vista), /no acredita vigencia administrativa|Portal del Empleado → Personal/u);
  assert.match(textoVisible(vista), /14 delegaciones/u);
  assert.doesNotMatch(textoVisible(vista), /Huella SHA-256|demo-v1|2026-09-06T00:00:00Z/u);
  boton.listeners.click();
  assert.equal(contenido.hidden, false);
  assert.equal(boton.atributos.get("aria-expanded"), "true");
  assert.match(textoVisible(vista), /demo-v1/u);
  assert.match(textoVisible(vista), /Esta estructura está en preparación.*Compruebe la organización vigente/u);
  assert.match(textoVisible(vista), /0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c/u);
  assert.doesNotMatch(textoVisible(vista), /2026-09-06T00:00:00Z/u);
  boton.listeners.click();
  assert.equal(contenido.hidden, true);
  modulo.desmontar();
});

test("la marca de estructura en preparación es una pastilla del tema", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const aviso = r.querySelector("[data-personal-estructura-aviso]");
  const componentes = readFileSync(new URL("../../portal-componentes.css", import.meta.url), "utf8");
  assert.equal(aviso.tagName, "span");
  assert.equal(aviso.className, "estado-chip");
  assert.equal(aviso.textContent, "En preparación");
  assert.match(componentes, /\.estado-chip\s*\{[^}]*white-space:\s*nowrap/u);
  modulo.desmontar();
});

test("la tabla prioriza nombre y adscripción legible, conserva clave y scroll interno", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  const tabla = vista.querySelector("[data-personal-estructura-tabla]");
  assert.equal(tabla.atributos.get("role"), "region");
  assert.equal(tabla.atributos.get("tabindex"), "0");
  const cuerpo = tabla.querySelector("tbody");
  assert.equal(cuerpo.children.length, 10);
  const claves = new Set();
  const recoger = () => vista.querySelector("tbody").children.forEach((fila) =>
    claves.add(fila.children.at(-1).children[0].textContent));
  recoger();
  assert.match(textoVisible(vista), /1 \/ 7/u);
  const siguiente = vista.querySelector("[data-personal-estructura-siguiente]");
  assert.equal(siguiente.textContent, "Siguiente");
  siguiente.focus();
  siguiente.listeners.click();
  assert.equal(vista.querySelector("tbody").children.length, 10);
  recoger();
  assert.equal(r.ownerDocument.activeElement.dataset.personalEstructuraSiguiente, "");
  const centro = vista.querySelector("tbody").children[4];
  assert.equal(centro.children[0].tagName, "th");
  assert.equal(centro.children[0].textContent, "Centro");
  assert.equal(centro.children[0].atributos.get("scope"), "row");
  assert.equal(centro.children[2].textContent, "Delegación");
  assert.equal(textoVisible(centro.children.at(-1)), " centro-0");
  for (let pagina = 2; pagina < 7; pagina += 1) {
    vista.querySelector("[data-personal-estructura-siguiente]").listeners.click();
    recoger();
  }
  assert.equal(claves.size, 66);
  assert.match(textoVisible(vista), /7 \/ 7/u);
  assert.equal(vista.querySelector("tbody").children.length, 6);
  assert.equal(vista.querySelector("[data-personal-estructura-siguiente]").disabled, true);
  assert.equal(r.ownerDocument.activeElement.dataset.personalEstructuraAnterior, "");
  vista.querySelector("[data-personal-estructura-anterior]").listeners.click();
  assert.equal(vista.querySelector("tbody").children.length, 10);
  modulo.desmontar();
});

test("mantiene abierta y enfocada la ayuda si termina la carga", async () => {
  const r = raiz();
  let resolver;
  const montaje = montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: {
    obtener: () => new Promise((resolve) => { resolver = resolve; }),
  } });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  const ayuda = vista.querySelector("[data-personal-estructura-ayuda]");
  ayuda.focus();
  ayuda.listeners.click();
  assert.equal(vista.querySelector("[data-personal-estructura-ayuda-contenido]").hidden, false);
  resolver(estructura());
  const modulo = await montaje;
  const nuevoBoton = vista.querySelector("[data-personal-estructura-ayuda]");
  assert.notEqual(nuevoBoton, ayuda);
  assert.equal(nuevoBoton.atributos.get("aria-expanded"), "true");
  assert.equal(vista.querySelector("[data-personal-estructura-ayuda-contenido]").hidden, false);
  assert.equal(r.ownerDocument.activeElement, nuevoBoton);
  assert.match(textoVisible(vista), /demo-v1/u);
  modulo.desmontar();
});

test("error y desmontaje abortan sin pintar respuesta tardía", async () => {
  const r = raiz();
  const avisos = [];
  await montarModuloEstructuraOrganizativaPublica({ raiz: r, anunciar: (...aviso) => avisos.push(aviso), cliente: { async obtener() { throw new Error("503"); } } });
  assert.equal(avisos.length, 1);
  let resolver;
  let signal;
  let limpiar;
  const otraRaiz = raiz();
  const montaje = montarModuloEstructuraOrganizativaPublica({ raiz: otraRaiz, registrarDesmontar: (fn) => { limpiar = fn; }, cliente: { obtener({ signal: s }) { signal = s; return new Promise((resolve) => { resolver = resolve; }); } } });
  await Promise.resolve();
  limpiar();
  resolver(estructura());
  const modulo = await montaje;
  modulo.desmontar();
  assert.equal(signal.aborted, true);
  assert.equal(otraRaiz.querySelector("[data-personal-estructura-organizativa-publica]"), null);
});

test("un error permite reintentar en la misma vista y recupera tabla y foco sin duplicar consultas", async () => {
  const r = raiz();
  const avisos = [];
  let resolverReintento;
  let consultas = 0;
  const modulo = await montarModuloEstructuraOrganizativaPublica({
    raiz: r,
    anunciar: (...aviso) => avisos.push(aviso),
    cliente: { obtener() {
      consultas++;
      if (consultas === 1) return Promise.reject(new Error("503"));
      return new Promise((resolve) => { resolverReintento = resolve; });
    } },
  });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  const boton = vista.querySelector("[data-personal-estructura-reintentar]");
  assert.equal(boton.tagName, "button");
  assert.equal(boton.textContent, "Reintentar consulta");
  assert.equal(vista.querySelector("[data-personal-estructura-tabla]"), null);
  boton.focus();
  boton.listeners.click();
  boton.listeners.click();
  assert.equal(consultas, 2);
  const carga = vista.querySelector("[data-personal-estructura-cargando]");
  assert.match(textoVisible(carga), /Cargando estructura organizativa/u);
  assert.equal(r.ownerDocument.activeElement, carga);
  resolverReintento(estructura());
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(r.querySelector("[data-personal-estructura-organizativa-publica]"), vista);
  assert.equal(vista.querySelector("[data-personal-estructura-reintentar]"), null);
  assert.equal(vista.querySelector("[data-personal-estructura-tabla]").querySelector("tbody").children.length, 10);
  assert.equal(r.ownerDocument.activeElement.tagName, "h2");
  assert.equal(avisos.length, 1);
  modulo.desmontar();
});

test("desmontar durante el reintento aborta y descarta una respuesta tardía", async () => {
  const r = raiz();
  let resolverReintento;
  let signalReintento;
  let consultas = 0;
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: {
    obtener({ signal }) {
      consultas++;
      if (consultas === 1) return Promise.reject(new Error("503"));
      signalReintento = signal;
      return new Promise((resolve) => { resolverReintento = resolve; });
    },
  } });
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  vista.querySelector("[data-personal-estructura-reintentar]").listeners.click();
  const carga = vista.querySelector("[data-personal-estructura-cargando]");
  modulo.desmontar();
  assert.equal(signalReintento.aborted, true);
  resolverReintento(estructura());
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(consultas, 2);
  assert.equal(r.querySelector("[data-personal-estructura-organizativa-publica]"), null);
  assert.equal(vista.querySelector("[data-personal-estructura-cargando]"), carga);
  assert.equal(vista.querySelector("[data-personal-estructura-tabla]"), null);
});

function buscar(r, texto) {
  const entrada = r.querySelector("[data-personal-estructura-buscar]");
  entrada.value = texto;
  entrada.listeners.input();
  return entrada;
}
function filtrarTipo(r, valor) {
  const tipo = r.querySelector("[data-personal-estructura-tipo]");
  tipo.value = valor;
  tipo.listeners.change();
  return tipo;
}

test("busca nombres y adscripciones sin acentos y conserva el nombre del padre filtrado", async () => {
  const r = raiz();
  const datos = estructura();
  datos.unidades[0].etiqueta = "Área de Gestión";
  datos.unidades[14].etiqueta = "Centro de Atención";
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return datos; } } });
  buscar(r, "  ATENCION  ");
  filtrarTipo(r, "centro");
  assert.equal(r.querySelector("tbody").children.length, 1);
  assert.equal(r.querySelector("tbody").children[0].children[2].textContent, "Área de Gestión");
  assert.equal(r.querySelector("[data-personal-estructura-recuento]").textContent, "1 de 66 unidades");
  buscar(r, "area de gestion");
  filtrarTipo(r, "centro");
  assert.equal(r.querySelector("tbody").children.length, 10);
  assert.equal(r.querySelector("[data-personal-estructura-recuento]").textContent, "41 de 66 unidades");
  assert.equal(r.querySelector("tbody").children[0].children[2].textContent, "Área de Gestión");
  assert.doesNotMatch(textoVisible(r.querySelector("tbody")), /delegacion-0/u);
  modulo.desmontar();
});

test("reinicia la página con los filtros y mantiene campo, selección, foco y ayuda", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const ayuda = r.querySelector("[data-personal-estructura-ayuda]");
  const contenido = r.querySelector("[data-personal-estructura-ayuda-contenido]");
  ayuda.listeners.click();
  r.querySelector("[data-personal-estructura-siguiente]").listeners.click();
  assert.match(textoVisible(r), /2 \/ 7/u);
  const entrada = r.querySelector("[data-personal-estructura-buscar]");
  entrada.focus();
  entrada.value = "centro";
  entrada.selectionStart = 3;
  entrada.selectionEnd = 5;
  entrada.listeners.input();
  assert.equal(r.querySelector("[data-personal-estructura-buscar]"), entrada);
  assert.equal(r.ownerDocument.activeElement, entrada);
  assert.equal(entrada.selectionStart, 3);
  assert.equal(entrada.selectionEnd, 5);
  assert.equal(r.querySelector("[data-personal-estructura-ayuda]"), ayuda);
  assert.equal(r.querySelector("[data-personal-estructura-ayuda-contenido]"), contenido);
  assert.equal(contenido.hidden, false);
  assert.match(textoVisible(r), /1 \/ 6/u);
  filtrarTipo(r, "centro");
  assert.match(textoVisible(r), /1 \/ 5/u);
  r.querySelector("[data-personal-estructura-siguiente]").listeners.click();
  assert.match(textoVisible(r), /2 \/ 5/u);
  buscar(r, "");
  filtrarTipo(r, "");
  assert.match(textoVisible(r), /1 \/ 7/u);
  assert.equal(r.querySelector("[data-personal-estructura-recuento]").textContent, "66 de 66 unidades");
  modulo.desmontar();
});

test("el vacío ofrece recuento cero y navegación deshabilitada; desmontar cierra los filtros", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const entrada = buscar(r, "unidad inexistente");
  const tipo = filtrarTipo(r, "centro");
  const vista = r.querySelector("[data-personal-estructura-organizativa-publica]");
  assert.equal(r.querySelector("tbody"), null);
  assert.equal(r.querySelector("[data-personal-estructura-recuento]").textContent, "0 de 66 unidades");
  assert.match(textoVisible(r), /No hay unidades que coincidan con los filtros/u);
  assert.match(textoVisible(r), /0 \/ 0/u);
  assert.equal(r.querySelector("[data-personal-estructura-anterior]").disabled, true);
  assert.equal(r.querySelector("[data-personal-estructura-siguiente]").disabled, true);
  buscar(r, "");
  assert.equal(r.querySelector("tbody").children.length, 10);
  const tablaAnterior = vista.querySelector("[data-personal-estructura-tabla]");
  modulo.desmontar();
  entrada.value = "inexistente";
  entrada.listeners.input();
  tipo.value = "puesto_responsabilidad";
  tipo.listeners.change();
  assert.equal(vista.querySelector("[data-personal-estructura-tabla]"), tablaAnterior);
  assert.equal(r.querySelector("[data-personal-estructura-organizativa-publica]"), null);
});

test("limpiar ambos filtros recupera la primera página y devuelve el foco al buscador", async () => {
  const r = raiz();
  const modulo = await montarModuloEstructuraOrganizativaPublica({ raiz: r, cliente: { async obtener() { return estructura(); } } });
  const limpiar = r.querySelector("[data-personal-estructura-limpiar]");
  assert.equal(limpiar.disabled, true);
  const entrada = buscar(r, "centro");
  const tipo = filtrarTipo(r, "centro");
  r.querySelector("[data-personal-estructura-siguiente]").listeners.click();
  assert.equal(limpiar.disabled, false);
  limpiar.listeners.click();
  assert.equal(entrada.value, "");
  assert.equal(tipo.value, "");
  assert.equal(limpiar.disabled, true);
  assert.equal(r.ownerDocument.activeElement, entrada);
  assert.equal(r.querySelector("[data-personal-estructura-recuento]").textContent, "66 de 66 unidades");
  assert.match(textoVisible(r), /1 \/ 7/u);
  assert.match(textoVisible(r), /Buscar por nombre o adscripción/u);
  modulo.desmontar();
  limpiar.listeners.click();
  assert.equal(r.querySelector("[data-personal-estructura-organizativa-publica]"), null);
});
