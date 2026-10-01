import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { catalogoTarifasDietasValido, formatearImporteCentimos, importeACentimos, montarCatalogoTarifasDietas } from "./vista.js";

const leer = async (ruta) => JSON.parse(await readFile(new URL(ruta, import.meta.url), "utf8"));

test("el ejemplo refleja las reglas existentes y nunca se clasifica como tarifa confirmada", async () => {
  const catalogo = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const original = await leer("../../../../../../data/catalogos/dietas/liquidacion-ejemplo-v1.json");
  assert.equal(catalogoTarifasDietasValido(catalogo), true);
  assert.equal(catalogo.estado, "ejemplo");
  assert.equal(catalogo.version, original.version);
  assert.equal(catalogo.version_tarifa_ref, original.version_tarifa_ref);
  assert.deepEqual(catalogo.fuentes, original.fuentes);
  assert.deepEqual(catalogo.tarifas.map((tarifa) => [tarifa.id, tarifa.importe_centimos]),
    original.reglas.map((regla) => [regla.referencia, regla.tope_centimos || regla.centimos_por_km]));
  assert.equal(catalogo.historia.every((entrada) => entrada.estado === "ejemplo"), true);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, historia: [{ ...catalogo.historia[0], idioma_motivo: "idioma-invalido" }] }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, estado: "aprobado" }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, fuentes: ["javascript:alert(1)"] }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, fuentes: ["https://user:secret@example.org/"] }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, fuentes: ["https://example.org/norma"] }), true);
});

test("el importe decimal se convierte sin redondeo ni separador de miles", () => {
  assert.equal(importeACentimos("37,40", "es-ES"), 3740);
  assert.equal(importeACentimos("0,26", "es-ES"), 26);
  assert.equal(importeACentimos("37.40", "en-GB"), 3740);
  assert.equal(importeACentimos("1,234.50", "en-GB"), null);
  assert.equal(importeACentimos("1,234", "es-ES"), null);
  assert.equal(importeACentimos("1,005", "es-ES"), null);
  assert.equal(importeACentimos("-1", "es-ES"), null);
  assert.equal(importeACentimos("999999999,99", "es-ES"), 99999999999);
});

test("el formato monetario conserva el último céntimo del entero seguro", () => {
  const legible = (valor) => valor.replace(/\s/gu, " ");
  assert.equal(legible(formatearImporteCentimos(Number.MAX_SAFE_INTEGER, "es-ES", "EUR")),
    "90.071.992.547.409,91 EUR");
  assert.equal(legible(formatearImporteCentimos(Number.MAX_SAFE_INTEGER - 1, "es-ES", "EUR")),
    "90.071.992.547.409,90 EUR");
  assert.equal(legible(formatearImporteCentimos(26, "en-GB", "EUR")), "EUR 0.26");
});

function documentoMinimo() {
  const doc = { activeElement: null };
  class Elemento {
    constructor(etiqueta) {
      this.ownerDocument = doc; this.tagName = etiqueta.toUpperCase(); this.nodeType = 1;
      this.children = []; this.parent = null; this.dataset = {}; this.listeners = new Map(); this.atributos = new Map();
      this._texto = ""; this.value = "";
    }
    set textContent(valor) { this.children = []; this._texto = String(valor); }
    get textContent() { return this._texto + this.children.map((n) => n.textContent).join(""); }
    append(...nodos) { for (const n of nodos) { this.children.push(n); n.parent = this; } }
    replaceChildren(...nodos) {
      for (let actual = doc.activeElement; actual; actual = actual.parent) {
        if (actual === this) { doc.activeElement = doc.body; break; }
      }
      this.children = []; this._texto = ""; this.append(...nodos);
    }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((n) => n !== this); }
    setAttribute(clave, valor) { this.atributos.set(clave, valor); }
    addEventListener(evento, callback) { this.listeners.set(evento, callback); }
    querySelector(selector) {
      const clave = selector.match(/^\[data-([a-z-]+)\]$/u)?.[1]?.replace(/-([a-z])/gu, (_m, letra) => letra.toUpperCase());
      if (clave && Object.hasOwn(this.dataset, clave)) return this;
      for (const hijo of this.children) { const hallado = hijo.querySelector?.(selector); if (hallado) return hallado; }
      return null;
    }
    focus(opciones) { doc.activeElement = this; this.foco = opciones; }
    scrollIntoView(opciones) { this.desplazamiento = opciones; }
  }
  doc.createElement = (etiqueta) => new Elemento(etiqueta);
  doc.createTextNode = (texto) => ({ nodeType: 3, textContent: texto });
  const raiz = doc.createElement("main"); doc.body = raiz; return raiz;
}

function descendientes(raiz, predicado) {
  return [raiz, ...raiz.children.flatMap((n) => n.children ? descendientes(n, predicado) : [])]
    .filter(predicado);
}

