import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { montarVistaRecorridosCronos, renderizarRecorridosCronos } from "./vista-recorridos.js";

const directorio = new URL("./", import.meta.url);

test("Cronos sin fuente muestra no_configurado y no proyecta personas ni saldos sintéticos", () => {
  const html = renderizarRecorridosCronos();
  assert.match(html, /data-estado-entrega="no_configurado"/);
  assert.match(html, /No hay una fuente de Cronos conectada ni catálogo de permisos gobernado y versionado/u);
  for (const id of ["cronos-persona", "cronos-responsable", "cronos-rrhh"]) assert.match(html, new RegExp(`id="${id}"`));
  assert.match(html, /id="cronos-responsable"[^>]+hidden/u);
  assert.match(html, /id="cronos-rrhh"[^>]+hidden/u);
  for (const valor of ["Antonio López", "María del Carmen", "14 días", "07:30", "05–09 oct.", "Pendiente de responsable", "Catálogo provisional"]) assert.doesNotMatch(html, new RegExp(valor, "u"));
  assert.doesNotMatch(html, /<tbody>|data-cronos-detalle=|data-cronos-control=/u);
});

test("el catálogo ausente deshabilita tipo, fechas, aclaración y registro", () => {
  const html = renderizarRecorridosCronos();
  assert.match(html, /data-cronos-alta-panel hidden/u);
  assert.match(html, /<select name="tipo" disabled aria-disabled="true"[^>]*><option value="">No configurado<\/option><\/select>/u);
  for (const campo of ["desde", "hasta", "observacion", "documento_ref"]) {
    assert.match(html, new RegExp(`name="${campo}"[^>]*disabled aria-disabled="true"`, "u"));
  }
  assert.match(html, /Registrar solicitud<\/button>/u);
  assert.match(html, /disabled aria-disabled="true" title="Registro deshabilitado: faltan catálogo versionado, vínculo de empleado y servicio autorizado\."/u);
  assert.match(html, /Catálogo de permisos no disponible: tipo, cuantía, cómputo, responsable y justificante/u);
  assert.match(html, /<details class="cronos-permisos-ayuda"><summary>\? Ayuda para esta solicitud<\/summary>/u);
  assert.match(html, /id="cronos-permisos-paso-2-titulo" tabindex="-1"/u);
});

