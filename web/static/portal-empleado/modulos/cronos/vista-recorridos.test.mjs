import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { montarVistaRecorridosCronos, renderizarRecorridosCronos } from "./vista-recorridos.js";

const directorio = new URL("./", import.meta.url);

test("Cronos presenta los tres roles, datos plausibles y el límite de backend", () => {
  const html = renderizarRecorridosCronos();
  for (const id of ["cronos-persona", "cronos-responsable", "cronos-rrhh"]) assert.match(html, new RegExp(`id="${id}"`));
  for (const texto of ["Antonio López Fernández", "María del Carmen Ruiz Moreno", "Jornada prevista hoy", "Bandeja del responsable", "Incidencias y correcciones", "WCRONOS"]) assert.match(html, new RegExp(texto));
  assert.match(html, /data-estado-entrega="visual_pendiente_backend"/);
  assert.match(html, /Identidad y relación Persona–Empleado verificadas por servidor/);
  assert.match(html, /Persistencia, auditoría, recibo e historia recuperables/);
  assert.doesNotMatch(html, /\b(?:DEMO|prueba|usuario de prueba)\b/i);
});

test("solo filtros y selección de filas son interactivos; los efectos explican su límite", () => {
  const html = renderizarRecorridosCronos();
  assert.match(html, /data-cronos-control="persona-periodo"/);
  assert.match(html, /data-cronos-fila="0" tabindex="0" aria-selected="true"/);
  assert.match(html, /disabled aria-disabled="true" title="Acción pendiente de backend"/);
  for (const texto of ["Solicitar ausencia", "Aprobar", "Notificar", "Guardar calendario"]) assert.match(html, new RegExp(texto));
});

test("la navegación entre roles es local y no muta el hash del portal", () => {
  const html = renderizarRecorridosCronos();
  assert.doesNotMatch(html, /href="#cronos-(?:persona|responsable|rrhh)"/u);
  assert.match(html, /data-cronos-rol="cronos-responsable" aria-controls="cronos-responsable"/u);
  let escuchar;
  const botones = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((rol) => ({ dataset: { cronosRol: rol }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; }, removeAttribute(nombre) { delete this.atributos[nombre]; } }));
  const destino = { desplazado: false, scrollIntoView() { this.desplazado = true; } };
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(_tipo, fn) { escuchar = fn; }, removeEventListener() {}, remove() {}, querySelectorAll(selector) { return selector === "[data-cronos-rol]" ? botones : []; }, querySelector(selector) { return selector === "#cronos-responsable" ? destino : null; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  montarVistaRecorridosCronos({ raiz });
  const hashPortal = "#portal-empleado/inicio";
  escuchar({ target: { closest: (selector) => selector === "[data-cronos-rol]" ? botones[1] : null } });
  assert.equal(hashPortal, "#portal-empleado/inicio");
  assert.equal(botones[1].atributos["aria-current"], "step");
  assert.equal(botones[0].atributos["aria-current"], undefined);
  assert.equal(destino.desplazado, true);
});

test("el montaje se desmonta sin peticiones ni almacenamiento", async () => {
  const documento = { createElement: () => ({ dataset: {}, innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() { this.eliminado = true; } }) };
  const raiz = { ownerDocument: documento, actual: null, append(nodo) { this.actual = nodo; } };
  const vista = montarVistaRecorridosCronos({ raiz });
  vista.desmontar();
  assert.equal(raiz.actual.eliminado, true);
  const fuente = await readFile(new URL("vista-recorridos.js", directorio), "utf8");
  assert.doesNotMatch(fuente, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation/i);
});

test("el catálogo inyectable se escapa", () => {
  const html = renderizarRecorridosCronos({ mensajes: { ...MENSAJES_CRONOS_ES, presentacion_titulo: '<img src=x onerror="alert(1)">' } });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
});
