import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarVistaOportunidades } from "../comun/oportunidades/vista.js";
import { traducir } from "./i18n.js";

test("B15 tiene ruta propia, título traducido y montaje que se desmonta al navegar", async () => {
  const [html, aplicacion] = await Promise.all([
    readFile(new URL("./index.html", import.meta.url), "utf8"),
    readFile(new URL("./aplicacion.js", import.meta.url), "utf8"),
  ]);
  assert.match(html, /href="\?vista=oportunidades" data-ruta="oportunidades"/);
  assert.match(html, /data-i18n="areaPersonal\.rutas\.oportunidades"/);
  assert.equal(traducir("areaPersonal.rutas.oportunidades"), "Oportunidades para ti");
  assert.match(aplicacion, /oportunidades: \["areaPersonal\.rutas\.oportunidades"/);
  assert.match(aplicacion, /estado\.desmontarOportunidades\?\.\(\);[\s\S]*espacio-trabajo"\)\.innerHTML/);
  assert.match(aplicacion, /montarVistaOportunidades\(\{ raiz: porId\("oportunidades-montaje"\), anunciar \}\)/);
});

test("sin proyección autorizada, el montaje queda no configurado y no ofrece solicitud", () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener(nombre, fn) { eventos.set(nombre, fn); }, removeEventListener(nombre) { eventos.delete(nombre); }, replaceChildren() { this.innerHTML = ""; } };
  const vista = montarVistaOportunidades({ raiz });
  assert.match(raiz.innerHTML, /data-estado="no_configurado"/);
  assert.doesNotMatch(raiz.innerHTML, /Iniciar solicitud precompletada|href="\/bolsa\//);
  vista.desmontar();
  assert.equal(raiz.innerHTML, "");
  assert.equal(eventos.size, 0);
});
