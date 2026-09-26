import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raizWeb = new URL("../../", import.meta.url);
const versionI18n = "20260926-convoca-f1-v2";
const versionI18nIndice = "20260926-convoca-f1-v2";
const versionControlador = "20260926-convoca-f1-v2";
const versionLista = "20260924-b10-reintento-foco-v1";
const versionAnterior = "20260924-bolsa-publica-final";
const versionListaAnterior = "20260924-bolsa-ayuda-v3";

function scriptsDelHTML(html) {
  return [...html.matchAll(/<script\b[^>]*\bsrc="([^"]+)"[^>]*><\/script>/gu)]
    .map((coincidencia) => coincidencia[1]);
}

async function manifiesto(nombre) {
  return new Set((await readFile(new URL(nombre, raizWeb), "utf8")).trim().split(/\r?\n/u));
}

test("una caché immutable previa solicita el catálogo y los controladores F2 por URL nueva", async () => {
  const anterior = new Map([
    [`/bolsa/i18n-publica.js?v=${versionAnterior}`, "/* catálogo anterior */"],
    [`/bolsa/bolsa.js?v=${versionAnterior}`, "/* controlador anterior */"],
    [`/bolsa/lista-bolsas.js?v=${versionListaAnterior}`, "/* lista anterior */"],
  ]);
  const esperados = new Map([
    ["index.html", [
      `/bolsa/i18n-publica.js?v=${versionI18nIndice}`,
      `/bolsa/bolsa.js?v=${versionControlador}`,
    ]],
    ["listas.html", [
      `/bolsa/i18n-publica.js?v=${versionI18n}`,
      `/bolsa/lista-bolsas.js?v=${versionLista}`,
    ]],
  ]);
  const descargas = new Set();
  const usadosDesdeCacheAnterior = new Set();

  for (const [pagina, urlsEsperadas] of esperados) {
    const html = await readFile(new URL(`static/bolsa/${pagina}`, raizWeb), "utf8");
    const scripts = scriptsDelHTML(html);
    const activosCambiados = scripts.filter((url) =>
      urlsEsperadas.some((esperado) => url.split("?", 1)[0] === esperado.split("?", 1)[0]));
    assert.deepEqual(activosCambiados, urlsEsperadas, `${pagina}: i18n precede al controlador`);
    for (const url of activosCambiados) {
      if (anterior.has(url)) {
        usadosDesdeCacheAnterior.add(url);
      } else {
        const ruta = `static${url.split("?", 1)[0]}`;
        assert.ok((await readFile(new URL(ruta, raizWeb), "utf8")).length > 0, `${ruta}: activo disponible`);
        descargas.add(url);
      }
    }
    for (const urlAntigua of anterior.keys()) {
      assert.ok(!scripts.includes(urlAntigua), `${pagina}: no solicita ${urlAntigua}`);
    }
  }

  assert.deepEqual(usadosDesdeCacheAnterior, new Set(), "ningún script cambiado reutiliza los bytes antiguos");
  assert.deepEqual(descargas, new Set([...esperados.values()].flat()), "se descargan las tres URL renovadas");
});

test("las páginas y los scripts renovados constan en ambos manifiestos de producto", async () => {
  const [publico, produccion] = await Promise.all([
    manifiesto("publico.manifest"),
    manifiesto("produccion.manifest"),
  ]);
  for (const ruta of [
    "static/bolsa/index.html",
    "static/bolsa/listas.html",
    "static/bolsa/i18n-publica.js",
    "static/bolsa/bolsa.js",
    "static/bolsa/lista-bolsas.js",
  ]) {
    assert.ok(publico.has(ruta), `${ruta}: manifiesto público`);
    assert.ok(produccion.has(ruta), `${ruta}: manifiesto productivo`);
    assert.ok((await readFile(new URL(ruta, raizWeb), "utf8")).length > 0, `${ruta}: archivo existente`);
  }
});
