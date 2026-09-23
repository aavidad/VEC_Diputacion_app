import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

const raizWeb = new URL("../../", import.meta.url);
const necesarios = [
  "static/comun/tema-vec.css",
  "static/comun/tema-vec.js",
  "static/portal-empleado/modulos/administracion/vista-apariencia.js",
  "static/portal-empleado/portal-i18n-baremacion.js",
  "static/portal-empleado/portal-i18n-contratos.js",
  "static/portal-empleado/portal-i18n-convocatorias.js",
  "static/portal-empleado/modulos/cronos/i18n-permisos.js",
  "static/portal-empleado/modulos/dietas/i18n-borradores.js",
  "static/portal-empleado/modulos/dietas/i18n-revision.js",
  "static/portal-empleado/portal-baremacion.css",
  "static/portal-empleado/portal-contratos.css",
  "static/portal-empleado/portal-convocatorias.css",
  "static/portal-empleado/modulos/cronos/permisos.css",
  "static/portal-empleado/modulos/dietas/borradores-propios.css",
];

test("el montaje F2 declara una sola vez sus recursos internos reales", async () => {
  const manifiesto = (await readFile(new URL("interno.manifest", raizWeb), "utf8")).trim().split(/\r?\n/u);
  assert.equal(new Set(manifiesto).size, manifiesto.length, "hay rutas duplicadas en el manifiesto interno");
  for (const ruta of necesarios) {
    assert.equal(manifiesto.filter((entrada) => entrada === ruta).length, 1, `${ruta} debe figurar una vez`);
    await access(new URL(ruta, raizWeb));
  }
});

test("los estilos F2 cargan una vez tras sus bases y todos están empaquetados", async () => {
  const [html, textoManifiesto] = await Promise.all([
    readFile(new URL("static/portal-empleado/index.html", raizWeb), "utf8"),
    readFile(new URL("interno.manifest", raizWeb), "utf8"),
  ]);
  const manifiesto = new Set(textoManifiesto.trim().split(/\r?\n/u));
  const estilos = [...html.matchAll(/<link\s+rel="stylesheet"\s+href="([^"]+)"/gu)]
    .map(([, href]) => href.split("?")[0]);
  assert.equal(new Set(estilos).size, estilos.length, "hay hojas CSS enlazadas dos veces");
  for (const ruta of estilos) {
    assert.ok(manifiesto.has(`static${ruta}`), `${ruta} debe estar empaquetada`);
    await access(new URL(`static${ruta}`, raizWeb));
  }
  const posicion = (ruta) => estilos.indexOf(ruta);
  assert.equal(posicion("/comun/tema-vec.css"), posicion("/portal-empleado/portal.css") + 1);
  for (const nombre of ["baremacion", "contratos", "convocatorias"]) {
    const ruta = `/portal-empleado/portal-${nombre}.css`;
    assert.notEqual(posicion(ruta), -1, `${ruta} debe estar enlazada`);
    assert.ok(posicion(ruta) > posicion("/portal-empleado/portal-menu-bolsa.css"));
  }
  for (const [base, extension] of [
    ["cronos/cronos.css", "cronos/permisos.css"],
    ["dietas/dietas.css", "dietas/borradores-propios.css"],
  ]) {
    assert.equal(posicion(`/portal-empleado/modulos/${extension}`),
      posicion(`/portal-empleado/modulos/${base}`) + 1);
  }
});
