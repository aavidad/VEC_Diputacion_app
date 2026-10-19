import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const leer = (ruta) => readFile(new URL(ruta, import.meta.url), "utf8");

test("Dietas interno alcanza la vista y sus clientes reales con una URL por recurso", async () => {
  const [html, portal, coordinador, cargador, recorridos] = await Promise.all([
    "index.html", "portal.js", "portal-modulos-coordinador.js", "portal-modulos-coordinador.js",
    "modulos/dietas/vista-recorridos.js",
  ].map(leer));
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  versionDe(coordinador, "./portal-composicion-empleado.js");
  for (const recurso of ["vista-recorridos.js", "mapa-ruta.js", "cliente-borradores-http.js"])
    versionDe(cargador, `./modulos/dietas/${recurso}`);
  versionDe(recorridos, "./vista-borradores-propios.js");
  assert.doesNotMatch(cargador, /vista-itinerario|adaptador-presentacion|datos-presentacion|calculador-rutas-presentacion/u);
  assert.doesNotMatch(recorridos, /vista-itinerario\.js|montarMapaInicialGranadaDietas/u);
});
