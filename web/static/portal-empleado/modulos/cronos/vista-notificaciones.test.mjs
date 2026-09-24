import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarVistaNotificacionesCronos, renderizarVistaNotificacionesCronos } from "./vista-notificaciones.js";

test("C9 no crea borradores ni mensajes ficticios", () => {
  const html = renderizarVistaNotificacionesCronos();
  assert.match(html, /data-estado-entrega="no_configurado"/u);
  assert.match(html, /Enviar a RRHH<\/button>/u);
  assert.match(html, /No hay mensajes disponibles/u);
  assert.match(html, /disabled aria-disabled="true" aria-describedby="cronos-notificaciones-sin-envio"/u);
  assert.match(html, /disabled aria-disabled="true" aria-describedby="cronos-notificaciones-sin-archivo"/u);
  assert.doesNotMatch(html, /<form|<input|<textarea|<select|DEMO-|data-recibo|Borrador local/iu);
});

test("el catálogo escapa textos variables y el montaje no conserva datos", async () => {
  const html = renderizarVistaNotificacionesCronos({ mensajes: { notificaciones_titulo: '<img src=x onerror="alert(1)">' } });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
  const contenedor = { innerHTML: "", remove() { this.retirado = true; } };
  let desmontar;
  montarVistaNotificacionesCronos({
    raiz: { ownerDocument: { createElement: () => contenedor }, append() {} },
    registrarDesmontar: (fn) => { desmontar = fn; },
  });
  desmontar();
  assert.equal(contenedor.innerHTML, "");
  assert.equal(contenedor.retirado, true);
  const fuente = await readFile(new URL("vista-notificaciones.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation/iu);
});
