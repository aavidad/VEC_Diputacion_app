import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
const leer = (ruta) => readFile(new URL(ruta, raiz), "utf8");

test("la entrada y sus importadores comparten URL para cada módulo", async () => {
  const [html, cache, portal, coordinador, dietas] = await Promise.all([
    "index.html", "cache-publica-v1.json", "portal.js", "portal-modulos-coordinador.js",
    "modulos/dietas/vista-recorridos.js",
  ].map(leer));
  exigirMismaVersion([html, cache], "portal.js");
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  exigirMismaVersion([portal, coordinador], "portal-i18n.js");
  versionDe(dietas, "./vista-borradores-propios.js");
  versionDe(coordinador, "./modulos/dietas/vista-recorridos.js");
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-itinerario\.js/u);
  for (const recurso of ["portal.css", "portal-componentes.css", "modulos/cronos/cronos.css",
    "modulos/dietas/dietas.css", "modulos/personal/ficha-integral.css",
    "modulos/contratacion-temporal/expedientes-operativo.css"]) {
    versionDe(html, `/portal-empleado/${recurso}`);
  }
  await access(new URL("portal-i18n.js", raiz));
});
