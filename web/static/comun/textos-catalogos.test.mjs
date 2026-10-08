import assert from "node:assert/strict";
import { readdir, readFile, stat } from "node:fs/promises";
import test from "node:test";
import { normalizarIndiceIdiomas } from "./idioma.js";
import { esCatalogoValido, esMensajePlural } from "./textos.js";
import { LEGADO_CON_DICCIONARIO } from "./textos-legado.test-helper.mjs";
import { esManifiestoPWA, validarManifiestoPWA } from "../pwa/manifiestos-validacion.test-helper.mjs";

// i18n puro: los textos visibles viven en `textos/<idioma>/<modulo>.json` y el
// código no contiene diccionarios ni nombra idiomas. Estas pruebas vigilan las
// dos mitades: catálogos completos y coherentes por idioma, y ausencia de
// diccionarios embebidos en el JavaScript servido.
const RAIZ_WEB = new URL("../", import.meta.url);
const RAIZ_TEXTOS = new URL("../textos/", import.meta.url);

async function leerJSON(url) {
  return JSON.parse(await readFile(url, "utf8"));
}

const indice = normalizarIndiceIdiomas(await leerJSON(new URL("idiomas.json", RAIZ_TEXTOS)));
const porDefecto = indice.porDefecto;
const modulos = (await readdir(new URL(`${porDefecto}/`, RAIZ_TEXTOS)))
  .filter((nombre) => nombre.endsWith(".json")).map((nombre) => nombre.slice(0, -".json".length)).sort();

const VARIABLE = /\{([A-Za-z_][A-Za-z0-9_]*)\}/gu;
function variables(mensaje) {
  const textos = typeof mensaje === "string" ? [mensaje] : Object.values(mensaje);
  return [...new Set(textos.flatMap((texto) => [...texto.matchAll(VARIABLE)].map(([, nombre]) => nombre)))].sort();
}

/** Aplana un catálogo a `ruta → mensaje`; falla ante valores que no sean texto, sección o plural. */
function hojas(catalogo, ruta = "", salida = new Map()) {
  for (const [clave, valor] of Object.entries(catalogo)) {
    const camino = ruta ? `${ruta}.${clave}` : clave;
    assert.match(clave, /^[A-Za-z0-9_]+$/u, `clave no simple: ${camino}`);
    if (typeof valor === "string" || esMensajePlural(valor)) {
      const vacios = (typeof valor === "string" ? [valor] : Object.values(valor)).filter((texto) => texto.trim() === "");
      assert.deepEqual(vacios, [], `mensaje vacío: ${camino}`);
      salida.set(camino, valor);
    } else {
      assert.ok(valor && typeof valor === "object" && !Array.isArray(valor), `valor no admitido en ${camino}`);
      hojas(valor, camino, salida);
    }
  }
  return salida;
}

function hojasDeModulo(catalogo, modulo, idioma) {
  if (!esManifiestoPWA(modulo)) return hojas(catalogo);
  validarManifiestoPWA(catalogo, modulo, idioma);
  const { icons, ...mensajes } = catalogo;
  return hojas(mensajes);
}

test("los catálogos generales siguen rechazando arrays fuera del esquema PWA", () => {
  for (const modulo of ["pwa", "pwa-no-registrada", "modulo-ordinario"]) {
    assert.throws(() => hojasDeModulo({ general: { texto: ["no admitido"] } }, modulo, porDefecto), /valor no admitido/u);
  }
});

test("hay al menos un catálogo en el idioma por defecto y un directorio por idioma", async () => {
  assert.ok(modulos.length > 0);
  for (const { codigo } of indice.idiomas) {
    assert.ok((await stat(new URL(`${codigo}/`, RAIZ_TEXTOS))).isDirectory(), `falta textos/${codigo}/`);
  }
});

for (const { codigo } of indice.idiomas) {
  test(`textos/${codigo}: mismos módulos y mismas claves que el idioma por defecto (${porDefecto})`, async () => {
    const propios = (await readdir(new URL(`${codigo}/`, RAIZ_TEXTOS))).filter((n) => n.endsWith(".json")).sort();
    assert.deepEqual(propios, modulos.map((m) => `${m}.json`).sort(), `textos/${codigo}/ debe tener exactamente los módulos de ${porDefecto}`);
    for (const modulo of modulos) {
      if (!esManifiestoPWA(modulo)) {
        assert.equal(esCatalogoValido(await leerJSON(new URL(`${codigo}/${modulo}.json`, RAIZ_TEXTOS))), true,
          `${codigo}/${modulo}.json debe ser un catálogo válido para la carga web`);
      }
      const base = hojasDeModulo(await leerJSON(new URL(`${porDefecto}/${modulo}.json`, RAIZ_TEXTOS)), modulo, porDefecto);
      const traducido = hojasDeModulo(await leerJSON(new URL(`${codigo}/${modulo}.json`, RAIZ_TEXTOS)), modulo, codigo);
      const faltan = [...base.keys()].filter((clave) => !traducido.has(clave));
      const sobran = [...traducido.keys()].filter((clave) => !base.has(clave));
      assert.deepEqual({ faltan, sobran }, { faltan: [], sobran: [] }, `${codigo}/${modulo}.json`);
      for (const [clave, mensaje] of base) {
        const otro = traducido.get(clave);
        assert.equal(typeof otro, typeof mensaje, `${codigo}/${modulo}.json ${clave}: texto frente a plural`);
        assert.deepEqual(variables(otro), variables(mensaje), `${codigo}/${modulo}.json ${clave}: variables {…}`);
      }
    }
  });
}

