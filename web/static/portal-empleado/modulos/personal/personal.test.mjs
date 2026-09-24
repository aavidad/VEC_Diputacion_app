import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { CAPACIDAD_CONSULTAR_PUESTO } from "./contrato.js";
import { ErrorClienteCategoriasPersonal } from "./cliente-http-categorias.js";
import { montarModuloPersonal } from "./vista.js";

test("categorías carga el i18n actualizado para sus estados internos", () => {
  const codigo = readFileSync(new URL("./vista.js", import.meta.url), "utf8");
  assert.match(codigo, /from "\.\/i18n\.js\?v=20260924-personal-interno-estados-v1";/u);
});

function raizFalsa() {
  class Nodo {
    constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); this.disabled = false; this.value = ""; }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
    replaceChildren(...nodos) { if (this.contains(this.ownerDocument.activeElement)) this.ownerDocument.activeElement = this.ownerDocument.body; this.children = []; this.append(...nodos); }
    removeChild(nodo) { if (nodo.contains(this.ownerDocument.activeElement)) this.ownerDocument.activeElement = this.ownerDocument.body; this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
    remove() { this.parent?.removeChild(this); }
    addEventListener(tipo, manejador) { this.listeners.set(tipo, manejador); }
    setAttribute(clave, valor) { this.atributos.set(clave, valor); }
    contains(nodo) { return nodo === this || this.children.some((hijo) => hijo.contains(nodo)); }
    focus() { this.ownerDocument.activeElement = this; }
    matches(selector) { const partes = selector.match(/^\[data-([a-z-]+)(?:="([a-z_-]+)")?\]$/u); if (!partes) return false; const clave = partes[1].replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase()); return this.dataset[clave] !== undefined && (partes[2] === undefined || this.dataset[clave] === partes[2]); }
    querySelector(selector) { if (this.matches(selector)) return this; for (const hijo of this.children) { const encontrado = hijo.querySelector(selector); if (encontrado) return encontrado; } return null; }
  }
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta), body: {} }; documento.activeElement = documento.body; return new Nodo(documento, "root");
}
function nodos(nodo) { return [nodo, ...nodo.children.flatMap(nodos)]; }
function texto(nodo) { return nodos(nodo).map((n) => n.textContent).join(" "); }
const completar = () => new Promise((resolve) => setImmediate(resolve));
function pagina(items = []) { return Object.freeze({ items: Object.freeze(items), total: items.length, limit: 25, offset: 0, catalogo: Object.freeze({ catalogo_id: "categorias-profesionales", catalogo_version: 1, catalogo_huella_sha256: "a".repeat(64) }), fuente: Object.freeze({ revision: "demo-v1", actualizada_en: "2026-09-20T08:00:00Z", demostracion: true, aviso: "DEMOSTRACIÓN pendiente de validación RRHH." }) }); }
const categoria = Object.freeze({ name: "Administrativo", area_etiqueta: "Administración general", state: "Demostración pendiente de validación RRHH" });

test("Personal muestra solo el catálogo RRHH autorizado, su demostración y su límite RPT", async () => {
  const raiz = raizFalsa(); const llamadas = [];
  const modulo = await montarModuloPersonal({ raiz, cliente: { async listarCategorias(consulta, { signal }) { llamadas.push({ consulta, signal }); return pagina([categoria]); } } });
  assert.equal(modulo.capacidad, CAPACIDAD_CONSULTAR_PUESTO); assert.equal(llamadas.length, 1); assert.deepEqual(llamadas[0].consulta, { q: "", area: "", limit: 25, offset: 0 });
  const texto = raiz.querySelector("[data-personal-categorias]").children.map((n) => n.textContent).join(" ");
  assert.match(texto, /demostracion:true/); assert.match(texto, /no es una RPT aprobada/); assert.doesNotMatch(texto, /nómina|servicios/i); modulo.desmontar(); assert.equal(raiz.querySelector("[data-personal-categorias]"), null);
});

