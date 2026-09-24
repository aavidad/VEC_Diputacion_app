import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarCatalogoPermisosCronos, renderizarCatalogoPermisosCronos } from "./vista-catalogo-permisos.js";

test("C6 no compila tipos ni cuantías y conserva tabla vacía", () => {
  const html = renderizarCatalogoPermisosCronos();
  assert.match(html, /data-cronos-c6-estado="no_configurado"/u);
  assert.match(html, /<table>/u);
  assert.match(html, /Catálogo de permisos no disponible/u);
  assert.match(html, /Solicitud deshabilitada: catálogo y servicio no disponibles/u);
  assert.match(html, /disabled aria-disabled="true" aria-describedby="cronos-c6-sin-solicitud"/u);
  assert.doesNotMatch(html, /Asuntos propios|Vacaciones|<form|<input|data-cronos-c6-tipo/u);
});

test("el título se escapa y el montaje desmonta sin almacenamiento", () => {
  const html = renderizarCatalogoPermisosCronos({ mensajes: { titulo: '<img src=x onerror="alert(1)">' } });
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
  assert.doesNotMatch(html, /<img/u);
  const contenedor = { innerHTML: "", remove() { this.retirado = true; } };
  let desmontar;
  montarCatalogoPermisosCronos({
    raiz: { ownerDocument: { createElement: () => contenedor }, append() {} },
    registrarDesmontar: (fn) => { desmontar = fn; },
  });
  desmontar();
  assert.equal(contenedor.retirado, true);
});

test("la fuente y el tema C6 no inventan lista local", async () => {
  const fuente = await readFile(new URL("./vista-catalogo-permisos.js", import.meta.url), "utf8");
  const i18n = await readFile(new URL("./i18n-c6.js", import.meta.url), "utf8");
  const css = await readFile(new URL("./vista-catalogo-permisos.css", import.meta.url), "utf8");
  assert.doesNotMatch(fuente + i18n, /tipo_asuntos_propios|TIPOS =|fetch\(|localStorage/u);
  assert.match(css, /var\(--portal-superficie/u);
  assert.doesNotMatch(css, /#[0-9a-fA-F]{3,8}\b/u);
});
