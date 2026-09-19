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
  assert.match(bolsas, /VECBolsaI18n\?\.t/);
});

test("el catálogo público devuelve castellano y falla cerrado en claves desconocidas", () => {
  const contexto = { globalThis: {} };
  vm.runInNewContext(catalogo, contexto);
  const { t } = contexto.globalThis.VECBolsaI18n;
  assert.equal(t("cargando_convocatorias"), "Cargando convocatorias…");
  assert.equal(t("documento_formato"), "El documento debe tener formato ***1234** (3 asteriscos, 4 dígitos y 2 asteriscos).");
  assert.equal(t("clave_ajena"), "clave_ajena");
});

test("la localización no cambia pushState ni popstate", () => {
  assert.match(convocatorias, /history\.pushState/);
  assert.match(convocatorias, /addEventListener\("popstate"/);
  assert.match(bolsas, /history\.pushState/);
  assert.match(bolsas, /addEventListener\("popstate"/);
});
