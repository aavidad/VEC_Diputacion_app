import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { montarVistaNotificacionesCronos, renderizarVistaNotificacionesCronos } from "./vista-notificaciones.js";

function nodo() {
  const eventos = new Map();
  return {
    value: "", textContent: "", hidden: true, disabled: true,
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    emitir(tipo, evento = {}) { eventos.get(tipo)?.(evento); },
    focus() { this.enfocado = true; },
    get eventos() { return eventos; },
  };
}

function entorno() {
  const fecha = nodo();
  const texto = nodo();
  const contador = nodo();
  const revisar = nodo();
  const revision = nodo();
  const fechaRevisada = nodo();
  const textoRevisado = nodo();
  const tituloRevision = nodo();
  const elementos = new Map([
    ['[name="fecha"]', fecha], ['[name="texto"]', texto],
    ["[data-cronos-notificacion-contador]", contador],
    ["[data-cronos-notificacion-revisar]", revisar],
    ["[data-cronos-notificacion-revision]", revision],
    ["[data-cronos-notificacion-fecha]", fechaRevisada],
    ["[data-cronos-notificacion-texto]", textoRevisado],
    ["#cronos-notificaciones-revision-titulo", tituloRevision],
  ]);
  const contenedor = { ...nodo(), innerHTML: "", querySelector: (selector) => elementos.get(selector), remove() { this.eliminado = true; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  return { raiz, contenedor, fecha, texto, contador, revisar, revision, fechaRevisada, textoRevisado, tituloRevision };
}

test("C9 muestra preparación local y bloquea tipo, adjunto, envío y archivo", () => {
  const html = renderizarVistaNotificacionesCronos();
  assert.match(html, /data-estado-entrega="sin_enviar"/u);
  assert.match(html, /<input type="date" name="fecha" required>/u);
  assert.match(html, /<textarea name="texto"[^>]*maxlength="512"/u);
  assert.match(html, /data-cronos-notificacion-revisar disabled/u);
  assert.match(html, /Enviar a RRHH<\/button>/u);
  assert.match(html, /Envío deshabilitado: faltan tipo gobernado, autorización y servicio con recibo/u);
  assert.match(html, /Archivo deshabilitado: faltan consulta autorizada e historial de mensajes/u);
  assert.match(html, /<select disabled aria-disabled="true"/u);
  assert.match(html, /<input type="file" disabled aria-disabled="true"/u);
  assert.doesNotMatch(html, /<tbody>|DEMO-|data-recibo=/u);
});

test("revisar requiere fecha civil y texto, no envía y borra el borrador al desmontar", () => {
  const e = entorno();
  let desmontarRegistrado;
  const avisos = [];
  const vista = montarVistaNotificacionesCronos({
    raiz: e.raiz, registrarDesmontar: (fn) => { desmontarRegistrado = fn; }, anunciar: (aviso) => avisos.push(aviso),
  });
  e.fecha.value = "2026-02-29";
  e.texto.value = '<img src=x onerror="alert(1)">';
  e.fecha.emitir("input");
  assert.equal(e.revisar.disabled, true);
  e.fecha.value = "2028-02-29";
  e.fecha.emitir("input");
  assert.equal(e.revisar.disabled, false);
  assert.equal(e.contador.textContent, `${e.texto.value.length} de 512 caracteres`);
  e.revisar.emitir("click");
  assert.equal(e.revision.hidden, false);
  assert.equal(e.fechaRevisada.textContent, "29 de febrero de 2028");
  assert.equal(e.textoRevisado.textContent, e.texto.value);
  assert.equal(e.tituloRevision.enfocado, true);
  assert.match(avisos[0], /No se ha registrado ni enviado/u);
  let impedido = false;
  e.contenedor.emitir("submit", { preventDefault() { impedido = true; } });
  assert.equal(impedido, true);
  e.texto.value = "Cambio de contenido";
  e.texto.emitir("input");
  assert.equal(e.revision.hidden, true);
  assert.equal(e.textoRevisado.textContent, "");
  e.texto.value = "a".repeat(513);
  e.texto.emitir("input");
  assert.equal(e.revisar.disabled, true);
  assert.equal(e.contador.textContent, "513 de 512 caracteres");
  desmontarRegistrado();
  vista.desmontar();
  assert.equal(e.contenedor.eliminado, true);
  assert.equal(e.fecha.value, "");
  assert.equal(e.texto.value, "");
  assert.equal(e.fecha.eventos.size, 0);
  assert.equal(e.texto.eventos.size, 0);
  assert.equal(e.revisar.eventos.size, 0);
  assert.equal(e.contenedor.eventos.size, 0);
});

test("el catálogo escapa textos variables y la vista no usa red ni almacenamiento", async () => {
  const html = renderizarVistaNotificacionesCronos({ mensajes: { notificaciones_titulo: '<img src=x onerror="alert(1)">' } });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
  const fuente = await readFile(new URL("vista-notificaciones.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation/iu);
});
