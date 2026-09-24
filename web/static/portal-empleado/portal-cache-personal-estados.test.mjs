import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const versionNueva = "20260924-f2-personal-estados-v4";
const versionEntrada = "20260924-web-c-v2";
const versionI18n = "20260924-personal-interno-estados-v1";

function versiones(codigo, recurso) {
  const nombre = recurso.replaceAll(".", "\\.");
  return [...codigo.matchAll(new RegExp(`${nombre}\\?v=([^"']+)`, "gu"))].map((resultado) => resultado[1]);
}

test("una caché caliente descarga el grafo renovado hasta Personal i18n", async () => {
  const cache = new Map([
    ["/portal-empleado/portal.js?v=20260924-p1-personal-interno-v2", "/* portal anterior */"],
    ["/portal-empleado/portal-modulos-coordinador.js?v=20260924-p1-personal-interno-v2", "/* coordinador anterior */"],
    [`/portal-empleado/portal.js?v=${versionNueva}`, "/* portal anterior de Personal */"],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${versionNueva}`, "/* coordinador anterior de Personal */"],
    ["/portal-empleado/portal.js?v=20260924-f2-cronos-permisos-v1", "/* portal anterior de Cronos */"],
    ["/portal-empleado/portal-modulos-coordinador.js?v=20260924-f2-cronos-permisos-v1", "/* coordinador anterior de Cronos */"],
    ["/portal-empleado/portal.js?v=20260924-f2-cronos-permisos-v2", "/* portal anterior de Cronos */"],
    ["/portal-empleado/portal-modulos-coordinador.js?v=20260924-f2-cronos-permisos-v2", "/* coordinador anterior de Cronos */"],
    ["/portal-empleado/portal.js?v=20260924-f2-dietas-consulta-v2", "/* portal anterior de Dietas */"],
    ["/portal-empleado/portal-modulos-coordinador.js?v=20260924-f2-dietas-consulta-v2", "/* coordinador anterior de Dietas */"],
    ["/portal-empleado/modulos/personal/vista.js?v=20260924-f2-cache-v3", "/* vista anterior */"],
    ["/portal-empleado/modulos/personal/i18n.js?v=20260924-f2-web2", "/* i18n anterior */"],
  ]);
  const urlsAntiguas = new Set(cache.keys());
  const descargas = new Set();
  const hitsAntiguos = [];
  async function cargar(recurso, version) {
    const url = `/portal-empleado/${recurso}?v=${version}`;
    if (urlsAntiguas.has(url)) hitsAntiguos.push(url);
    if (!cache.has(url)) {
      cache.set(url, await readFile(new URL(recurso, raiz), "utf8"));
      descargas.add(url);
    }
    return cache.get(url);
  }

  const html = await readFile(new URL("index.html", raiz), "utf8");
  assert.deepEqual(versiones(html, "/portal-empleado/portal.js"), [versionEntrada]);
  const portal = await cargar("portal.js", versionEntrada);
  assert.deepEqual(versiones(portal, "./portal-modulos-coordinador.js"), [versionEntrada]);
  const coordinador = await cargar("portal-modulos-coordinador.js", versionEntrada);
  assert.deepEqual(versiones(coordinador, "./modulos/personal/vista.js"), [versionNueva, versionNueva],
    "presentación e interno usan la misma vista renovada");
  const vista = await cargar("modulos/personal/vista.js", versionNueva);
  assert.deepEqual(versiones(vista, "./i18n.js"), [versionI18n]);
  await cargar("modulos/personal/i18n.js", versionI18n);

  assert.deepEqual(descargas, new Set([
    `/portal-empleado/portal.js?v=${versionEntrada}`,
    `/portal-empleado/portal-modulos-coordinador.js?v=${versionEntrada}`,
    `/portal-empleado/modulos/personal/vista.js?v=${versionNueva}`,
    `/portal-empleado/modulos/personal/i18n.js?v=${versionI18n}`,
  ]));
  assert.deepEqual(hitsAntiguos, [], "ninguna respuesta immutable antigua se reutiliza");
});

test("producto e interno empaquetan la cadena sin rutas de presentación internas", async () => {
  const recursos = ["portal.js", "portal-modulos-coordinador.js", "modulos/personal/vista.js", "modulos/personal/i18n.js"];
  for (const nombre of ["produccion.manifest", "interno.manifest"]) {
    const texto = await readFile(new URL(`../../${nombre}`, raiz), "utf8");
    const entradas = texto.split(/\r?\n/u).map((linea) => linea.trim()).filter((linea) => linea && !linea.startsWith("#"));
    for (const recurso of recursos) {
      assert.ok(entradas.includes(`static/portal-empleado/${recurso}`), `${nombre}: falta ${recurso}`);
    }
    if (nombre === "interno.manifest") {
      assert.deepEqual(entradas.filter((ruta) => /(?:^|[/-])presentacion(?:[./-]|$)/u.test(ruta)), [],
        "el paquete interno excluye los activos de presentación");
    }
  }
});
