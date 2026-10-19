import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { PLANTILLA_TESELAS_OSM_INTERNA } from "./modulos/dietas/contrato.js";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
const leer = (ruta) => readFile(new URL(ruta, raiz), "utf8");

test("las entradas y las vistas comparten la versión calculada", async () => {
  const [html, cache, portal, coordinador, dietas] = await Promise.all([
    "index.html", "cache-publica-v1.json", "portal.js", "portal-modulos-coordinador.js",
    "modulos/dietas/vista-recorridos.js",
  ].map(leer));
  exigirMismaVersion([html, cache], "portal.js");
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  versionDe(coordinador, "./modulos/dietas/mapa-ruta.js");
  versionDe(dietas, "./vista-borradores-propios.js");
  for (const nombre of ["vista-calendario", "vista-correcciones", "vista-catalogo-permisos", "vista-notificaciones"])
    versionDe(html, `/portal-empleado/modulos/cronos/${nombre}.css`);
});

test("los manifiestos conservan vistas de Cronos y mapa de Dietas", async () => {
  const cronos = ["i18n-c4.js", "i18n-c5.js", "i18n-c6.js", "i18n-c9.js",
    ...["vista-calendario", "vista-correcciones", "vista-catalogo-permisos", "vista-notificaciones"]
      .flatMap((nombre) => [`${nombre}.js`, `${nombre}.css`])];
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const entradas = (await leer(`../../${nombre}`)).split(/\r?\n/u);
    for (const recurso of cronos) assert.ok(entradas.includes(`static/portal-empleado/modulos/cronos/${recurso}`), recurso);
    for (const recurso of ["mapa-ruta.js", "vista-mapa-comision.js"])
      assert.ok(entradas.includes(`static/portal-empleado/modulos/dietas/${recurso}`), recurso);
    for (const recurso of ["i18n-d1.js", "i18n-d4.js", "vista-acceso-papeles.js", "vista-itinerario.js"])
      assert.ok(!entradas.includes(`static/portal-empleado/modulos/dietas/${recurso}`), recurso);
  }
  assert.match(await leer("modulos/dietas/mapa-ruta.js"), /OpenStreetMap/u);
  assert.equal(PLANTILLA_TESELAS_OSM_INTERNA, "/tiles/osm/{z}/{x}/{y}.png");
});
