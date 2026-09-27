import test from "node:test";
import { readFile } from "node:fs/promises";
import { exigirRenovado } from "./versiones-cache.test-helper.mjs";

test("B24 renueva toda la cadena de caché immutable hasta la entrada del portal", async () => {
  const [html, portal, api, panel] = await Promise.all([
    "index.html", "portal.js", "portal-bolsas-api.js", "portal-panel-interno.js",
  ].map((nombre) => readFile(new URL(nombre, import.meta.url), "utf8")));

  exigirRenovado([api, panel, html], "portal-bolsas-sanciones.js", "20260926-pulido-portal-v1");
  exigirRenovado([portal, html], "portal-bolsas-api.js", "20260926-pulido-portal-v1");
  exigirRenovado([portal, html], "portal-panel-interno.js", "20260926-pulido-portal-v1");
  exigirRenovado(html, "portal.js", "20260926-pulido-portal-v1");
});
