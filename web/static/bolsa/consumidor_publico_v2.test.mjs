import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

const leer = (nombre) => readFileSync(new URL(nombre, import.meta.url), "utf8");
const html = leer("index.html");
const huella = "a".repeat(64);
const snapshot = { catalogo_id: "categorias-profesionales", version: 1, huella_sha256: huella, huella_proyeccion_sha256: huella };
const categoria = { clave: "auxiliar-administrativo", version: 1, etiqueta: "Categoría del snapshot publicado", semantica: "informacion", catalogo_categorias: snapshot };
const fuente = { revision: "sintetica-v2", actualizada_en: "2026-09-30T10:00:00Z", demostracion: false };
const convocatoria = {
  identificador_publico: "auxiliares-2026", version: "v2", huella_sha256: huella,
  titulo: "Bolsa sintética de auxiliares", resumen: "Información pública sintética.",
  tipo: { clave: "bolsa", version: 1, etiqueta: "Bolsa", semantica: "informacion" },
  estado: { clave: "publicada", version: 1, etiqueta: "Publicada", semantica: "informacion" },
  catalogo_categorias: snapshot, categorias: [{ clave: categoria.clave, version: 1 }],
  numero_requisitos: 0, numero_documentos: 0, numero_ayudas: 0, publicada_en: fuente.actualizada_en,
};
const listado = {
  esquema: "vec.bolsa.publico.convocatorias.v2", fuente,
  facetas: { tipos: [], categorias: [{ ...categoria, numero_resultados: 1 }], estados: [] }, diccionario_categorias: [categoria],
  paginacion: { pagina: 1, tamano: 12, total: 1, paginas: 1 }, convocatorias: [convocatoria],
};
const detalle = {
  esquema: "vec.bolsa.publico.convocatoria.v2", fuente, convocatoria, diccionario_categorias: [categoria],
  descripcion: "Detalle sintético público.", plazos: [], requisitos: [], documentos: [], ayuda: [],
};

class Nodo {
  constructor() { this.children = []; this.dataset = {}; this.value = ""; this.textContent = ""; this.hidden = true; this.eventos = {}; this.atributos = {}; }
  get firstChild() { return this.children[0]; }
  get options() { return this.children; }
  append(...nodos) { this.children.push(...nodos); }
  appendChild(nodo) { this.append(nodo); return nodo; }
  removeChild(nodo) { this.children.splice(this.children.indexOf(nodo), 1); }
  replaceChildren(...nodos) { this.children = nodos; }
  setAttribute(nombre, valor) { this.atributos[nombre] = valor; }
  addEventListener(nombre, fn) { this.eventos[nombre] = fn; }
  focus() {}
}

async function montar(respuestaListado = listado, respuestaDetalle = detalle) {
  const nodos = new Map([...html.matchAll(/\bid="([^"]+)"/gu)].map(([, id]) => [id, new Nodo()]));
  const peticiones = [];
  const documento = {
    getElementById: (id) => nodos.get(id), createElement: () => new Nodo(), querySelectorAll: () => [],
    body: { classList: { contains: () => false, toggle() {} } },
  };
  const contexto = vm.createContext({
    document: documento, URLSearchParams, AbortController, Intl,
    window: { location: { search: "?convocatoria=auxiliares-2026", hash: "" }, addEventListener() {} },
    history: { state: null, replaceState() {}, pushState() {} },
    VECBolsaI18n: { t: (clave, valores = {}) => `${clave} ${Object.values(valores).join(" ")}` },
    fetch: async (url, opciones) => {
      peticiones.push({ url, opciones });
      const datos = url.includes("/categorias")
        ? { esquema: "vec.bolsa.publico.categorias.v1", fuente, catalogo: { catalogo_id: snapshot.catalogo_id, version: 1, total: 0, huella_sha256: huella, huella_proyeccion_sha256: huella }, categorias: [] }
        : url.includes("/convocatorias/") ? respuestaDetalle : respuestaListado;
      return { ok: true, headers: { get: () => "application/json" }, json: async () => structuredClone(datos) };
    },
  });
  const scripts = [...html.matchAll(/<script\b[^>]*src="\/bolsa\/([^"?]+)\?[^" ]+"/gu)].map(([, nombre]) => nombre);
  for (const script of scripts.filter((nombre) => nombre !== "i18n-publica.js" && !nombre.startsWith("preparacion/"))) vm.runInContext(leer(script), contexto);
  await new Promise(setImmediate);
  return { nodos, peticiones };
}

function textos(nodo) {
  return typeof nodo === "string" ? nodo : [nodo.textContent, ...nodo.children.map(textos)].join(" ");
}

test("la página monta exclusivamente V2 antes del controlador renovado", () => {
  const version = "20260930-codexe-publico-v2-v1";
  assert.ok(html.indexOf(`/bolsa/contrato-v2.js?v=${version}`) < html.indexOf("/bolsa/bolsa.js?v=20261001-convoca-preparacion-v1"));
  assert.doesNotMatch(html, /contrato-v1\.js/);
});

test("el consumidor V2 muestra lista y detalle con las etiquetas del snapshot de cada convocatoria", async () => {
  const { nodos, peticiones } = await montar();
  assert.equal(nodos.get("estado-error").hidden, true);
  assert.equal(nodos.get("lista-convocatorias").hidden, false);
  assert.match(textos(nodos.get("lista-convocatorias")), /Bolsa sintética de auxiliares.*Categoría|Categoría.*Bolsa sintética/s);
  assert.equal(nodos.get("contenido-detalle").hidden, false);
  assert.equal(nodos.get("titulo-detalle").textContent, convocatoria.titulo);
  assert.match(textos(nodos.get("detalle-etiquetas")), /Categoría del snapshot publicado/);
  assert.ok(peticiones.every(({ opciones }) => opciones.credentials === "omit"));
});

test("el consumidor rechaza V1 sin mostrar datos ni solicitar su detalle", async () => {
  const { nodos, peticiones } = await montar({ ...listado, esquema: "vec.bolsa.publico.convocatorias.v1" });
  assert.equal(nodos.get("estado-error").hidden, false);
  assert.equal(nodos.get("lista-convocatorias").hidden, true);
  assert.equal(nodos.get("lista-convocatorias").children.length, 0);
  assert.ok(peticiones.every(({ url }) => !url.includes("/convocatorias/")));
});

test("el detalle V2 falla cerrado cuando su snapshot no coincide o el backend devuelve V1", async () => {
  for (const respuesta of [
    { ...detalle, esquema: "vec.bolsa.publico.convocatoria.v1" },
    { ...detalle, convocatoria: { ...convocatoria, catalogo_categorias: { ...snapshot, huella_sha256: "b".repeat(64) } } },
  ]) {
    const { nodos } = await montar(listado, respuesta);
    assert.equal(nodos.get("lista-convocatorias").hidden, false);
    assert.equal(nodos.get("detalle-error").hidden, false);
    assert.equal(nodos.get("contenido-detalle").hidden, true);
  }
});


test("el directorio usa catalogo_id del DTO canónico en resumen y etiqueta accesible", async () => {
  const { nodos } = await montar();
  const resumen = nodos.get("integridad-catalogo-categorias");
  assert.match(resumen.textContent, /categorias-profesionales/);
  assert.match(resumen.atributos["aria-label"], /categorias-profesionales/);
  assert.doesNotMatch(resumen.textContent, /undefined/);
});
