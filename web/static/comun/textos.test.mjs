import assert from "node:assert/strict";
import test from "node:test";
import { cambiarIdioma, IDIOMA_POR_DEFECTO, IDIOMAS_DISPONIBLES, leerPorRed, leerRecursoJSON, localizacionDe, montarSelectorIdioma,
  normalizarIndiceIdiomas, prepararIdiomas, seleccionarIdioma } from "./idioma.js";
import { cargarTextos, crearTextos, esCatalogoValido, esMensajePlural, reintentarTextos, urlCatalogo } from "./textos.js";

// Índice propio de la prueba: los idiomas son datos, también un tercero inventado.
const INDICE = normalizarIndiceIdiomas({
  por_defecto: "aa", seguir_navegador: true,
  idiomas: [
    { codigo: "aa", nombre: "Idioma A", localizacion: "es-ES" },
    { codigo: "bb", nombre: "Idioma B", localizacion: "en-GB" },
    { codigo: "cc", nombre: "Idioma C", localizacion: "fr-FR" },
  ],
});

await prepararIdiomas();

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

test("cargarTextos lee sólo el elegido y usa el respaldo entero si falla", async () => {
  const leidos = [];
  const leer = async (url) => {
    leidos.push(url.pathname.split("/textos/")[1]);
    if (url.pathname.endsWith("/bb/prueba.json")) return { general: { saludo: "Hello, {nombre}" } };
    if (url.pathname.endsWith("/aa/prueba.json")) return RESPALDO;
    throw new Error("no existe");
  };
  const raiz = new URL("https://vec.example/textos/");
  const bb = await cargarTextos("prueba", { idioma: "bb", porDefecto: "aa", leer, raiz, avisar: () => {} });
  assert.deepEqual(leidos, ["bb/prueba.json"]);
  assert.equal(bb.idioma, "bb");
  assert.equal(bb.traducir("general.saludo", { nombre: "Ann" }), "Hello, Ann");
  assert.throws(() => bb.traducir("general.solo"), /desconocida/u);
  const avisos = [];
  const cc = await cargarTextos("prueba", { idioma: "cc", porDefecto: "aa", leer, raiz, avisar: (m) => avisos.push(m) });
  assert.equal(cc.idioma, "aa");
  assert.equal(cc.incidenciaCatalogo.codigo, "catalogo_no_disponible");
  assert.equal(cc.traducir("general.saludo", { nombre: "Ana" }), "Hola, Ana");
  assert.equal(avisos.length, 1);
  await assert.rejects(cargarTextos("otro", { idioma: "bb", porDefecto: "aa", leer, raiz, avisar: () => {} }), /no existe/u);
  assert.deepEqual(leidos.slice(-2), ["bb/otro.json", "aa/otro.json"]);
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

test("el transporte reintenta una vez 502, 503 y caída de red; conserva la incidencia final", async () => {
  const url = new URL("./idioma.js", import.meta.url);
  for (const fallo of [502, 503, new Error("red")]) {
    const llamadas = [];
    const fetchImpl = async (_url, opciones) => {
      llamadas.push(opciones);
      if (llamadas.length === 1) {
        if (fallo instanceof Error) throw fallo;
        return { ok: false, status: fallo };
      }
      return { ok: true, headers: { get: () => null }, text: async () => "{}" };
    };
    assert.equal(await leerPorRed(url, fetchImpl), "{}");
    assert.equal(llamadas.length, 2);
    assert.equal(llamadas[0].credentials, "same-origin");
    assert.equal(llamadas[0].redirect, "error");
    assert.equal(llamadas[0].cache, "no-store");
  }
  let intentos = 0;
  await assert.rejects(leerPorRed(url, async () => { intentos++; return { ok: false, status: 503 }; }), /503/u);
  assert.equal(intentos, 2);
});

test("la lectura de red limita bytes UTF-8 y cancela el flujo demasiado grande", async () => {
  const url = new URL("./idioma.js", import.meta.url);
  await assert.rejects(leerPorRed(url, async () => ({ ok: true, headers: { get: () => null },
    text: async () => "á".repeat(1_100_000) })), /demasiado grande/u);
  let cancelado = false;
  await assert.rejects(leerPorRed(url, async () => ({ ok: true, headers: { get: () => null },
    body: { getReader: () => ({ read: async () => ({ done: false, value: new Uint8Array(2_097_153) }),
      cancel: async () => { cancelado = true; }, releaseLock: () => {} }) } })), /demasiado grande/u);
  assert.equal(cancelado, true);
  await assert.rejects(leerPorRed(url, async () => ({ ok: true, headers: { get: () => null },
    body: { getReader: () => ({ read: async () => ({ done: false, value: new Uint8Array(2_097_153) }),
      cancel: async () => { throw new Error("cancelación fallida"); }, releaseLock: () => {} }) } })),
  (error) => error instanceof AggregateError && error.errors.length === 2
    && /demasiado grande/u.test(error.errors[0].message)
    && /cancelación fallida/u.test(error.errors[1].message));
});

test("JSON roto en el elegido recupera el catálogo por defecto y registra la causa", async () => {
  const leidos = [];
  const leer = async (url) => {
    leidos.push(url.pathname);
    if (url.pathname.includes("/en/")) throw new SyntaxError("JSON inválido");
    return RESPALDO;
  };
  const textos = await cargarTextos("prueba", { idioma: "en", porDefecto: "es", leer,
    raiz: new URL("https://vec.example/textos/"), avisar: () => {} });
  assert.equal(textos.idioma, "es");
  assert.equal(textos.incidenciaCatalogo.causa.name, "SyntaxError");
  assert.deepEqual(leidos.map((ruta) => ruta.match(/\/(es|en)\/prueba/u)[1]), ["en", "es"]);
});

test("catálogo 200 inválido se reintenta y luego usa respaldo válido", async () => {
  for (const malo of [{}, { general: {} }, { general: { saludo: "" } },
    { general: { dias: { one: "día" } } }, { general: { saludo: 3 } }]) {
    assert.equal(esCatalogoValido(malo), false);
    const leidos = [];
    const textos = await cargarTextos("prueba", { idioma: "en", porDefecto: "es",
      leer: async (url) => { leidos.push(url.pathname); return url.pathname.includes("/en/") ? malo : RESPALDO; },
      raiz: new URL("https://vec.example/textos/"), avisar: () => {} });
    assert.deepEqual(leidos.map((ruta) => ruta.match(/\/(es|en)\/prueba/u)[1]), ["en", "en", "es"]);
    assert.equal(textos.idioma, "es");
    assert.equal(textos.incidenciaCatalogo.causa.name, "TypeError");
  }
});

test("catálogo elegido inválido puede recuperarse en su reintento sin pedir respaldo", async () => {
  const leidos = [];
  const textos = await cargarTextos("prueba", { idioma: "en", porDefecto: "es",
    leer: async (url) => { leidos.push(url.pathname); return leidos.length === 1 ? {} : RESPALDO; },
    raiz: new URL("https://vec.example/textos/"), avisar: () => {} });
  assert.deepEqual(leidos.map((ruta) => ruta.match(/\/(es|en)\/prueba/u)[1]), ["en", "en"]);
  assert.equal(textos.idioma, "en");
  assert.equal(textos.incidenciaCatalogo, null);
});

test("catálogo por defecto inválido también rechaza tras un único reintento", async () => {
  const leidos = [];
  await assert.rejects(cargarTextos("prueba", { idioma: "en", porDefecto: "es",
    leer: async (url) => { leidos.push(url.pathname); return {}; },
    raiz: new URL("https://vec.example/textos/"), avisar: () => {} }), /no válido/u);
  assert.deepEqual(leidos.map((ruta) => ruta.match(/\/(es|en)\/prueba/u)[1]), ["en", "en", "es", "es"]);
});

test("los catálogos reales es y en se cargan sin incidencia", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("portal", { idioma, porDefecto: "es" });
    assert.equal(textos.idioma, idioma);
    assert.equal(textos.incidenciaCatalogo, null);
    assert.ok(Object.keys(textos.seccion("general")).length > 0);
  }
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

test("reintentarTextos invalida la lectura previa y devuelve una nueva instancia", async () => {
  const primero = await cargarTextos("portal", { idioma: "es", porDefecto: "es" });
  const segundo = await reintentarTextos("portal", { idioma: "es", porDefecto: "es" });
  assert.notEqual(segundo, primero);
  assert.deepEqual(segundo.seccion("general"), primero.seccion("general"));
});
