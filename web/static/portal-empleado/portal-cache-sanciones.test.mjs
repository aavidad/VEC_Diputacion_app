import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

test("Bolsa comparte módulos y difiere su descarga hasta abrir la pantalla", async () => {
  const [html, portal, api, panel, sanciones] = await Promise.all([
    "index.html", "portal.js", "portal-bolsas-api.js", "portal-panel-interno.js", "portal-bolsas-sanciones.js",
  ].map((nombre) => readFile(new URL(nombre, import.meta.url), "utf8")));
  exigirMismaVersion([html, portal, api, panel, sanciones], "portal-i18n.js");
  exigirMismaVersion([api, panel], "portal-bolsas-sanciones.js");
  versionDe(portal, "portal-bolsas-api.js");
  versionDe(portal, "portal-panel-interno.js");
  versionDe(html, "portal.js");
  for (const modulo of ["portal-bolsas-sanciones.js", "portal-bolsas-api.js", "portal-panel-interno.js"]) {
    assert.doesNotMatch(html, new RegExp(`<link rel="modulepreload" href="[^"]*${modulo.replace(".", "\\.")}`),
      `${modulo} se carga al abrir Bolsa`);
  }
});
