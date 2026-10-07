import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { crearTextos } from "../../../../comun/textos.js";
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


test("dos propuestas conservan revisión, corrección y retirada independientes en ambos idiomas", async () => {
  const original = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const respaldo = await leer("../../../../textos/es/dietas-catalogo.json");
  for (const [idioma, localizacion, cantidades] of [["es", "es-ES", ["38,40", "0,20", "39,40"]], ["en", "en-GB", ["38.40", "0.20", "39.40"]]]) {
    const propio = await leer(`../../../../textos/${idioma}/dietas-catalogo.json`);
    const { traducir, faltantes } = crearTextos({ modulo: "dietas-catalogo", idioma, localizacion, respaldo, propio });
    const textos = propio.catalogo, raiz = documentoMinimo(), anuncios = [];
    const antes = structuredClone(original);
    let llamadas = 0;
    const desmontar = montarCatalogoTarifasDietas(raiz, {
      fuente: async () => { llamadas++; return original; }, traducir, localizacion, anunciar: (texto) => anuncios.push(texto),
    });
    await new Promise(setImmediate);
    const resumen = raiz.querySelector("[data-dietas-catalogo-propuestas]");
    const editor = raiz.querySelector("[data-dietas-catalogo-editor]");
    const botones = (zona, texto) => descendientes(zona, (n) => n.tagName === "BUTTON" && n.textContent === texto);
    const control = (nombre) => descendientes(editor, (n) => n.name === nombre)[0];
    const enviar = () => descendientes(editor, (n) => n.tagName === "FORM")[0].listeners.get("submit")({ preventDefault() {} });
    const preparar = botones(raiz, textos.preparar);
    assert.match(resumen.textContent, new RegExp(textos.propuestas_vacias, "u"));
    for (const indice of [0, 1]) {
      preparar[indice].listeners.get("click")();
      control("importe_propuesto").value = cantidades[indice];
      control("motivo").value = `<img src=x onerror=alert(1)> ${indice}`;
      control("fuente_propuesta").value = original.fuentes[indice];
      enviar();
    }
    assert.equal(botones(resumen, textos.corregir_propuesta).length, 2);
    assert.match(resumen.textContent, /<img src=x onerror=alert\(1\)> 0/u);
    assert.equal(descendientes(resumen, (n) => n.tagName === "IMG").length, 0);
    const moneda = (centimos) => formatearImporteCentimos(centimos, localizacion, "EUR");
    assert.ok(resumen.textContent.includes(traducir("catalogo.aumento", { importe: moneda(100) })));
    assert.ok(resumen.textContent.includes(traducir("catalogo.reduccion", { importe: `${moneda(6)}${textos.por_km}` })));
    assert.deepEqual(descendientes(resumen, (n) => n.tagName === "A").map((n) => n.href), original.fuentes.slice(0, 2));
    assert.equal(descendientes(resumen, (n) => n.tagName === "TD" && n.textContent.includes("<img"))[0].atributos.get("lang"), localizacion);
    assert.equal(descendientes(editor, (n) => n.tagName === "BUTTON" && n.textContent === textos.publicar)[0].disabled, true);
    // La selección recupera lo revisado y enfoca el editor sin perder la otra tarifa.
    botones(resumen, textos.corregir_propuesta)[0].listeners.get("click")();
    assert.equal(raiz.ownerDocument.activeElement.tagName, "H3");
    assert.equal(control("importe_propuesto").value, cantidades[0]);
    assert.equal(control("fuente_propuesta").value, original.fuentes[0]);
    assert.equal(botones(resumen, textos.corregir_propuesta).length, 2);
    control("importe_propuesto").value = cantidades[2]; control("importe_propuesto").listeners.get("input")();
    assert.equal(botones(resumen, textos.corregir_propuesta).length, 1);
    control("importe_propuesto").value = "-1"; enviar();
    assert.equal(botones(resumen, textos.corregir_propuesta).length, 1);
    assert.equal(raiz.ownerDocument.activeElement, control("importe_propuesto"));
    control("importe_propuesto").value = cantidades[2]; enviar();
    assert.equal(botones(resumen, textos.corregir_propuesta).length, 2);
    assert.ok(resumen.textContent.includes(traducir("catalogo.aumento", { importe: moneda(200) })));
    // Retirar la otra tarifa no reemplaza el editor que se está corrigiendo.
    botones(resumen, textos.retirar_propuesta)[1].listeners.get("click")();
    assert.equal(editor.hidden, false);
    assert.equal(control("importe_propuesto").value, cantidades[2]);
    assert.equal(raiz.ownerDocument.activeElement, botones(resumen, textos.corregir_propuesta)[0]);
    botones(resumen, textos.retirar_propuesta)[0].listeners.get("click")();
    assert.equal(editor.hidden, true);
    assert.equal(botones(resumen, textos.corregir_propuesta).length, 0);
    assert.equal(raiz.ownerDocument.activeElement.tagName, "H3");
    assert.ok(resumen.textContent.includes(textos.propuestas_vacias));
    assert.ok(anuncios.at(-1).includes(traducir("catalogo.propuesta_retirada", { tarifa: traducir("catalogo.tarifa_resumen", {concepto: textos.concepto_manutencion, grupo: "2", pais: "ES"}) })));
    preparar[0].listeners.get("click")();
    assert.equal(control("importe_propuesto").value, "");
    assert.deepEqual(original, antes); assert.equal(llamadas, 1); assert.deepEqual(faltantes, []);
    desmontar();
    assert.equal(raiz.children.length, 0);
  }
});

