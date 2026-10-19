import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const leer = (ruta) => readFile(new URL(ruta, import.meta.url), "utf8");

test("paradas, periodos y sus hojas comparten la versión calculada", async () => {
  const [html, portal, coordinador, i18nDietas, borradores, mapa, recorridos] = await Promise.all([
    "index.html", "portal.js", "portal-modulos-coordinador.js", "modulos/dietas/i18n.js",
    "modulos/dietas/vista-borradores-propios.js", "modulos/dietas/mapa-ruta.js",
    "modulos/dietas/vista-recorridos.js",
  ].map(leer));
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  exigirMismaVersion([borradores, mapa, recorridos], "i18n.js");
  exigirMismaVersion([i18nDietas, borradores], "i18n-borradores.js");
  versionDe(coordinador, "./modulos/dietas/mapa-ruta.js");
  versionDe(html, "/portal-empleado/modulos/dietas/borradores-propios.css");
  versionDe(html, "/portal-empleado/modulos/cronos/cronos.css");
});
