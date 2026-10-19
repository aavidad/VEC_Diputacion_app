import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const leer = (ruta) => readFile(new URL(ruta, import.meta.url), "utf8");

test("OSM comparte la versión de mapa entre importadores y conserva las hojas", async () => {
  const [html, portal, coordinador, recorridos, borradores] = await Promise.all([
    "index.html", "portal.js", "portal-modulos-coordinador.js",
    "modulos/dietas/vista-recorridos.js", "modulos/dietas/vista-borradores-propios.js",
  ].map(leer));
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  versionDe(coordinador, "./modulos/dietas/mapa-ruta.js");
  versionDe(recorridos, "./vista-borradores-propios.js");
  versionDe(borradores, "./vista-mapa-comision.js");
  versionDe(html, "/portal-empleado/modulos/dietas/dietas.css");
  assert.match(await leer("modulos/dietas/mapa-ruta.js"), /OpenStreetMap/u);
});
