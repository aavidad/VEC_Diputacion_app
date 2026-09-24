import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("el portal carga el tema común inmediatamente después de sus tokens base", async () => {
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  const estilos = [...html.matchAll(/<link\s+rel="stylesheet"\s+href="([^"]+)"/gu)]
    .map(([, href]) => href);
  const base = estilos.findIndex((href) => href.startsWith("/portal-empleado/portal.css?"));
  assert.notEqual(base, -1, "falta portal.css");
  assert.match(estilos[base + 1] ?? "", /^\/comun\/tema-vec\.css\?v=[A-Za-z0-9-]+$/u);
  assert.equal(estilos.filter((href) => href.startsWith("/comun/tema-vec.css?")).length, 1);

  const manifiesto = await readFile(new URL("../../interno.manifest", import.meta.url), "utf8");
  assert.ok(manifiesto.split(/\r?\n/u).includes("static/comun/tema-vec.css"));
  assert.doesNotMatch(html, /<script[^>]+src="\/comun\/tema-vec\.js/u,
    "la vista ADMIN importa el controlador cuando se necesita");
});
