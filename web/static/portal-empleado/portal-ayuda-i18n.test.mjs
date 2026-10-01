import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { AYUDA_PORTAL_BOLSA, AYUDA_CONTRATACION_TEMPORAL, TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js?v=20261001-ct-a-i18n-v1";
import { MENSAJES_AYUDANTE_TRAMITES_ES, crearAyudanteTramites } from "./ayudante-tramites.js?v=20261001-ct-a-i18n-v1";
import { MENSAJES_PORTAL, crearTraductorPortal, traducirPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { exigirRenovado } from "./versiones-cache.test-helper.mjs";

const MENSAJES_AYUDA_PORTAL = JSON.parse(await readFile(new URL("../textos/es/portal-ayuda.json", import.meta.url), "utf8")).ayuda;

test("todo texto de la ayuda y del ayudante procede del catálogo común", () => {
  const valores = new Set(Object.values(MENSAJES_AYUDA_PORTAL));
  const comprobar = (valor) => assert.ok(valores.has(valor), `fuera del catálogo: ${valor}`);
  [AYUDA_PORTAL_BOLSA.titulo, AYUDA_PORTAL_BOLSA.introduccion, AYUDA_PORTAL_BOLSA.transcripcion,
    ...AYUDA_PORTAL_BOLSA.pasos, ...AYUDA_PORTAL_BOLSA.preguntas.flatMap(({ pregunta, respuesta }) => [pregunta, respuesta])]
    .forEach(comprobar);
  for (const ayuda of [...Object.values(AYUDA_CONTRATACION_TEMPORAL.vistas), ...Object.values(AYUDA_CONTRATACION_TEMPORAL.fases)]) {
    comprobar(ayuda.titulo);
    ayuda.frases.forEach(comprobar);
  }
  for (const tramite of TRAMITES_AYUDANTE_PORTAL) {
    [tramite.titulo, tramite.modulo, tramite.resumen].forEach(comprobar);
    for (const paso of tramite.pasos) {
      [paso.titulo, paso.instruccion, paso.objetivo, paso.preparacion, paso.resultado, paso.actor, paso.limite].forEach(comprobar);
    }
  }
  Object.values(MENSAJES_AYUDANTE_TRAMITES_ES).forEach(comprobar);
  const traducirPersonalizado = crearTraductorPortal({ ...MENSAJES_PORTAL, ayuda_pasos: "Etapas" });
  assert.equal(traducirPersonalizado("ayuda_pasos"), "Etapas");
  assert.equal(traducirPortal("ayuda_abrir_contextual", { contexto: "Dietas" }), "Abrir ayuda de Dietas");
});

test("el botón ? abre la ayuda contextual sin cargar una grabación obsoleta", async () => {
  const [html, portal, ayuda, ayudante, i18n] = await Promise.all([
    "index.html", "portal.js", "ayuda-contenido.js", "ayudante-tramites.js", "portal-i18n.js",
  ].map((nombre) => readFile(new URL(nombre, import.meta.url), "utf8")));
  assert.match(html, /data-accion="ayuda"[^>]+data-i18n-portal-aria-label="ayuda_abrir_inicial"[^>]+aria-haspopup="dialog"/u);
  assert.doesNotMatch(html, /data-accion="ayuda"[^>]*>[\s\S]*?<span>Ayuda<\/span>/u);
  assert.match(portal, /ayuda_abrir_contextual/u);
  assert.match(portal, /const enfocarAyuda = \(\{ contenedor \}\)/u);
  assert.doesNotMatch(portal, /<audio controls/u);
  assert.doesNotMatch(ayuda, /ayuda-llamamiento-bolsa\.mp3/u);
  assert.match(ayudante, /\[data-ayudante-detalle\]/u);
  assert.match(i18n, /"portal-ayuda"/u);
  assert.ok(crearAyudanteTramites().contenido.includes("data-ayudante-tramite"));
});

test("la cadena de módulos renueva caché hasta el HTML", async () => {
  const [html, portal, ayuda, ayudante, panel] = await Promise.all([
    "index.html", "portal.js", "ayuda-contenido.js", "ayudante-tramites.js", "portal-panel-interno.js",
  ].map((nombre) => readFile(new URL(nombre, import.meta.url), "utf8")));
  // Versión publicada de la cadena antes de su último cambio: cada eslabón
  // cambiado pide una URL nueva, única entre todos sus importadores.
  const anterior = "20260924-rescate-web-v4";
  exigirRenovado(html, "/portal-empleado/portal.js", anterior);
  exigirRenovado(portal, "./ayudante-tramites.js", anterior);
  exigirRenovado([portal, ayudante], "./ayuda-contenido.js", anterior);
  exigirRenovado([portal, ayuda, ayudante], "./portal-i18n.js", anterior);
  // Los textos de la ayuda son datos (`textos/<idioma>/portal-ayuda.json`,
  // servidos sin caché); el aviso del panel sigue siendo un módulo versionado.
  exigirRenovado(panel, "./portal-panel-interno-i18n.js", "20260924-ayuda-panel-i18n-v4");
});