// --- Diccionarios embebidos en JavaScript ---------------------------------
// Heurística: cinco o más propiedades `clave: "texto con letras y espacios"`
// en un mismo fichero servido, o constantes con sufijo de idioma (`…_ES`).
const PROPIEDAD_CON_TEXTO = /(?:^|[{,\s])(?:"[\w.-]+"|'[\w.-]+'|[A-Za-z_$][\w$]*)\s*:\s*(["'`])(?:(?!\1)[^\\\n]|\\.)*?\p{L}{2,}(?:(?!\1)[^\\\n])*?\s(?:(?!\1)[^\\\n])*?\1/gu;
const NOMBRE_CON_IDIOMA = /\b[A-Z][A-Z0-9_]*_(?:ES|EN)\b/u;
const LOCALIZACION_FIJA = /["'`][a-z]{2,3}-[A-Z]{2}["'`]|(?:idioma|IDIOMA|lang|LANG)\w*\s*[!=]==?\s*["'`][a-z]{2,3}["'`]/u;
// Una localización usada solo para cálculo (p. ej. obtener AAAA-MM-DD con
// formatToParts), nunca para presentar, se marca en línea y queda exenta.
const LOCALIZACION_TECNICA = /\/\* localización técnica \*\/\s*["'`][a-z]{2,3}-[A-Z]{2}["'`]/gu;

function tieneDiccionario(codigo) {
  return (codigo.match(PROPIEDAD_CON_TEXTO)?.length ?? 0) >= 5 || NOMBRE_CON_IDIOMA.test(codigo);
}

async function ficherosJS(directorio, salida = []) {
  for (const entrada of await readdir(directorio, { withFileTypes: true })) {
    const url = new URL(entrada.name + (entrada.isDirectory() ? "/" : ""), directorio);
    if (entrada.isDirectory()) await ficherosJS(url, salida);
    else if (/\.m?js$/u.test(entrada.name) && !/\.test\.|test-helper/u.test(entrada.name)) salida.push(url);
  }
  return salida;
}

const javascript = await ficherosJS(RAIZ_WEB);
const relativa = (url) => url.pathname.slice(RAIZ_WEB.pathname.length);

test("ningún JavaScript nuevo incorpora un diccionario de textos (usar textos/<idioma>/<modulo>.json)", async () => {
  const nuevos = [];
  const yaMigrados = [];
  for (const url of javascript) {
    const ruta = relativa(url);
    const embebido = tieneDiccionario(await readFile(url, "utf8"));
    if (embebido && !LEGADO_CON_DICCIONARIO.has(ruta)) nuevos.push(ruta);
    if (!embebido && LEGADO_CON_DICCIONARIO.has(ruta)) yaMigrados.push(ruta);
  }
  assert.deepEqual(nuevos, [], "diccionarios embebidos fuera de la lista de legado pendiente de migrar");
  // Una entrada obsoleta de la lista no rompe la prueba: se retira al migrar.
  if (yaMigrados.length > 0) console.warn(`retirar de textos-legado.test-helper.mjs: ${yaMigrados.join(", ")}`);
});

test("los módulos migrados no nombran idiomas ni localizaciones en su código", async () => {
  const infracciones = [];
  for (const modulo of modulos) {
    const carpeta = new URL(`portal-empleado/modulos/${modulo}/`, RAIZ_WEB);
    const propios = javascript.filter((url) => url.href.startsWith(carpeta.href));
    for (const url of propios) {
      const codigo = await readFile(url, "utf8");
      if (tieneDiccionario(codigo)) infracciones.push(`${relativa(url)}: diccionario embebido`);
      if (LOCALIZACION_FIJA.test(codigo.replace(LOCALIZACION_TECNICA, ""))) infracciones.push(`${relativa(url)}: idioma o localización fijos`);
      if (LEGADO_CON_DICCIONARIO.has(relativa(url))) infracciones.push(`${relativa(url)}: sigue en la lista de legado`);
    }
  }
  assert.deepEqual(infracciones, []);
});

test("la heurística detecta diccionarios y respeta el código sin textos", () => {
  assert.equal(tieneDiccionario('export const M = { a: "Uno dos", b: "Tres cuatro", c: "Cinco seis", d: "Siete ocho", e: "Nueve diez" };'), true);
  assert.equal(tieneDiccionario("export const MENSAJES_ALGO_EN = x;"), true);
  assert.equal(tieneDiccionario('const t = await cargarTextos("cronos"); const x = { a: "uno", b: t.seccion("c") };'), false);
  assert.equal(LOCALIZACION_FIJA.test('locale = "es-ES"'), true);
  assert.equal(LOCALIZACION_FIJA.test('IDIOMA_ACTUAL === "en"'), true);
  assert.equal(LOCALIZACION_FIJA.test("locale = LOCALIZACION_ACTUAL"), false);
  assert.equal(LOCALIZACION_FIJA.test('new Intl.DateTimeFormat(/* localización técnica */ "en-CA", {}'.replace(LOCALIZACION_TECNICA, "")), false);
});
