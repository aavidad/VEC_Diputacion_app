import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaFichaIntegralPersonal } from "./vista-ficha-integral.js";

function raizFalsa() {
  class Nodo {
    constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); this.disabled = false; }
    append(...hijos) { this.children.push(...hijos); hijos.forEach((hijo) => { hijo.parent = this; }); }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((hijo) => hijo !== this); }
    addEventListener(tipo, fn) { this.listeners.set(tipo, fn); }
    setAttribute(clave, valor) { this.atributos.set(clave, valor); }
    focus() { this.enfocado = true; }
    matches(selector) { const coincide = selector.match(/^\[data-([a-z-]+)(?:="([a-z_-]+)")?\]$/u); if (!coincide) return false; const clave = coincide[1].replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase()); return this.dataset[clave] !== undefined && (coincide[2] === undefined || this.dataset[clave] === coincide[2]); }
    querySelector(selector) { if (this.matches(selector)) return this; for (const hijo of this.children) { const encontrado = hijo.querySelector(selector); if (encontrado) return encontrado; } return null; }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) };
  return new Nodo(documento, "root");
}
function nodos(n) { return [n, ...n.children.flatMap(nodos)]; }
function texto(n) { return nodos(n).map((item) => item.textContent).join(" "); }
function tab(ficha, clave) { return nodos(ficha).find((n) => n.dataset.personalFichaTab === clave); }
const completar = () => new Promise((resolve) => setImmediate(resolve));

test("la portada no fabrica persona, relación, curso, fichaje ni nómina", () => {
  const raiz = raizFalsa(); montarVistaFichaIntegralPersonal({ raiz }); const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  assert.ok(ficha); assert.equal(tab(ficha, "tiempo").textContent, "Tiempo");
  assert.match(texto(ficha), /Abra un apartado para consultar su fuente/);
  assert.doesNotMatch(texto(ficha), /Antonio López|Funcionario de carrera|Junio 2026|Nómina orientativa/);
  assert.equal(raiz.querySelector("[data-personal-ficha-ayuda]").tagName, "details");
  assert.ok(nodos(ficha).filter((n) => n.dataset.personalFichaDestino).every((n) => n.disabled));
});

test("cada apartado se consulta solo al abrirlo y conserva procedencia sin referencias en la petición", async () => {
  const raiz = raizFalsa(); const llamadas = [];
  montarVistaFichaIntegralPersonal({ raiz, fuentes: {
    servicios: { consultarPropios(entrada) { llamadas.push(entrada); return { estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ desde: "2020-01-01", procedencia: "Diputación", reconocimiento: "Confirmado", estado: "Reconocido" }] }; } },
    tiempo: { consultarPropios() { throw new Error("no debe abrirse"); } },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); assert.equal(llamadas.length, 0);
  tab(ficha, "servicios").listeners.get("click")(); await completar();
  assert.equal(llamadas.length, 1); assert.deepEqual(Object.keys(llamadas[0]), ["signal"]);
  assert.match(texto(ficha), /Fuente: Personal/); assert.match(texto(ficha), /Reconocido/); assert.match(texto(ficha), /1 ene 2020/);
  assert.doesNotMatch(texto(ficha), /curso acreditado|trienio concedido|Entrada a las 08:00/i);
});

test("estados separados: fuente ausente, vacío autorizado, denegado y error", async () => {
  const raiz = raizFalsa(); const avisos = [];
  montarVistaFichaIntegralPersonal({ raiz, anunciar: (...args) => avisos.push(args), fuentes: {
    servicios: { consultarPropios: () => ({ estado: "vacio", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [] }) },
    tiempo: { consultarPropios: () => ({ estado: "denegado", items: [{ tipo: "oculto" }] }) },
    formacion: { consultarPropios: () => Promise.reject(new Error("detalle interno")) },
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  tab(ficha, "economia").listeners.get("click")(); assert.match(texto(ficha), /No hay una fuente propia autorizada conectada/);
  tab(ficha, "servicios").listeners.get("click")(); await completar(); assert.match(texto(ficha), /no devuelve registros/);
  tab(ficha, "tiempo").listeners.get("click")(); await completar(); assert.match(texto(ficha), /No tiene permiso/); assert.doesNotMatch(texto(ficha), /oculto/);
  tab(ficha, "formacion").listeners.get("click")(); await completar(); assert.match(texto(ficha), /No se pudo consultar/); assert.doesNotMatch(texto(ficha), /detalle interno/); assert.equal(avisos.length, 1);
});

test("cambiar de pestaña y desmontar aborta consultas sin pintar respuestas tardías", async () => {
  const raiz = raizFalsa(); let resolver; let senal;
  const montaje = montarVistaFichaIntegralPersonal({ raiz, fuentes: { relaciones: { consultarPropios({ signal }) { senal = signal; return new Promise((resolve) => { resolver = resolve; }); } } } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "relaciones").listeners.get("click")(); await completar();
  tab(ficha, "servicios").listeners.get("click")(); assert.equal(senal.aborted, true);
  resolver({ estado: "disponible", fuente: "Personal", actualizado_en: "2026-09-24T08:00:00Z", items: [{ puesto: "No visible" }] }); await completar();
  assert.doesNotMatch(texto(ficha), /No visible/); montaje.desmontar(); assert.equal(raiz.querySelector("[data-personal-ficha-integral]"), null);
});

test("teclado y navegación a otros módulos no transportan identidad", () => {
  const raiz = raizFalsa(); const destinos = [];
  montarVistaFichaIntegralPersonal({ raiz, navegarModulo: (...args) => destinos.push(args) }); const ficha = raiz.querySelector("[data-personal-ficha-integral]");
  let prevenido = false; tab(ficha, "ficha").listeners.get("keydown")({ key: "ArrowRight", preventDefault() { prevenido = true; } });
  assert.equal(prevenido, true); assert.equal(tab(ficha, "relaciones").atributos.get("aria-selected"), "true"); assert.equal(tab(ficha, "relaciones").enfocado, true);
  tab(ficha, "ficha").listeners.get("click")(); nodos(ficha).find((n) => n.dataset.personalFichaDestino === "dietas").listeners.get("click")();
  assert.deepEqual(destinos, [["dietas"]]);
});

test("catálogos existentes se montan bajo demanda y se limpian al salir", async () => {
  const raiz = raizFalsa(); let montajes = 0; let limpiezas = 0;
  montarVistaFichaIntegralPersonal({ raiz, montarCatalogos: ({ registrarDesmontar }) => {
    montajes += 1; registrarDesmontar(() => { limpiezas += 1; }); return { desmontar() { limpiezas += 1; } };
  } });
  const ficha = raiz.querySelector("[data-personal-ficha-integral]"); tab(ficha, "catalogos").listeners.get("click")(); await completar();
  assert.equal(montajes, 1); assert.match(texto(ficha), /consulta HTTP completa/);
  tab(ficha, "ficha").listeners.get("click")(); assert.ok(limpiezas >= 1);
});