test("sin tarifas conserva versión, fuentes e historia", async () => {
  const original = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const textos = (await leer("../../../../textos/es/dietas-catalogo.json")).catalogo;
  const traducir = (clave, valores = {}) => textos[clave.slice(9)].replace(/\{(\w+)\}/gu, (_m, k) => valores[k] ?? "");
  const raiz = documentoMinimo();
  const desmontar = montarCatalogoTarifasDietas(raiz, {
    fuente: async () => ({ ...original, tarifas: [] }), traducir, localizacion: "es-ES",
  });
  await new Promise(setImmediate);
  assert.match(raiz.textContent, /Esta versión no contiene tarifas/u);
  assert.match(raiz.textContent, /propuesta:liquidacion:ejemplo:v1/u);
  assert.match(raiz.textContent, /Elena Martín/u);
  const motivo = descendientes(raiz, (n) => n.tagName === "DD" && n.textContent.includes("Preparación sintética"))[0];
  assert.equal(motivo.atributos.get("lang"), original.historia[0].idioma_motivo);
  assert.match(raiz.textContent, /BOE-A-2002-10337/u);
  assert.equal(raiz.querySelector("[data-dietas-catalogo-editor]"), null);
  desmontar();
});

test("reintentar devuelve el foco tras cargar y respeta otro foco elegido durante la espera", async () => {
  const original = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const textos = (await leer("../../../../textos/es/dietas-catalogo.json")).catalogo;
  const traducir = (clave, valores = {}) => textos[clave.slice(9)].replace(/\{(\w+)\}/gu, (_m, k) => valores[k] ?? "");
  for (const moverFoco of [false, true]) {
    const raiz = documentoMinimo(); let llamada = 0, resolver;
    const fuente = () => ++llamada === 1
      ? Promise.reject(new Error("503")) : new Promise((completar) => { resolver = completar; });
    const desmontar = montarCatalogoTarifasDietas(raiz, { fuente, traducir, localizacion: "es-ES" });
    await new Promise(setImmediate);
    const boton = descendientes(raiz, (n) => n.tagName === "BUTTON" && n.textContent === textos.reintentar)[0];
    boton.focus(); boton.listeners.get("click")();
    const otro = raiz.ownerDocument.createElement("button");
    if (moverFoco) { raiz.append(otro); otro.focus(); }
    resolver(original); await new Promise(setImmediate);
    assert.equal(raiz.ownerDocument.activeElement, moverFoco ? otro :
      descendientes(raiz, (n) => n.atributos?.get("role") === "status")[0]);
    assert.match(raiz.textContent, /Catálogo de tarifas de Dietas/u);
    desmontar();
  }
});

test("elegir tarifa enfoca la propuesta y conserva la unidad en revisión", async () => {
  const original = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const textos = (await leer("../../../../textos/es/dietas-catalogo.json")).catalogo;
  const traducir = (clave, valores = {}) => textos[clave.slice(9)].replace(/\{(\w+)\}/gu, (_m, k) => valores[k] ?? "");
  const raiz = documentoMinimo();
  const desmontar = montarCatalogoTarifasDietas(raiz, { fuente: async () => original, traducir, localizacion: "es-ES" });
  await new Promise(setImmediate);
  const editor = raiz.querySelector("[data-dietas-catalogo-editor]");
  assert.equal(editor.hidden, true);
  const botones = descendientes(raiz, (n) => n.tagName === "BUTTON" && n.textContent === textos.preparar);
  botones[1].listeners.get("click")();
  assert.equal(editor.hidden, false);
  assert.equal(raiz.ownerDocument.activeElement.tagName, "H3");
  assert.deepEqual(editor.desplazamiento, { block: "nearest", inline: "nearest" });
  assert.match(editor.textContent, /0,26\sEUR por km/u);
  const control = (nombre) => descendientes(editor, (n) => n.name === nombre)[0];
  control("importe_propuesto").value = "0,30";
  control("motivo").value = "Revisión del importe por kilómetro";
  descendientes(editor, (n) => n.tagName === "FORM")[0].listeners.get("submit")({ preventDefault() {} });
  assert.match(editor.textContent, /0,30\sEUR por km/u);
  assert.match(editor.textContent, /0,26\sEUR por km/u);
  assert.equal(descendientes(raiz, (n) => n.tagName === "BUTTON" && n.textContent === textos.publicar)[0].disabled, true);
  desmontar();
});

test("las dos traducciones cubren las mismas claves de la vista", async () => {
  const es = (await leer("../../../../textos/es/dietas-catalogo.json")).catalogo;
  const en = (await leer("../../../../textos/en/dietas-catalogo.json")).catalogo;
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  assert.equal(Object.values(es).every((valor) => typeof valor === "string" && valor.length > 0), true);
  assert.equal(Object.values(en).every((valor) => typeof valor === "string" && valor.length > 0), true);
});
