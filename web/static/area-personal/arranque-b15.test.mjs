import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("el HTML y los módulos cambiados usan URLs nuevas bajo caché inmutable", async () => {
  const [html, arranque, aplicacion, vista] = await Promise.all([
    readFile(new URL("./index.html", import.meta.url), "utf8"),
    readFile(new URL("./arranque.js", import.meta.url), "utf8"),
    readFile(new URL("./aplicacion.js", import.meta.url), "utf8"),
    readFile(new URL("../comun/oportunidades/vista.js", import.meta.url), "utf8"),
  ]);
  const version = "20260924-f2-b15-area-v1";
  for (const ruta of ["/area-personal/area-personal.css", "/comun/oportunidades/oportunidades.css", "/area-personal/arranque.js"]) {
    assert.ok(html.includes(`${ruta}?v=${version}`), ruta);
  }
  assert.ok(arranque.includes(`./aplicacion.js?v=${version}`));
  assert.ok(arranque.includes(`./i18n.js?v=${version}`));
  assert.ok(aplicacion.includes(`./i18n.js?v=${version}`));
  assert.ok(aplicacion.includes(`../comun/oportunidades/vista.js?v=${version}`));
  assert.match(vista, /\.\/i18n\.js\?v=20260924-f2-web2/);
  assert.match(arranque, /\.\/cliente-http\.js\?v=20260924-f2-b11-v1/);
});

test("un fallo de arranque usa i18n genérico y no expone su causa", async () => {
  const arranque = await readFile(new URL("./arranque.js", import.meta.url), "utf8");
  assert.match(arranque, /catch \{[\s\S]*traducir\("areaPersonal\.estado\.error\.titulo"\)/);
  assert.match(arranque, /traducir\("areaPersonal\.estado\.error\.detalle"\)/);
  assert.match(arranque, /traducir\("areaPersonal\.estado\.error\.carga\.garantia"\)/);
  assert.doesNotMatch(arranque, /error\.message|error instanceof Error/);
});
