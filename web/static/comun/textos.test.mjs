import assert from "node:assert/strict";
import test from "node:test";
import { cambiarIdioma, IDIOMA_POR_DEFECTO, IDIOMAS_DISPONIBLES, leerRecursoJSON, localizacionDe, montarSelectorIdioma,
  normalizarIndiceIdiomas, seleccionarIdioma } from "./idioma.js";
import { cargarTextos, crearTextos, esMensajePlural, urlCatalogo } from "./textos.js";

// Índice propio de la prueba: los idiomas son datos, también un tercero inventado.
const INDICE = normalizarIndiceIdiomas({
  por_defecto: "aa", seguir_navegador: true,
  idiomas: [
    { codigo: "aa", nombre: "Idioma A", localizacion: "es-ES" },
    { codigo: "bb", nombre: "Idioma B", localizacion: "en-GB" },
    { codigo: "cc", nombre: "Idioma C", localizacion: "fr-FR" },
  ],
});

test("el idioma sale del índice de datos: URL, navegador y defecto", () => {
  assert.equal(seleccionarIdioma("cc", ["bb"], INDICE), "cc");
  assert.equal(seleccionarIdioma("zz", ["xx-YY", "bb-GB"], INDICE), "bb");
  assert.equal(seleccionarIdioma("", ["xx"], INDICE), "aa");
  const sinNavegador = normalizarIndiceIdiomas({ ...INDICE, por_defecto: "aa", seguir_navegador: false,
    idiomas: INDICE.idiomas.map((i) => ({ ...i })) });
  assert.equal(seleccionarIdioma("", ["bb"], sinNavegador), "aa");
  assert.equal(localizacionDe("cc", INDICE), "fr-FR");
  assert.equal(localizacionDe("zz", INDICE), "es-ES");
});

test("el índice del repositorio es válido y define el idioma por defecto", () => {
  assert.ok(IDIOMAS_DISPONIBLES.length >= 1);
  assert.ok(IDIOMAS_DISPONIBLES.some((idioma) => idioma.codigo === IDIOMA_POR_DEFECTO));
});

test("un índice mal formado se rechaza", () => {
  for (const malo of [null, {}, { por_defecto: "aa", idiomas: [] },
    { por_defecto: "zz", idiomas: [{ codigo: "aa", nombre: "A", localizacion: "es-ES" }] },
    { por_defecto: "aa", idiomas: [{ codigo: "../x", nombre: "A", localizacion: "es-ES" }] },
    { por_defecto: "aa", idiomas: [{ codigo: "aa", nombre: "A", localizacion: "no válida" }] },
    { por_defecto: "aa", idiomas: [{ codigo: "aa", nombre: "A", localizacion: "es-ES" }, { codigo: "aa", nombre: "B", localizacion: "es-ES" }] }]) {
    assert.throws(() => normalizarIndiceIdiomas(malo), TypeError);
  }
});

test("el selector se rellena con los idiomas del índice y conserva la URL", () => {
  const creados = [];
  const documento = { createElement: () => { const o = {}; creados.push(o); return o; } };
  let opciones = [];
  let alCambiar;
  const selector = { ownerDocument: documento, value: "", replaceChildren: (...o) => { opciones = o; },
    addEventListener: (_tipo, manejador) => { alCambiar = manejador; } };
  let destino;
  const ubicacion = { href: "https://vec.example/portal-empleado/?lang=bb&vista=x#y", assign: (url) => { destino = url; } };
  assert.equal(montarSelectorIdioma(selector, ubicacion, INDICE), true);
  assert.deepEqual(opciones.map((o) => [o.value, o.lang, o.textContent]),
    [["aa", "aa", "Idioma A"], ["bb", "bb", "Idioma B"], ["cc", "cc", "Idioma C"]]);
  assert.equal(selector.value, "bb");
  selector.value = "cc";
  alCambiar();
  assert.equal(destino, "https://vec.example/portal-empleado/?lang=cc&vista=x#y");
  assert.equal(cambiarIdioma("zz", ubicacion, INDICE), false);
});

const RESPALDO = Object.freeze({
  general: { saludo: "Hola, {nombre}", solo: "Solo en respaldo", dias: { one: "{cuenta} día", other: "{cuenta} días" } },
  otra: { titulo: "Título" },
});

test("respaldo en el idioma por defecto para claves que faltan, con aviso", () => {
  const avisos = [];
  const textos = crearTextos({ modulo: "prueba", idioma: "bb", localizacion: "en-GB", respaldo: RESPALDO,
    propio: { general: { saludo: "Hello, {nombre}", dias: { one: "{cuenta} day", other: "{cuenta} days" }, sobrante: "x" } },
    avisar: (m) => avisos.push(m) });
  assert.equal(textos.traducir("general.saludo", { nombre: "Ana <b>" }), "Hello, Ana <b>");
  assert.equal(textos.traducir("general.solo"), "Solo en respaldo");
  assert.equal(textos.traducir("otra.titulo"), "Título");
  assert.deepEqual([...textos.faltantes], ["general.solo", "otra.titulo"]);
  assert.equal(avisos.length, 1);
  assert.equal(Object.hasOwn(textos.seccion("general"), "sobrante"), false, "solo cuentan las claves del respaldo");
  assert.throws(() => textos.traducir("general.inexistente"), /desconocida/u);
  assert.ok(Object.isFrozen(textos.seccion("general")));
});

