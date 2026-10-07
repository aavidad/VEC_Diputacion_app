import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { montarModuloRPTPublicaV2 } from "./vista-rpt-publica-v2.js";

function raiz() {
  class Nodo {
    constructor(doc, etiqueta = "div") { this.ownerDocument = doc; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.atributos = new Map(); this.parent = null; this.textContent = ""; }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); }
    replaceChildren(...nodos) { this.children.forEach((n) => { n.parent = null; }); this.children = []; this.append(...nodos); }
    remove() { this.parent?.removeChild(this); }
    removeChild(n) { this.children = this.children.filter((x) => x !== n); n.parent = null; }
    setAttribute(k, v) { this.atributos.set(k, v); }
    addEventListener(k, f) { this.listeners.set(k, f); }
    focus() { this.ownerDocument.activeElement = this; }
    matches(s) { const k = s.match(/^\[data-([a-z0-9-]+)\]$/u)?.[1]?.replace(/-([a-z])/gu, (_m, c) => c.toUpperCase()); return k && this.dataset[k] !== undefined; }
    querySelector(s) { if (this.matches(s)) return this; for (const c of this.children) { const found = c.querySelector(s); if (found) return found; } return null; }
  }
  const doc = { createElement: (etiqueta) => new Nodo(doc, etiqueta), activeElement: null };
  const root = new Nodo(doc, "root"); return root;
}
function texto(n) { return `${n.textContent} ${n.children.map(texto).join(" ")}`; }
function pagina(vista, cambios = {}) {
  const categoria = { clave: "administrativo", denominacion: "ADMINISTRATIVO", origen: "categoria", grupos: ["C1"], escalas: [], puestos: 57, dotacion: 158, nivel_destino_mediana: 17, complemento_especifico_anual_centimos_mediana: 1461544 };
  const puesto = { codigo: "430-101-001", denominacion: "SECRETARÍA", centro_codigo: "101", centro: "GABINETE", delegacion: "PRESIDENCIA", grupos: [], escala: "", categoria_clave: "administrativo", categorias_claves: ["administrativo"], categorias_pendientes: [], nivel_destino: 17, complemento_especifico_anual_centimos: 1461544, dotacion: 1, tipo: "E", provision: "I" };
  return { vista, items: [vista === "puestos" ? puesto : categoria], total: vista === "puestos" ? 62 : 131, limit: 25, offset: 0,
    estado: "preparacion_no_autoritativa", corte: "2026-05-07", publicacion_ref: "rpt-publicada:2026-05-07", huella_sha256: "a".repeat(64),
    fuente: { documento: "RPT publicada", importacion: "rpt-2026", generado_en: "2026-09-17", aviso: "Sin ocupantes." },
    resumen: { puestos: 842, dotacion: 1714, categorias: 131, centros: 41 }, categorias_pendientes_grupo: ["AUXILIAR"], evidencia: { recibo_ref: "rptpublica:prueba" }, ...cambios };
}
const esperar = async () => { await Promise.resolve(); await Promise.resolve(); };

test("textos ES/EN mantienen las mismas claves y describen la revisión pendiente", () => {
  const catalogo = (idioma) => JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal-rpt-v2.json`, import.meta.url), "utf8")).general;
  const es = catalogo("es"), en = catalogo("en");
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  assert.match(es.estado, /pendiente de cotejo/u);
  assert.match(en.estado, /awaiting verification/u);
});

test("cada recuento abre la lista y categoría usa filtro exacto sin mostrar agregados incompatibles", async () => {
  const r = raiz(), llamadas = [];
  await montarModuloRPTPublicaV2({ raiz: r, cliente: { async listar(c) { llamadas.push(c); return pagina(c.vista); } } });
  assert.equal(llamadas[0].vista, "categorias");
  const contenedor = r.querySelector("[data-personal-rpt-publica-v2]");
  assert.ok(contenedor);
  assert.match(texto(contenedor), /131/);
  assert.doesNotMatch(texto(contenedor), /158/);
  const categoria = r.querySelector("[data-personal-rpt-v2-categoria]");
  categoria.listeners.get("click")(); await esperar();
  assert.deepEqual(llamadas[1], { vista: "puestos", q: "", limit: 25, offset: 0, categoria_clave: "administrativo", centro_codigo: "" });
  assert.match(texto(contenedor), /62/);
  const resumen = r.querySelector("[data-personal-rpt-v2-resumen]");
  resumen.listeners.get("click")(); await esperar();
  assert.equal(llamadas[2].categoria_clave, "");
  assert.equal(llamadas[2].vista, "puestos");
});

test("error borra filas previas y desmontar cancela la consulta pendiente", async () => {
  const r = raiz(); let resolver, signal, limpiar;
  const montaje = montarModuloRPTPublicaV2({ raiz: r, registrarDesmontar: (f) => { limpiar = f; }, cliente: { listar(_c, o) { signal = o.signal; return new Promise((resolve) => { resolver = resolve; }); } } });
  await esperar();
  const contenedor = r.querySelector("[data-personal-rpt-publica-v2]");
  assert.match(texto(contenedor), /Cargando/);
  limpiar();
  resolver(pagina("categorias"));
  const modulo = await montaje; modulo.desmontar();
  assert.equal(signal.aborted, true);
  assert.equal(r.querySelector("[data-personal-rpt-publica-v2]"), null);
});
