import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearTraductorSeleccionComunicaciones, MENSAJES_SELECCION_COMUNICACIONES_ES } from "./i18n.js";
import { renderizarVistaSeleccionComunicaciones } from "./vista.js";

const comunicacion = {
  referencia: "com:01", asunto: "Convocatoria de prueba", canal: "correo", destinatariosResumen: "3 destinatarios autorizados",
  preparacion: { estado: "preparada", fecha: "2026-09-24T10:00:00Z", referencia: "prep:01" },
  revision: { estado: "revisada", fecha: "2026-09-24T11:00:00Z", referencia: "rev:01" },
  transporte: { estado: "aceptado", fecha: "2026-09-24T12:00:00Z", referencia: "smtp:01" },
};

test("sin fuente presenta vacío honesto y envío deshabilitado", () => {
  const html = renderizarVistaSeleccionComunicaciones();
  assert.match(html, /Sin fuente de comunicaciones de selección conectada/u);
  assert.match(html, /No se han consultado destinatarios ni se ha generado ningún envío/u);
  assert.match(html, /<button[^>]+disabled[^>]*>Enviar<\/button>/u);
  assert.match(html, /No hay canal corporativo configurado/u);
  assert.doesNotMatch(html, /recibo:[0-9a-f-]+/iu);
});

test("transporte aceptado no se presenta como entrega ni lectura", () => {
  const html = renderizarVistaSeleccionComunicaciones({ estadoFuente: "disponible", comunicaciones: [comunicacion], canalCorporativoConfigurado: true });
  assert.match(html, /Aceptado por transporte/u);
  assert.match(html, /smtp:01/u);
  assert.match(html, /Entrega[\s\S]*Sin constancia/u);
  assert.match(html, /Lectura[\s\S]*Sin constancia/u);
  assert.match(html, /Falta el conector de envío autorizado/u);
  assert.match(html, /disabled aria-disabled="true"/u);
});

test("denegación oculta toda proyección y la salida escapa datos de fuente", () => {
  const ataque = { ...comunicacion, asunto: '<img src=x onerror="alert(1)">' };
  const denegado = renderizarVistaSeleccionComunicaciones({ estadoFuente: "denegado", comunicaciones: [ataque] });
  assert.doesNotMatch(denegado, /img src/u);
  assert.match(denegado, /no está autorizada/u);
  const visible = renderizarVistaSeleccionComunicaciones({ estadoFuente: "disponible", comunicaciones: [ataque] });
  assert.match(visible, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
  assert.doesNotMatch(visible, /<img src=/u);
});

test("i18n cerrado y CSS responsive sin almacenamiento o red en la vista", async () => {
  const t = crearTraductorSeleccionComunicaciones();
  assert.equal(t("titulo"), MENSAJES_SELECCION_COMUNICACIONES_ES.titulo);
  assert.throws(() => t("inexistente"), /desconocida/u);
  assert.throws(() => crearTraductorSeleccionComunicaciones({ ...MENSAJES_SELECCION_COMUNICACIONES_ES, titulo: "" }), /incompleto/u);
  const [js, css] = await Promise.all([readFile(new URL("./vista.js", import.meta.url), "utf8"), readFile(new URL("./comunicaciones.css", import.meta.url), "utf8")]);
  assert.doesNotMatch(js, /fetch\(|localStorage|sessionStorage|document\.cookie/u);
  assert.match(css, /@media \(max-width: 820px\)/u);
  assert.match(css, /@media \(max-width: 600px\)/u);
  assert.match(css, /@media \(forced-colors: active\)/u);
});

test("estados sin datos no filtran registros y el montaje libera escuchas", async () => {
  for (const estadoFuente of ["cargando", "vacio", "error", "denegado", "no_configurado"]) {
    const html = renderizarVistaSeleccionComunicaciones({ estadoFuente, comunicaciones: [comunicacion] });
    assert.doesNotMatch(html, /Convocatoria de prueba/u);
  }
  const { montarVistaSeleccionComunicaciones } = await import("./vista.js");
  const listeners = new Map();
  const raiz = {
    innerHTML: "", replaceChildren() { this.innerHTML = ""; },
    addEventListener(tipo, escucha) { listeners.set(tipo, escucha); },
    removeEventListener(tipo, escucha) { if (listeners.get(tipo) === escucha) listeners.delete(tipo); },
  };
  const vista = montarVistaSeleccionComunicaciones({ raiz });
  assert.equal(listeners.size, 2);
  vista.actualizarModelo({ estadoFuente: "disponible", comunicaciones: [comunicacion] });
  assert.match(raiz.innerHTML, /Convocatoria de prueba/u);
  vista.desmontar();
  assert.equal(listeners.size, 0);
  assert.equal(raiz.innerHTML, "");
});

test("bandeja distingue transporte, entrega y lectura sin ocultar falta de evidencia", () => {
  const html = renderizarVistaSeleccionComunicaciones({ estadoFuente: "disponible", comunicaciones: [comunicacion] });
  const fila = html.match(/<tr data-seleccionada="true">([\s\S]*?)<\/tr>/u)?.[1];
  assert.ok(fila);
  assert.match(html, /<th scope="col">Transporte, entrega y lectura<\/th>/u);
  assert.match(fila, /<dt>Transporte<\/dt>[\s\S]*<dt>Entrega<\/dt>[\s\S]*<dt>Lectura<\/dt>/u);
  assert.match(fila, /Aceptado por transporte/u);
  assert.equal((fila.match(/Sin constancia/gu) || []).length, 2);
  assert.match(fila, /aria-current="true"/u);
  assert.match(html, /1 visible/u);
});

test("filtro sin coincidencias no conserva un detalle ajeno al resultado", () => {
  const html = renderizarVistaSeleccionComunicaciones({ estadoFuente: "disponible", comunicaciones: [comunicacion] }, { filtro: "otro asunto", canal: "todas", seleccion: "com:01" });
  assert.match(html, /0 visibles/u);
  assert.match(html, /Ninguna comunicación coincide con el filtro/u);
  assert.match(html, /Seleccione una comunicación de la lista para ver sus hitos/u);
  assert.doesNotMatch(html, /smtp:01/u);
});

test("S6 carga el catálogo i18n con la versión F2 de caché", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.match(fuente, /^import \{ crearTraductorSeleccionComunicaciones \} from "\.\/i18n\.js\?v=20260924-f2-web2";/mu);
  const modulo = await import("./vista.js?v=20260924-f2-web2");
  assert.match(modulo.renderizarVistaSeleccionComunicaciones(), /Sin fuente de comunicaciones de selección conectada/u);
});
