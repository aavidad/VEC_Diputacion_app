import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarVistaCorreccionesCronos, renderizarCorreccionesCronos } from "./vista-correcciones.js";

test("C5 no simula una solicitud y distingue estados de consulta", () => {
  const html = renderizarCorreccionesCronos();
  assert.match(html, /data-cronos-c5-estado="no_configurado"/u);
  assert.match(html, /Solicitar corrección<\/button>/u);
  assert.match(html, /disabled aria-disabled="true" aria-describedby="cronos-c5-sin-envio"/u);
  assert.match(html, /servicio de correcciones no disponible/u);
  assert.doesNotMatch(html, /<form|<input|<textarea|recibo:|Preparación local|borrador/iu);
  assert.match(renderizarCorreccionesCronos({ estado: "error" }), /No se pudieron consultar los movimientos propios/u);
  assert.match(renderizarCorreccionesCronos({ estado: "vacio" }), /No hay movimientos propios/u);
  assert.match(renderizarCorreccionesCronos({ estado: "denegado" }), /Consulta de movimientos no autorizada/u);
  assert.throws(() => renderizarCorreccionesCronos({ estado: "inventado" }), /estado C5/u);
});

test("no muestra fichajes sin trámite y escapa el catálogo", () => {
  const html = renderizarCorreccionesCronos({
    estado: "disponible",
    fichajes: [{ id: "fichaje:propio-001", instante: "2026-09-24T08:00:00Z" }],
    mensajes: { c5_titulo: '<img src=x onerror="x()">' },
  });
  assert.doesNotMatch(html, /fichaje:propio-001|<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;x\(\)&quot;&gt;/u);
});

test("montaje y desmontaje no conservan datos", () => {
  const contenedor = { dataset: {}, innerHTML: "", remove() { this.retirado = true; } };
  let desmontar;
  montarVistaCorreccionesCronos({
    raiz: { ownerDocument: { createElement: () => contenedor }, append() {} },
    registrarDesmontar: (fn) => { desmontar = fn; },
  });
  assert.match(contenedor.innerHTML, /Olvido de marcaje/u);
  desmontar();
  assert.equal(contenedor.innerHTML, "");
  assert.equal(contenedor.retirado, true);
});

test("C5 consume los tokens del tema sin colores propios", async () => {
  const css = await readFile(new URL("./vista-correcciones.css", import.meta.url), "utf8");
  assert.match(css, /var\(--portal-aviso-suave/u);
  assert.match(css, /:focus-visible/u);
  assert.doesNotMatch(css, /#[0-9a-fA-F]{3,8}\b/u);
});
