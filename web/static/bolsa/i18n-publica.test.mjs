import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";

const base = new URL("./", import.meta.url);
const [catalogo, indice, listas, convocatorias, bolsas] = await Promise.all([
  readFile(new URL("i18n-publica.js", base), "utf8"),
  readFile(new URL("index.html", base), "utf8"),
  readFile(new URL("listas.html", base), "utf8"),
  readFile(new URL("bolsa.js", base), "utf8"),
  readFile(new URL("lista-bolsas.js", base), "utf8"),
]);

test("las dos superficies públicas cargan el catálogo común antes de sus controladores", () => {
  for (const [html, controlador] of [[indice, "bolsa.js"], [listas, "lista-bolsas.js"]]) {
    assert.ok(html.indexOf("i18n-publica.js") < html.indexOf(controlador));
  }
  assert.match(convocatorias, /VECBolsaI18n\?\.t/);
  assert.match(bolsas, /VECBolsaI18n/);
});

test("el catálogo público devuelve castellano y falla cerrado en claves desconocidas", () => {
  const contexto = { globalThis: {} };
  vm.runInNewContext(catalogo, contexto);
  const { t } = contexto.globalThis.VECBolsaI18n;
  assert.equal(t("cargando_convocatorias"), "Cargando convocatorias…");
  assert.equal(t("documento_formato"), "El documento debe tener formato ***1234** (3 asteriscos, 4 dígitos y 2 asteriscos).");
  assert.equal(t("clave_ajena"), "clave_ajena");
});

test("el inglés público conserva claves y marcadores y localiza números", async () => {
  const contexto = { URL, globalThis: {
    location: { href: "https://vec.example/bolsa/?lang=en" },
    navigator: { languages: ["es-ES"] },
  } };
  vm.runInNewContext(catalogo, contexto);
  const { idioma, localizacion, t, numero, plural, mensajesES, mensajesEN } = contexto.globalThis.VECBolsaI18n;
  assert.equal(idioma, "en");
  const indiceIdiomas = JSON.parse(await readFile(new URL("../textos/idiomas.json", base), "utf8"));
  assert.equal(localizacion, indiceIdiomas.idiomas.find((entrada) => entrada.codigo === idioma).localizacion);
  assert.deepEqual(Object.keys(mensajesEN), Object.keys(mensajesES));
  const marcadores = (valor) => [...valor.matchAll(/\{[a-z_]+\}/gu)].map((x) => x[0]).sort();
  for (const clave of Object.keys(mensajesES)) {
    assert.deepEqual(marcadores(mensajesEN[clave]), marcadores(mensajesES[clave]), clave);
    assert.ok(mensajesEN[clave].trim(), clave);
  }
  assert.equal(t("cargando_convocatorias"), "Loading recruitment notices…");
  assert.equal(numero(1234), "1,234");
  assert.equal(plural("requisito", 2), "2 requirements");
});

test("el navegador elige inglés si no hay parámetro y rechaza idiomas ajenos", () => {
  const ingles = { URL, globalThis: { location: { href: "https://vec.example/bolsa/" }, navigator: { languages: ["fr-FR", "en-GB"] } } };
  vm.runInNewContext(catalogo, ingles);
  assert.equal(ingles.globalThis.VECBolsaI18n.idioma, "en");
  const ajeno = { URL, globalThis: { location: { href: "https://vec.example/bolsa/?lang=../../privado" }, navigator: { languages: ["fr-FR"] } } };
  vm.runInNewContext(catalogo, ajeno);
  assert.equal(ajeno.globalThis.VECBolsaI18n.idioma, "es");
});

test("la localización no cambia pushState ni popstate", () => {
  assert.match(convocatorias, /history\.pushState/);
  assert.match(convocatorias, /addEventListener\("popstate"/);
  assert.match(bolsas, /history\.pushState/);
  assert.match(bolsas, /addEventListener\("popstate"/);
});
