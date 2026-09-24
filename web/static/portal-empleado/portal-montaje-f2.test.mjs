import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

const raizWeb = new URL("../../", import.meta.url);
const necesarios = [
  "static/comun/tema-vec.css",
  "static/comun/tema-vec.js",
  "static/portal-empleado/modulos/administracion/vista-apariencia.js",
  "static/portal-empleado/portal-i18n-contratos.js",
  "static/portal-empleado/modulos/cronos/i18n-permisos.js",
  "static/portal-empleado/modulos/dietas/i18n-borradores.js",
  "static/portal-empleado/modulos/dietas/i18n-revision.js",
  "static/portal-empleado/portal-contratos.css",
  "static/portal-empleado/modulos/cronos/permisos.css",
  "static/portal-empleado/modulos/dietas/borradores-propios.css",
];
const recursosPublicosConsumidos = [
  "static/bolsa/i18n-publica.js",
  "static/area-personal/i18n.js",
];
const recursosOportunidadesPreparados = [
  "static/comun/oportunidades/vista.js",
  "static/comun/oportunidades/i18n.js",
  "static/comun/oportunidades/oportunidades.css",
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
  for (const nombre of ["contratos"]) {
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

test("el producto incluye los activos F2 y solo prepara oportunidades en la composición integrada", async () => {
  const [texto, textoPublico, textoInterno] = await Promise.all([
    readFile(new URL("produccion.manifest", raizWeb), "utf8"),
    readFile(new URL("publico.manifest", raizWeb), "utf8"),
    readFile(new URL("interno.manifest", raizWeb), "utf8"),
  ]);
  const entradas = texto.trim().split(/\r?\n/u);
  const manifiesto = new Set(entradas);
  const publico = new Set(textoPublico.trim().split(/\r?\n/u));
  const interno = new Set(textoInterno.trim().split(/\r?\n/u));
  assert.equal(manifiesto.size, entradas.length, "el manifiesto productivo no admite duplicados");
  for (const ruta of [...necesarios, ...recursosPublicosConsumidos, ...recursosOportunidadesPreparados]) {
    assert.ok(manifiesto.has(ruta), `${ruta} debe figurar en producto`);
    await access(new URL(ruta, raizWeb));
  }
  for (const ruta of recursosOportunidadesPreparados) {
    assert.ok(!publico.has(ruta), `${ruta} no debe figurar en público`);
    assert.ok(!interno.has(ruta), `${ruta} no debe figurar en interno`);
  }
  for (const ruta of ["static/comun/oportunidades/vista.test.mjs"]) {
    assert.ok(!manifiesto.has(ruta), `${ruta} no debe figurar en producto`);
    assert.ok(!publico.has(ruta), `${ruta} no debe figurar en público`);
    assert.ok(!interno.has(ruta), `${ruta} no debe figurar en interno`);
  }
});

test("los catálogos públicos y F2 responden a imports o scripts existentes", async () => {
  const consumidores = new Map([
    ["static/bolsa/i18n-publica.js", ["static/bolsa/index.html", "static/bolsa/listas.html"]],
    ["static/area-personal/i18n.js", ["static/area-personal/arranque.js"]],
    ["static/comun/tema-vec.js", ["static/portal-empleado/modulos/administracion/vista-apariencia.js"]],
    ["static/portal-empleado/modulos/administracion/vista-apariencia.js", ["static/portal-empleado/modulos/administracion/vista.js"]],
    ["static/portal-empleado/portal-i18n-contratos.js", ["static/portal-empleado/portal-vistas-operaciones.js"]],
    ["static/portal-empleado/modulos/dietas/i18n-borradores.js", ["static/portal-empleado/modulos/dietas/vista-borradores-propios.js"]],
    ["static/portal-empleado/modulos/dietas/i18n-revision.js", ["static/portal-empleado/modulos/dietas/i18n.js"]],
  ]);
  for (const [recurso, origenes] of consumidores) {
    for (const origen of origenes) {
      const codigo = await readFile(new URL(origen, raizWeb), "utf8");
      assert.ok(codigo.includes(recurso.slice(recurso.lastIndexOf("/") + 1)), `${origen} debe consumir ${recurso}`);
    }
  }
});
