import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { MENSAJES_NOMINAS_ES, crearTraductorNominas } from "./i18n.js";
import { montarVistaNominas } from "./vista.js";

assert.equal(crearTraductorNominas()("titulo"), "Nóminas y retribuciones");
assert.match(MENSAJES_NOMINAS_ES.explicacion_no_configurado, /No se muestran importes, pagos ni recibos de ejemplo/);
assert.match(MENSAJES_NOMINAS_ES.explicacion_vacio, /no acredita un importe cero ni un pago/);
assert.match(MENSAJES_NOMINAS_ES.certificados_pendientes, /No se generan certificados/);
assert.equal(crearTraductorNominas()("seleccionado", { periodo: "2026-09", version: 2 }), "Recibo seleccionado: 2026-09, versión 2.");

const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
const i18nVersionada = new URL("./i18n.js?v=20260924-f2-web2", import.meta.url);
assert.match(vista, /from "\.\/i18n\.js\?v=20260924-f2-web2"/);
assert.equal((await import(i18nVersionada.href)).crearTraductorNominas()("titulo"), "Nóminas y retribuciones");
assert.equal(typeof (await import("./vista.js")).montarVistaNominas, "function");
assert.doesNotMatch(vista, /datos-presentacion|datos-sinteticos|localStorage|sessionStorage|document\.cookie|fetch\(/);
assert.match(vista, /fuente\.consultar\(\{ signal: controlador\.signal \}\)/);
assert.match(vista, /descarga\.disabled = descargando \|\| !recibo\.descargable \|\| typeof fuente\?\.descargar !== "function"/);
assert.match(vista, /await validarDocumento\(await fuente\.descargar\(recibo\.referencia/);
assert.match(vista, /URL\.createObjectURL\(archivo\.contenido\)/);
assert.match(vista, /URL\.revokeObjectURL\(url\)/);

const css = await readFile(new URL("./nominas.css", import.meta.url), "utf8");
assert.match(css, /\.nominas-principal \{ display: grid; grid-template-columns:/);
assert.match(css, /\.nominas-tabla \{ max-width: 100%; max-height: min\(45vh, 420px\); min-width: 0; overflow: auto;/);
assert.match(css, /@media \(max-width: 900px\)/);
assert.match(css, /@media \(max-width: 620px\)/);
console.log("nominas presentation tests: ok");

function crearDOM() {
  class Nodo {
    constructor(doc, etiqueta) {
      this.ownerDocument = doc;
      this.tagName = etiqueta;
      this.children = [];
      this.dataset = {};
      this.atributos = new Map();
      this.listeners = new Map();
      this.textContent = "";
      this.hidden = false;
    }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    setAttribute(clave, valor) { this.atributos.set(clave, String(valor)); }
    addEventListener(clave, fn) { this.listeners.set(clave, fn); }
    focus() { this.ownerDocument.activeElement = this; }
    remove() { this.parent.children = this.parent.children.filter((nodo) => nodo !== this); }
    querySelectorAll(selector) {
      const coincide = (nodo) => selector === "button[aria-controls]"
        ? nodo.tagName === "button" && nodo.atributos.has("aria-controls") : nodo.tagName === selector;
      return this.children.flatMap((nodo) => [
        ...(coincide(nodo) ? [nodo] : []), ...nodo.querySelectorAll(selector),
      ]);
    }
    querySelector(selector) { return this.querySelectorAll(selector)[0] ?? null; }
  }
  const doc = { createElement: (etiqueta) => new Nodo(doc, etiqueta), activeElement: null };
  const raiz = new Nodo(doc, "root");
  const buscar = (clase) => {
    const visitar = (nodo) => nodo.className?.split(" ").includes(clase)
      ? nodo : nodo.children.map(visitar).find(Boolean);
    return visitar(raiz);
  };
  return { raiz, doc, buscar };
}

test("la ayuda extensa se abre solo con ? y Escape devuelve el foco", () => {
  const { raiz, doc, buscar } = crearDOM();
  const vistaMontada = montarVistaNominas({ raiz });
  const ayuda = buscar("nominas-ayuda");
  const boton = buscar("nominas-ayuda-boton");
  const texto = ayuda.querySelector("p");
  assert.equal(boton.textContent, "?");
  assert.equal(boton.atributos.get("aria-label"), MENSAJES_NOMINAS_ES.ayuda);
  assert.equal(boton.atributos.get("aria-controls"), texto.id);
  assert.equal(texto.hidden, true);
  boton.listeners.get("click")();
  assert.equal(texto.hidden, false);
  assert.equal(boton.atributos.get("aria-expanded"), "true");
  boton.listeners.get("keydown")({ key: "Escape" });
  assert.equal(texto.hidden, true);
  assert.equal(doc.activeElement, boton);
  vistaMontada.desmontar();
});

test("filtrar período actualiza filas, detalle y foco sin conservar otro recibo", async () => {
  const { raiz, doc, buscar } = crearDOM();
  const recibos = [
    { referencia: "REC-SEP", periodo: "2026-09", tipo: "Nómina", version: 1, descargable: false },
    { referencia: "REC-AGO", periodo: "2026-08", tipo: "Nómina", version: 2, descargable: false },
  ];
  const vistaMontada = montarVistaNominas({ raiz, fuente: {
    consultar: async () => ({ estado: "disponible", origen: "Fuente de prueba", recibos }),
  } });
  await vistaMontada.consultar();
  const detalle = buscar("nominas-detalle");
  const texto = (nodo) => [nodo.textContent, ...nodo.children.map(texto)].join(" ");
  let selector = buscar("nominas-barra").querySelector("select");
  selector.value = "2026-08";
  selector.listeners.get("change")();
  selector = buscar("nominas-barra").querySelector("select");
  assert.equal(selector.value, "2026-08");
  assert.equal(doc.activeElement, selector);
  assert.match(texto(detalle), /REC-AGO/);
  assert.doesNotMatch(texto(detalle), /REC-SEP/);
  assert.equal(buscar("nominas-tabla").querySelectorAll("tr").length, 2);
  assert.equal(buscar("nominas-tabla").querySelector("button[aria-controls]").atributos.get("aria-pressed"), "true");
  vistaMontada.desmontar();
});
