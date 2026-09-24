import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

const raizWeb = new URL("../../", import.meta.url);
const paginas = ["organizacion", "peticiones-centro"];
const rutaTema = "/comun/tema-vec.css";

function estilos(html) {
  return [...html.matchAll(/<link\s+rel="stylesheet"\s+href="([^"]+)"/gu)]
    .map(([, href]) => href);
}

function urlEstilo(href, pagina) {
  return new URL(href, `https://vec.test/portal-empleado/${pagina}/`);
}

test("las páginas internas cargan una sola hoja común tras portal.css y comparten versión F2", async () => {
  const portal = await readFile(new URL("static/portal-empleado/index.html", raizWeb), "utf8");
  const versionPortal = estilos(portal)
    .map((href) => new URL(href, "https://vec.test/"))
    .find((url) => url.pathname === rutaTema)?.searchParams.get("v");
  assert.match(versionPortal ?? "", /^[A-Za-z0-9-]+$/u, "el portal principal debe versionar el tema");

  for (const pagina of paginas) {
    const html = await readFile(new URL(`static/portal-empleado/${pagina}/index.html`, raizWeb), "utf8");
    const urls = estilos(html).map((href) => urlEstilo(href, pagina));
    const rutas = urls.map((url) => url.pathname);
    const base = rutas.indexOf("/portal-empleado/portal.css");
    assert.notEqual(base, -1, `${pagina}: falta portal.css`);
    assert.equal(rutas[base + 1], rutaTema, `${pagina}: tema fuera de orden`);
    assert.equal(rutas.filter((ruta) => ruta === rutaTema).length, 1, `${pagina}: tema duplicado`);
    assert.equal(new Set(rutas).size, rutas.length, `${pagina}: ruta CSS duplicada`);
    assert.equal(urls[base + 1].searchParams.get("v"), versionPortal,
      `${pagina}: versión del tema distinta de la compartida`);
  }
});

test("todas las hojas de ambas páginas existen y están en los dos manifiestos", async () => {
  const manifiestos = await Promise.all(["interno.manifest", "produccion.manifest"]
    .map(async (nombre) => (await readFile(new URL(nombre, raizWeb), "utf8"))
      .split(/\r?\n/u).map((linea) => linea.trim()).filter((linea) => linea && !linea.startsWith("#"))));
  for (const entradas of manifiestos) {
    assert.equal(entradas.filter((ruta) => ruta === "static/comun/tema-vec.css").length, 1,
      "el tema debe figurar una vez por manifiesto");
  }

  for (const pagina of paginas) {
    const html = await readFile(new URL(`static/portal-empleado/${pagina}/index.html`, raizWeb), "utf8");
    for (const href of estilos(html)) {
      const url = urlEstilo(href, pagina);
      const ruta = `static${url.pathname}`;
      assert.ok(manifiestos.every((entradas) => entradas.includes(ruta)),
        `${pagina}: ${ruta} debe figurar en ambos manifiestos`);
      await access(new URL(ruta, raizWeb));
    }
  }
});