test("el catálogo usa tabla semántica y rótulos localizados", async () => {
  const raiz = raizFalsa(); await montarModuloPersonal({ raiz, cliente: { async listarCategorias() { return pagina([categoria]); } } });
  const contenedor = raiz.querySelector("[data-personal-categorias]"); const tabla = nodos(contenedor).find((n) => n.tagName === "table");
  assert.equal(tabla.children[0].tagName, "caption"); assert.equal(tabla.children[1].tagName, "thead"); assert.equal(tabla.children[1].children[0].children[0].tagName, "th"); assert.equal(tabla.children[1].children[0].children[0].atributos.get("scope"), "col");
  const marco = contenedor.querySelector("[data-personal-categorias-marco]");
  assert.match(marco.className, /marco-tabla-paginado panel/);
  assert.equal(marco.children[0].atributos.get("role"), "region");
  assert.match(marco.children[0].atributos.get("style"), /max-height:min\(40vh,360px\)/);
});

test("Personal cierra la pantalla sin conservar filas tras un error", async () => {
  const raiz = raizFalsa(); const avisos = [];
  await montarModuloPersonal({ raiz, anunciar: (...argumentos) => avisos.push(argumentos), cliente: { async listarCategorias() { throw new Error("503"); } } });
  const texto = raiz.querySelector("[data-personal-categorias]").children.map((n) => n.textContent).join(" ");
  assert.match(texto, /No se pudo consultar/); assert.equal(avisos.length, 1); assert.doesNotMatch(texto, /Administrativo/); assert.ok(raiz.querySelector("[data-personal-categorias]").children.some((n) => n.tagName === "form"));
});

test("Personal distingue 401/403, 404 y 503 sin conservar filas anteriores", async () => {
  for (const [estado, tipo, fragmento] of [[401, "denegado", /no autorizó/i], [403, "denegado", /no autorizó/i], [404, "sin_ruta", /no encontró la consulta/i], [503, "servicio_no_disponible", /temporalmente indisponible/i]]) {
    const raiz = raizFalsa(); const avisos = []; let llamadas = 0;
    await montarModuloPersonal({ raiz, anunciar: (...argumentos) => avisos.push(argumentos), cliente: { listarCategorias() {
      llamadas += 1; if (llamadas === 1) return Promise.resolve(pagina([categoria]));
      return Promise.reject(new ErrorClienteCategoriasPersonal("estado_no_valido", estado));
    } } });
    const contenedor = raiz.querySelector("[data-personal-categorias]");
    const boton = contenedor.querySelector('[data-personal-categorias-foco="buscar"]'); boton.focus();
    nodos(contenedor).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} }); await completar();
    assert.equal(contenedor.dataset.personalCategoriasEstado, tipo);
    assert.match(texto(contenedor), fragmento); assert.doesNotMatch(texto(contenedor), /Administrativo/);
    assert.equal(avisos.length, 1); assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "buscar");
  }
});

test("buscar conserva foco y borrador; salir del formulario evita recuperar foco ajeno", async () => {
  const raiz = raizFalsa(); const pendientes = []; let llamadas = 0;
  await montarModuloPersonal({ raiz, cliente: { listarCategorias() {
    llamadas += 1; return llamadas === 1 ? Promise.resolve(pagina([categoria])) : new Promise((resolve) => pendientes.push(resolve));
  } } });
  const contenedor = raiz.querySelector("[data-personal-categorias]");
  let campo = controlPrueba(contenedor, "q"); campo.focus(); campo.value = "admin";
  nodos(contenedor).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "q");
  campo = controlPrueba(contenedor, "q"); campo.value = "borrador posterior";
  pendientes.shift()(pagina([categoria])); await completar();
  assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "q");
  assert.equal(controlPrueba(contenedor, "q").value, "borrador posterior");
  nodos(contenedor).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  raiz.focus(); pendientes.shift()(pagina([categoria])); await completar();
  assert.equal(raiz.ownerDocument.activeElement, raiz);
});

