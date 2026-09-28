import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const leer = (nombre) => readFile(new URL(nombre, import.meta.url), "utf8");

// Estas aristas son las que mantienen vivo el JS anterior con Cache-Control:
// immutable si solo se renueva el fichero hoja.
test("el corte RRHH renueva las rutas de B55 y CT133 hasta la entrada HTML", async () => {
  const [operaciones, panel, api, vistaCT, coordinador, portal, html] = await Promise.all([
    "portal-bolsas-operaciones.js", "portal-panel-interno.js", "portal-bolsas-api.js",
    "modulos/contratacion-temporal/vista-expedientes.js", "portal-modulos-coordinador.js",
    "portal.js", "index.html",
  ].map(leer));
  const aristas = [
    [operaciones, "portal-bolsas-reincorporaciones.js", "20260928-rrhh-reincorporaciones-v5"],
    [panel, "portal-bolsas-reincorporaciones.js", "20260928-rrhh-reincorporaciones-v5"],
    [api, "portal-bolsas-operaciones.js", "20260928-rrhh-corte3-cache-v5"],
    [vistaCT, "vista-borradores-publicados.js", "20260928-rrhh-corte3-cache-v2"],
    [coordinador, "rrhh-plantillas-vista.js", "20260928-rrhh-corte3-cache-v3"],
    [coordinador, "vista-expedientes.js", "20260928-rrhh-corte3-cache-v5"],
    [portal, "rrhh-plantillas-cliente.js", "20260928-rrhh-corte3-cache-v2"],
    [portal, "portal-modulos-coordinador.js", "20260928-rrhh-corte3-cache-v6"],
    [html, "portal.js", "20260928-rrhh-corte3-cache-v6"],
  ];
  for (const [fuente, modulo, version] of aristas) {
    assert.ok(fuente.includes(`${modulo}?v=${version}`), `${modulo}: falta URL renovada`);
  }
  assert.ok(![operaciones, panel, api, vistaCT, coordinador, portal, html].some((fuente) =>
    fuente.includes("portal-bolsas-reincorporaciones.js?v=20260928-rrhh-reincorporaciones-v4")));
  assert.ok(!coordinador.includes("rrhh-plantillas-vista.js?v=20260928-rrhh-corte3-cache-v2"));
  assert.ok(!portal.includes("portal-modulos-coordinador.js?v=20260928-rrhh-corte3-cache-v5"));
  assert.ok(!html.includes("portal.js?v=20260928-rrhh-corte3-cache-v5"));
});

test("manifiestos conservan las cuatro hojas y la ayuda distribuida", async () => {
  for (const nombre of ["../../interno.manifest", "../../produccion.manifest"]) {
    const lineas = (await leer(nombre)).split(/\r?\n/u);
    for (const ruta of [
      "static/portal-empleado/portal-bolsas-reincorporaciones.js",
      "static/portal-empleado/modulos/contratacion-temporal/rrhh-plantillas-cliente.js",
      "static/portal-empleado/modulos/contratacion-temporal/rrhh-plantillas-vista.js",
      "static/portal-empleado/modulos/contratacion-temporal/vista-borradores-publicados.js",
      "static/portal-empleado/portal-i18n-ayuda.js",
      "static/portal-empleado/ayuda-contenido.js",
    ]) assert.ok(lineas.includes(ruta), `${nombre}: falta ${ruta}`);
  }
});