test("los tres pasos son una explicación local y el envío se impide", () => {
  const listeners = new Map();
  const pasos = [1, 2, 3].map((numero) => ({ dataset: { cronosPaso: String(numero) }, hidden: numero !== 1 }));
  const indicadores = [1, 2, 3].map((numero) => ({ dataset: { cronosPasoIndicador: String(numero) }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; }, removeAttribute(nombre) { delete this.atributos[nombre]; } }));
  const alta = { atributos: { "aria-expanded": "false" }, getAttribute(nombre) { return this.atributos[nombre]; }, setAttribute(nombre, valor) { this.atributos[nombre] = valor; } };
  const panelAlta = { hidden: true };
  const titulos = [1, 2, 3].map(() => ({ focus() { this.enfocado = true; } }));
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(tipo, fn) { listeners.set(tipo, fn); }, removeEventListener(tipo) { listeners.delete(tipo); }, remove() {}, querySelectorAll(selector) { if (selector === "[data-cronos-paso]") return pasos; if (selector === "[data-cronos-paso-indicador]") return indicadores; return []; }, querySelector(selector) { if (selector === "[data-cronos-alta-panel]") return panelAlta; const paso = selector.match(/^#cronos-permisos-paso-(\d)-titulo$/u)?.[1]; return paso ? titulos[Number(paso) - 1] : null; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  const vista = montarVistaRecorridosCronos({ raiz });
  const objetivo = (selectorEsperado, nodo = {}) => ({ target: { closest: (selector) => selector === selectorEsperado ? nodo : null } });
  listeners.get("click")(objetivo("[data-cronos-alta]", alta));
  assert.equal(panelAlta.hidden, false);
  listeners.get("click")(objetivo("[data-cronos-paso-siguiente]"));
  assert.equal(pasos[1].hidden, false);
  assert.equal(titulos[1].enfocado, true);
  assert.equal(indicadores[0].atributos["data-estado"], "hecho");
  assert.equal(indicadores[1].atributos["aria-current"], "step");
  listeners.get("click")(objetivo("[data-cronos-paso-siguiente]"));
  assert.equal(pasos[2].hidden, false);
  listeners.get("click")(objetivo("[data-cronos-paso-anterior]"));
  assert.equal(pasos[1].hidden, false);
  let impedido = false;
  listeners.get("submit")({ target: { matches: () => true }, preventDefault() { impedido = true; } });
  assert.equal(impedido, true);
  vista.desmontar();
  assert.equal(listeners.size, 0);
});

test("la navegación de roles es local y no concede permisos", () => {
  const html = renderizarRecorridosCronos();
  assert.doesNotMatch(html, /href="#cronos-(?:persona|responsable|rrhh)"/u);
  const listeners = new Map();
  const botones = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((rol) => ({ dataset: { cronosRol: rol }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; } }));
  const etapas = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((id) => ({ id, hidden: id !== "cronos-persona" }));
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(tipo, fn) { listeners.set(tipo, fn); }, removeEventListener() {}, remove() {}, querySelectorAll(selector) { if (selector === "[data-cronos-rol]") return botones; if (selector === ".cronos-recorrido-etapa") return etapas; return []; }, querySelector() { return null; } };
  montarVistaRecorridosCronos({ raiz: { ownerDocument: { createElement: () => contenedor }, append() {} } });
  listeners.get("click")({ target: { closest: (selector) => selector === "[data-cronos-rol]" ? botones[1] : null } });
  assert.equal(botones[1].atributos["aria-selected"], "true");
  assert.equal(botones[1].atributos.tabindex, "0");
  assert.equal(botones[0].atributos.tabindex, "-1");
  assert.equal(etapas[0].hidden, true);
  assert.equal(etapas[1].hidden, false);
  assert.match(contenedor.innerHTML, /data-estado-entrega="no_configurado"/u);
});

test("las flechas, Inicio y Fin activan pestañas y mueven el foco sin cambiar la ruta", () => {
  const listeners = new Map();
  const botones = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((rol) => ({ dataset: { cronosRol: rol }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; }, focus() { this.enfocado = true; } }));
  const etapas = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((id) => ({ id, hidden: id !== "cronos-persona" }));
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(tipo, fn) { listeners.set(tipo, fn); }, removeEventListener() {}, remove() {}, querySelectorAll(selector) { if (selector === "[data-cronos-rol]") return botones; if (selector === ".cronos-recorrido-etapa") return etapas; return []; }, querySelector() { return null; } };
  montarVistaRecorridosCronos({ raiz: { ownerDocument: { createElement: () => contenedor }, append() {} } });
  let impedidos = 0;
  const pulsar = (indice, key) => listeners.get("keydown")({ key, target: { closest: (selector) => selector === "[data-cronos-rol]" ? botones[indice] : null }, preventDefault() { impedidos++; } });
  pulsar(0, "ArrowLeft");
  assert.equal(botones[2].enfocado, true);
  assert.equal(etapas[2].hidden, false);
  pulsar(2, "Home");
  assert.equal(botones[0].enfocado, true);
  pulsar(0, "End");
  assert.equal(botones[2].atributos["aria-selected"], "true");
  pulsar(2, "ArrowDown");
  assert.equal(botones[0].atributos["aria-selected"], "true");
  pulsar(2, "Tab");
  assert.equal(impedidos, 4);
});

test("el montaje no accede a red, ubicación ni almacenamiento", async () => {
  const contenedor = { dataset: {}, innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() { this.eliminado = true; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  const vista = montarVistaRecorridosCronos({ raiz });
  vista.desmontar();
  assert.equal(contenedor.eliminado, true);
  const fuente = await readFile(new URL("vista-recorridos.js", directorio), "utf8");
  assert.doesNotMatch(fuente, /datos-sinteticos|obtenerAtlasSintetico|fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation/i);
});

test("los textos inyectables se escapan", () => {
  const html = renderizarRecorridosCronos({ mensajes: { ...MENSAJES_CRONOS_ES, presentacion_titulo: '<img src=x onerror="alert(1)">' } });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
});
