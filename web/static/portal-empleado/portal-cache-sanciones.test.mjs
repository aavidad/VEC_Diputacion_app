import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { exigirRenovado } from "./versiones-cache.test-helper.mjs";

test("B24 renueva toda la cadena de caché immutable hasta la entrada del portal", async () => {
  const [html, portal, api, panel, sanciones] = await Promise.all([
    "index.html", "portal.js", "portal-bolsas-api.js", "portal-panel-interno.js", "portal-bolsas-sanciones.js",
  ].map((nombre) => readFile(new URL(nombre, import.meta.url), "utf8")));

  exigirRenovado([html, portal, api, panel, sanciones], "portal-i18n.js", "20260926-pulido-portal-v1");
  const previas = ["20260926-pulido-portal-v1", "20260927-rrhh-sanciones-v1"];
  exigirRenovado([api, panel], "portal-bolsas-sanciones.js", previas);
  exigirRenovado(portal, "portal-bolsas-api.js", previas);
  exigirRenovado(portal, "portal-panel-interno.js", previas);
  for (const modulo of ["portal-bolsas-sanciones.js", "portal-bolsas-api.js", "portal-panel-interno.js"]) {
    assert.doesNotMatch(html, new RegExp(`<link rel="modulepreload" href="[^"]*${modulo.replace(".", "\\.")}`),
      `${modulo} no se precarga al abrir CT`);
  }
  exigirRenovado(html, "portal.js", previas);
});