test("paginar conserva foco en el contexto disponible al terminar", async () => {
  const raiz = raizFalsa(); let resolver; let llamadas = 0;
  const primera = { ...pagina(Array.from({ length: 25 }, () => categoria)), total: 26 };
  const ultima = { ...pagina([categoria]), total: 26, offset: 25 };
  await montarModuloPersonal({ raiz, cliente: { listarCategorias() {
    llamadas += 1; return llamadas === 1 ? Promise.resolve(primera) : new Promise((resolve) => { resolver = resolve; });
  } } });
  const contenedor = raiz.querySelector("[data-personal-categorias]"); const siguiente = controlPrueba(contenedor, "siguiente");
  assert.equal(siguiente.disabled, false); siguiente.focus(); siguiente.listeners.get("click")();
  assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "espera");
  resolver(ultima); await completar();
  assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "anterior");
  assert.equal(controlPrueba(contenedor, "siguiente").disabled, true);
});

test("un error al paginar lleva el foco al control de reintento y purga las filas", async () => {
  const raiz = raizFalsa(); let llamadas = 0;
  const primera = { ...pagina(Array.from({ length: 25 }, () => categoria)), total: 26 };
  await montarModuloPersonal({ raiz, cliente: { listarCategorias() {
    llamadas += 1; return llamadas === 1 ? Promise.resolve(primera) : Promise.reject(new ErrorClienteCategoriasPersonal("estado_no_valido", 503));
  } } });
  const contenedor = raiz.querySelector("[data-personal-categorias]"); const siguiente = controlPrueba(contenedor, "siguiente");
  siguiente.focus(); siguiente.listeners.get("click")(); await completar();
  assert.equal(contenedor.dataset.personalCategoriasEstado, "servicio_no_disponible");
  assert.doesNotMatch(texto(contenedor), /Administrativo/);
  assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "buscar");
});

test("un filtro inválido cancela la consulta anterior y evita repintar datos tardíos", async () => {
  const raiz = raizFalsa(); let resolver; let senal; let llamadas = 0;
  await montarModuloPersonal({ raiz, cliente: { listarCategorias(_consulta, { signal }) {
    llamadas += 1; if (llamadas === 1) return Promise.resolve(pagina([categoria]));
    senal = signal; return new Promise((resolve) => { resolver = resolve; });
  } } });
  const contenedor = raiz.querySelector("[data-personal-categorias]");
  controlPrueba(contenedor, "q").value = "válido";
  nodos(contenedor).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  const campo = controlPrueba(contenedor, "q"); campo.focus(); campo.value = "x".repeat(101);
  nodos(contenedor).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  assert.equal(senal.aborted, true); assert.match(texto(contenedor), /filtro local no es válido/i);
  resolver(pagina([categoria])); await completar();
  assert.equal(contenedor.dataset.personalCategoriasEstado, "error"); assert.doesNotMatch(texto(contenedor), /Administrativo/);
  assert.equal(raiz.ownerDocument.activeElement.dataset.personalCategoriasFoco, "q");
});

function controlPrueba(contenedor, clave) { return contenedor.querySelector(`[data-personal-categorias-foco="${clave}"]`); }

test("desmontar aborta la consulta y no deja DOM tardío", async () => {
  const raiz = raizFalsa(); let resolver; let signal; let desmontar; const pendiente = new Promise((resolve) => { resolver = resolve; });
  const montaje = montarModuloPersonal({ raiz, registrarDesmontar: (limpiar) => { desmontar = limpiar; }, cliente: { listarCategorias(_consulta, opciones) { signal = opciones.signal; return pendiente; } } });
  await Promise.resolve(); const contenedor = raiz.querySelector("[data-personal-categorias]");
  // La limpieza se registra antes de esperar la respuesta y puede cancelar la navegación.
  assert.ok(contenedor); desmontar(); resolver(pagina([categoria])); const modulo = await montaje; modulo.desmontar(); assert.equal(signal.aborted, true); assert.equal(raiz.querySelector("[data-personal-categorias]"), null);
});