test("diferencia del importe seguro conserva cada céntimo sin modificar la tarifa", async () => {
  const original = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const catalogo = structuredClone(original); catalogo.tarifas[0].importe_centimos = Number.MAX_SAFE_INTEGER;
  const respaldo = await leer("../../../../textos/es/dietas-catalogo.json");
  const { traducir } = crearTextos({ modulo: "dietas-catalogo", idioma: "es", localizacion: "es-ES", respaldo });
  const raiz = documentoMinimo();
  const desmontar = montarCatalogoTarifasDietas(raiz, { fuente: async () => catalogo, traducir, localizacion: "es-ES" });
  await new Promise(setImmediate);
  descendientes(raiz, (n) => n.tagName === "BUTTON" && n.textContent === respaldo.catalogo.preparar)[0].listeners.get("click")();
  const editor = raiz.querySelector("[data-dietas-catalogo-editor]");
  descendientes(editor, (n) => n.name === "importe_propuesto")[0].value = "0,00";
  descendientes(editor, (n) => n.name === "motivo")[0].value = "Revisión local";
  descendientes(editor, (n) => n.tagName === "FORM")[0].listeners.get("submit")({ preventDefault() {} });
  assert.ok(raiz.querySelector("[data-dietas-catalogo-propuestas]").textContent.includes(traducir("catalogo.reduccion", {
    importe: formatearImporteCentimos(Number.MAX_SAFE_INTEGER, "es-ES", "EUR"),
  })));
  assert.equal(catalogo.tarifas[0].importe_centimos, Number.MAX_SAFE_INTEGER);
  desmontar();
});


test("una mutación de la fuente no reinterpreta la propuesta como otra versión", async () => {
  const fuente = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const original = structuredClone(fuente);
  const respaldo = await leer("../../../../textos/es/dietas-catalogo.json");
  const { traducir } = crearTextos({ modulo: "dietas-catalogo", idioma: "es", localizacion: "es-ES", respaldo });
  const raiz = documentoMinimo();
  const desmontar = montarCatalogoTarifasDietas(raiz, { fuente: async () => fuente, traducir, localizacion: "es-ES" });
  await new Promise(setImmediate);
  descendientes(raiz, (n) => n.tagName === "BUTTON" && n.textContent === respaldo.catalogo.preparar)[0].listeners.get("click")();
  fuente.version = "catalogo:version:posterior"; fuente.tarifas[0].importe_centimos = 9000;
  fuente.fuentes[0] = "https://example.org/fuente-distinta";
  const editor = raiz.querySelector("[data-dietas-catalogo-editor]");
  descendientes(editor, (n) => n.name === "importe_propuesto")[0].value = "38,40";
  descendientes(editor, (n) => n.name === "motivo")[0].value = "Revisión local";
  descendientes(editor, (n) => n.tagName === "FORM")[0].listeners.get("submit")({ preventDefault() {} });
  const resumen = raiz.querySelector("[data-dietas-catalogo-propuestas]");
  assert.equal(descendientes(resumen, (n) => n.tagName === "BUTTON" && n.textContent === respaldo.catalogo.corregir_propuesta).length, 1);
  assert.ok(editor.textContent.includes(original.version));
  assert.equal(raiz.textContent.includes(fuente.version), false);
  assert.ok(resumen.textContent.includes(formatearImporteCentimos(original.tarifas[0].importe_centimos, "es-ES", "EUR")));
  assert.equal(descendientes(resumen, (n) => n.tagName === "A")[0].href, original.fuentes[0]);
  desmontar();
});
