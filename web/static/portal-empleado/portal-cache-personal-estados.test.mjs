import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
const leer = (ruta) => readFile(new URL(ruta, raiz), "utf8");

test("Personal comparte la cadena desde HTML hasta su catálogo", async () => {
  const [html, cache, portal, coordinador, vista] = await Promise.all([
    "index.html", "cache-publica-v1.json", "portal.js", "portal-modulos-coordinador.js",
    "modulos/personal/vista.js",
  ].map(leer));
  exigirMismaVersion([html, cache], "portal.js");
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  versionDe(coordinador, "./modulos/personal/vista.js");
  versionDe(vista, "./i18n.js");
});

test("producto e interno empaquetan Personal sin rutas de presentación internas", async () => {
  const recursos = ["portal.js", "portal-modulos-coordinador.js", "modulos/personal/vista.js", "modulos/personal/i18n.js"];
  for (const nombre of ["produccion.manifest", "interno.manifest"]) {
    const entradas = (await leer(`../../${nombre}`)).split(/\r?\n/u).map((linea) => linea.trim());
    for (const recurso of recursos)
      assert.ok(entradas.includes(`static/portal-empleado/${recurso}`), `${nombre}: falta ${recurso}`);
    if (nombre === "interno.manifest")
      assert.deepEqual(entradas.filter((ruta) => /(?:^|[/-])presentacion(?:[./-]|$)/u.test(ruta)), []);
  }
});