test("plurales con Intl.PluralRules y formatos por la localización del catálogo", () => {
  const es = crearTextos({ modulo: "prueba", idioma: "aa", localizacion: "es-ES", respaldo: RESPALDO });
  assert.equal(es.plural("general.dias", 1), "1 día");
  assert.equal(es.plural("general.dias", 1500), "1500 días");
  assert.equal(es.traducir("general.dias", { cuenta: 2 }), "2 días");
  assert.equal(es.numero(12345.5), "12.345,5");
  assert.equal(es.fecha(new Date(Date.UTC(2026, 8, 29, 12)), { dateStyle: "long", timeZone: "UTC" }), "29 de septiembre de 2026");
  const en = crearTextos({ modulo: "prueba", idioma: "bb", localizacion: "en-GB", respaldo: RESPALDO,
    propio: { ...RESPALDO, general: { ...RESPALDO.general, dias: { one: "{cuenta} day", other: "{cuenta} days" } } } });
  assert.equal(en.plural("general.dias", 12000), "12,000 days");
  assert.equal(esMensajePlural({ one: "a", other: "b" }), true);
  assert.equal(esMensajePlural({ one: "a" }), false);
  assert.equal(esMensajePlural({ other: "b", titulo: "c" }), false);
});

test("cargarTextos lee respaldo y propio; si falta el propio usa el respaldo entero", async () => {
  const leidos = [];
  const leer = async (url) => {
    leidos.push(url.pathname.split("/textos/")[1]);
    if (url.pathname.endsWith("/bb/prueba.json")) return { general: { saludo: "Hello, {nombre}" } };
    if (url.pathname.endsWith("/aa/prueba.json")) return RESPALDO;
    throw new Error("no existe");
  };
  const raiz = new URL("https://vec.example/textos/");
  const bb = await cargarTextos("prueba", { idioma: "bb", porDefecto: "aa", leer, raiz, avisar: () => {} });
  assert.deepEqual(leidos.sort(), ["aa/prueba.json", "bb/prueba.json"]);
  assert.equal(bb.idioma, "bb");
  assert.equal(bb.traducir("general.saludo", { nombre: "Ann" }), "Hello, Ann");
  const avisos = [];
  const cc = await cargarTextos("prueba", { idioma: "cc", porDefecto: "aa", leer, raiz, avisar: (m) => avisos.push(m) });
  assert.equal(cc.idioma, "aa");
  assert.equal(cc.traducir("general.saludo", { nombre: "Ana" }), "Hola, Ana");
  assert.equal(avisos.length, 1);
  await assert.rejects(cargarTextos("otro", { idioma: "bb", porDefecto: "aa", leer, raiz, avisar: () => {} }));
});

test("las rutas de catálogo no admiten recorridos ni nombres arbitrarios", () => {
  assert.equal(urlCatalogo("aa", "cronos", new URL("https://vec.example/textos/")).href, "https://vec.example/textos/aa/cronos.json");
  for (const [idioma, modulo] of [["../aa", "cronos"], ["aa", "../x"], ["aa", "Cronos"], ["aa", "a/b"], ["", "x"]]) {
    assert.throws(() => urlCatalogo(idioma, modulo), TypeError);
  }
});

test("el transporte JSON solo lee el propio origen, sin redirecciones ni Referer", async () => {
  const llamadas = [];
  const fetchImpl = async (url, opciones) => {
    llamadas.push([url, opciones]);
    return { ok: true, headers: { get: () => "12" }, text: async () => "{\"a\":\"b\"}" };
  };
  await assert.rejects(leerRecursoJSON("https://otro.example/textos/idiomas.json", { fetchImpl }), /propio origen/u);
  assert.deepEqual(llamadas, []);
});

// 06/10/2026: el catálogo del idioma por defecto es el respaldo de todos los
// demás y se descargaba una vez por idioma pedido. Ahora se lee una sola vez.
test("cada catálogo se lee una sola vez y una lectura fallida se puede repetir", async () => {
  const { leerCatalogoUnaVez, urlCatalogo, URL_RAIZ_TEXTOS } = await import("./textos.js");
  const url = urlCatalogo("es", "portal", URL_RAIZ_TEXTOS);
  const primera = leerCatalogoUnaVez(url);
  assert.equal(leerCatalogoUnaVez(new URL(url.href)), primera);
  assert.equal(typeof (await primera), "object");
  const ausente = urlCatalogo("es", "no-existe-catalogo", URL_RAIZ_TEXTOS);
  const fallida = leerCatalogoUnaVez(ausente);
  await assert.rejects(fallida);
  assert.notEqual(leerCatalogoUnaVez(ausente), fallida);
  await assert.rejects(leerCatalogoUnaVez(ausente));
});
