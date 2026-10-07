import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

const leer = (nombre) => readFileSync(new URL(nombre, import.meta.url), "utf8");
const html = leer("index.html");
const controlador = leer("bolsa.js");
const contextoCatalogo = { globalThis: {} };
vm.runInNewContext(leer("i18n-publica.js"), contextoCatalogo);
const { mensajes, t, numero, plural } = contextoCatalogo.globalThis.VECBolsaI18n;
const huella = "a".repeat(64);
const snapshot = { catalogo_id: "categorias-profesionales", version: 1, huella_sha256: huella, huella_proyeccion_sha256: huella };
const categoria = { clave: "nandu", version: 1, etiqueta: "Ñandú", semantica: "informacion", catalogo_categorias: snapshot };
const instante = "2026-09-30T10:00:00Z";
const fuente = { revision: "sintetica-v2", actualizada_en: instante, demostracion: false };
const convocatoria = {
  identificador_publico: "aviso-2026", version: "v2", huella_sha256: huella,
  titulo: "Convocatoria sintética", resumen: "Información pública sintética.",
  tipo: { clave: "bolsa", version: 1, etiqueta: "Bolsa", semantica: "informacion" },
  estado: { clave: "publicada", version: 1, etiqueta: "Publicada", semantica: "informacion" },
  catalogo_categorias: snapshot, categorias: [{ clave: categoria.clave, version: 1 }],
  numero_requisitos: 0, numero_documentos: 0, numero_ayudas: 0, publicada_en: instante,
  plazo_destacado: {
    titulo: "Plazo sintético", tipo: { clave: "presentacion", version: 1, etiqueta: "Presentación", semantica: "informacion" },
    abre_en: instante, cierra_en: "2026-10-02T10:00:00Z",
    etiqueta_situacion: "Abierto", semantica_situacion: "exito",
  },
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
const directorio = {
  esquema: "vec.bolsa.publico.categorias.v1", fuente,
  catalogo: { ...snapshot, total: 2 },
  categorias: [
    { clave: "nandu", etiqueta: "Ñandú", orden: 1, area: "administracion", area_etiqueta: "Administración", numero_convocatorias: 0 },
    { clave: "nube", etiqueta: "Nube", orden: 1, area: "administracion", area_etiqueta: "Administración", numero_convocatorias: 0 },
  ],
};

class Nodo {
  constructor() {
    this.children = []; this.dataset = {}; this.value = ""; this.textContent = "";
    this.hidden = true; this.eventos = {}; this.atributos = {};
  }
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

const texto = (nodo) => typeof nodo === "string" ? nodo : [nodo.textContent, ...nodo.children.map(texto)].join(" ");

test("el catálogo público localiza números y plurales de resultados y directorio", () => {
  assert.equal(numero(12345), "12.345");
  assert.equal(plural("convocatoria_encontrada", 0), "0 convocatorias encontradas");
  assert.equal(plural("convocatoria_encontrada", 1), "1 convocatoria encontrada");
  assert.equal(plural("convocatoria_encontrada", 12345), "12.345 convocatorias encontradas");
  assert.equal(plural("requisito", 1), "1 requisito");
  assert.equal(plural("requisito", 2), "2 requisitos");
  assert.equal(plural("documento", 0), "0 documentos");
  assert.equal(plural("ayuda", 1), "1 ayuda");
  assert.equal(plural("proceso_publicado", 1), "1 proceso publicado");
  assert.equal(plural("plazo_abierto", 2), "2 plazos abiertos");
});

test("las plantillas conservan los valores de la fuente y traducen solo el envoltorio", () => {
  const revision = "ref:publica:2026";
  assert.equal(t("fuente_actualizada", { revision, fecha: "24 sept 2026, 07:14" }),
    "Fuente ref:publica:2026 · actualizada 24 sept 2026, 07:14");
  assert.equal(t("catalogo_resumen", { referencia: "cat:publico", version: "1", total: "2", huella: huella.slice(0, 16) }),
    `Catálogo cat:publico · versión 1 · 2 categorías · huella ${huella.slice(0, 16)}…`);
  assert.equal(t("catalogo_resumen_aria", { referencia: "cat:publico", version: "1", total: "2", huella }),
    `Catálogo cat:publico, versión 1, 2 categorías, huella SHA-256 ${huella}`);
  assert.equal(t("abrir_documento", { formato: "PDF", titulo: "Bases públicas" }), "Abrir PDF: Bases públicas");
  assert.equal(t("ver_procesos_de", { categoria: "Auxiliar administrativo" }), "Ver procesos de Auxiliar administrativo");
});

test("cada clave visible usada por el controlador existe, incluidas las dos formas plurales", () => {
  const claves = [...controlador.matchAll(/\bt\("([a-z_]+)"/g)].map((coincidencia) => coincidencia[1]);
  const plurales = [...controlador.matchAll(/\bplural\("([a-z_]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.ok(claves.length > 20);
  assert.ok(plurales.length >= 6);
  for (const clave of claves) assert.equal(typeof mensajes[clave], "string", `falta ${clave}`);
  for (const clave of plurales) {
    assert.equal(typeof mensajes[`${clave}_uno`], "string", `falta ${clave}_uno`);
    assert.equal(typeof mensajes[`${clave}_otros`], "string", `falta ${clave}_otros`);
  }
  assert.doesNotMatch(controlador, /"(?:Publicada el|Bases publicadas el|Ver procesos|Cargando el catálogo profesional…|El directorio no está disponible\.)"/);
});

async function montar(idioma, respuestaDirectorio = directorio) {
  const nodos = new Map([...html.matchAll(/\bid="([^"]+)"/gu)].map(([, id]) => [id, new Nodo()]));
  const raiz = { lang: "es" };
  const documento = {
    documentElement: raiz,
    getElementById: (id) => nodos.get(id), createElement: () => new Nodo(), querySelectorAll: () => [],
    body: { classList: { contains: () => false, toggle() {} } },
  };
  const contexto = vm.createContext({
    document: documento, URL, URLSearchParams, AbortController, Intl,
    location: { href: `https://vec.example/bolsa/?lang=${idioma}&convocatoria=aviso-2026` },
    navigator: { languages: [idioma === "en" ? "es-ES" : "en-GB"] },
    window: { location: { search: `?lang=${idioma}&convocatoria=aviso-2026`, hash: "" }, addEventListener() {} },
    history: { state: null, replaceState() {}, pushState() {} },
    fetch: async (url) => {
      const datos = url.includes("/categorias") ? respuestaDirectorio
        : url.includes("/convocatorias/") ? detalle : listado;
      return { ok: true, headers: { get: () => "application/json" }, json: async () => structuredClone(datos) };
    },
  });
  for (const nombre of ["i18n-publica.js", "contrato-v2.js", "bolsa.js"]) vm.runInContext(leer(nombre), contexto);
  await new Promise(setImmediate);
  return { nodos, raiz };
}

for (const [idioma, locale, orden] of [
  ["es", "es-ES", ["Nube", "Ñandú"]],
  ["en", "en-GB", ["Ñandú", "Nube"]],
]) {
  test(`el portal público ${idioma} muestra fechas de Madrid y ordena las categorías con su idioma`, async () => {
    const { nodos, raiz } = await montar(idioma);
    const fechaHora = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(instante));
    const fechaDia = new Intl.DateTimeFormat(locale, { dateStyle: "long", timeZone: "Europe/Madrid" }).format(new Date(instante));
    assert.equal(raiz.lang, idioma);
    assert.ok(nodos.get("revision-fuente").textContent.includes(fechaHora));
    assert.ok(texto(nodos.get("lista-convocatorias")).includes(fechaHora));
    assert.ok(texto(nodos.get("lista-convocatorias")).includes(fechaDia));
    assert.ok(texto(nodos.get("detalle-publicacion")).includes(fechaDia));
    const grupos = nodos.get("grupos-directorio-categorias");
    const categorias = grupos.children[0].children[1].children.map((fila) => fila.children[0].children[0].textContent);
    assert.deepEqual(categorias, orden);

    const buscador = nodos.get("buscar-categoria");
    buscador.value = "nandu";
    buscador.eventos.input();
    assert.match(texto(grupos), /Ñandú/u);
    assert.doesNotMatch(texto(grupos), /Nube/u);
    buscador.value = "inexistente";
    buscador.eventos.input();
    assert.equal(nodos.get("vacio-directorio-categorias").hidden, false);
  });
}

test("un directorio inválido conserva el estado de error en ambos idiomas", async () => {
  for (const idioma of ["es", "en"]) {
    const { nodos } = await montar(idioma, { ...directorio, catalogo: { ...directorio.catalogo, total: 3 } });
    assert.equal(nodos.get("error-directorio-categorias").hidden, false);
    assert.equal(nodos.get("grupos-directorio-categorias").hidden, true);
  }
});
