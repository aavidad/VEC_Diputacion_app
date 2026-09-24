import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { montarVistaRecorridosCronos, renderizarRecorridosCronos } from "./vista-recorridos.js";

const directorio = new URL("./", import.meta.url);

test("Cronos sin servicio presenta no configurado y no proyecta personas ni saldos sintéticos", () => {
  const html = renderizarRecorridosCronos();
  assert.match(html, /data-estado-entrega="no_configurado"/);
<<<<<<< HEAD
  assert.match(html, /Cronos todavía no tiene datos de permisos conectados\./u);
=======
  assert.match(html, /Servicio de Cronos no disponible/u);
>>>>>>> b6ae6c62 (Cronos: retira presentación volátil y catálogo compilado)
  for (const id of ["cronos-persona", "cronos-responsable", "cronos-rrhh"]) assert.match(html, new RegExp(`id="${id}"`));
  assert.match(html, /id="cronos-responsable"[^>]+hidden/u);
  assert.match(html, /id="cronos-rrhh"[^>]+hidden/u);
  for (const valor of ["Antonio López", "María del Carmen", "14 días", "07:30", "05–09 oct.", "Pendiente de responsable", "Catálogo provisional"]) assert.doesNotMatch(html, new RegExp(valor, "u"));
  assert.doesNotMatch(html, /<tbody>|data-cronos-detalle=|data-cronos-control=/u);
});

test("la estructura conserva etapas y hojas sin formularios ni acciones inertes", () => {
  const html = renderizarRecorridosCronos();
  for (const titulo of ["Solicitudes", "Jornada y fichaje", "Movimientos e incidencias", "Permisos y licencias", "Bandeja de equipo", "Revisión de incidencias"]) {
    assert.match(html, new RegExp(titulo, "u"));
  }
<<<<<<< HEAD
  assert.match(html, /Registrar solicitud<\/button>/u);
  assert.match(html, /disabled aria-disabled="true" title="El registro de solicitudes todavía no está disponible\."/u);
  assert.match(html, /Catálogo de permisos no disponible\./u);
  assert.match(html, /<details class="cronos-permisos-ayuda"><summary aria-label="\? Ayuda para esta solicitud">\?<\/summary><p>El recorrido tendrá tipo y fechas/u);
  assert.doesNotMatch(html, /<details class="cronos-permisos-ayuda" open|<summary[^>]*tabindex="-1"/u);
  assert.match(html, /id="cronos-permisos-paso-2-titulo" tabindex="-1"/u);
});

test("la ayuda mantiene el control nativo y el foco sin marcador adicional", async () => {
  const html = renderizarRecorridosCronos();
  const css = await readFile(new URL("permisos.css", directorio), "utf8");
  assert.match(html, /<details class="cronos-permisos-ayuda"><summary aria-label="\? Ayuda para esta solicitud">\?<\/summary>/u);
  assert.match(css, /\.cronos-permisos-ayuda summary\s*\{[^}]*list-style:\s*none;/u);
  assert.match(css, /\.cronos-permisos-ayuda summary::-webkit-details-marker\s*\{\s*display:\s*none;\s*\}/u);
  assert.match(css, /\.cronos-permisos-ayuda summary:focus-visible\s*\{[^}]*outline:\s*3px solid var\(--portal-cian\);/u);
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
=======
  assert.doesNotMatch(html, /<form|<input|<select|<textarea|data-cronos-alta|data-cronos-paso|<details|cronos-recorrido-privacidad/u);
  assert.doesNotMatch(html, /Recorrido visual|Jornada, movimientos, calendario|Esta pantalla no solicita/u);
>>>>>>> b6ae6c62 (Cronos: retira presentación volátil y catálogo compilado)
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
  const html = renderizarRecorridosCronos({ mensajes: { ...MENSAJES_CRONOS_ES, recorridos_titulo: '<img src=x onerror="alert(1)">' } });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
});

test("las hojas del empleado se alcanzan y se desmontan al cambiar de apartado o rol", () => {
  const html = renderizarRecorridosCronos();
  for (const apartado of ["solicitudes", "catalogo", "correcciones", "notificaciones"]) {
    assert.match(html, new RegExp(`data-cronos-apartado="${apartado}"`));
    assert.match(html, new RegExp(`id="cronos-apartado-${apartado}"`));
  }
  const escuchas = new Map();
  const hijos = [];
  const crearNodo = () => {
    const control = () => ({ value: "", textContent: "", disabled: false, hidden: true, addEventListener() {}, removeEventListener() {}, focus() {} });
    const controles = Object.fromEntries([
      '[name="fecha"]', '[name="texto"]', "[data-cronos-notificacion-contador]", "[data-cronos-notificacion-revisar]",
      "[data-cronos-notificacion-revision]", "[data-cronos-notificacion-fecha]", "[data-cronos-notificacion-texto]",
      "#cronos-notificaciones-revision-titulo",
    ].map((selector) => [selector, control()]));
    return { dataset: {}, innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() { this.eliminado = true; }, querySelector(selector) { return controles[selector] || null; } };
  };
  const documento = { createElement: crearNodo };
  const destinos = Object.fromEntries(["catalogo", "correcciones", "notificaciones"].map((clave) => [clave, {
    ownerDocument: documento, append(nodo) { hijos.push({ clave, nodo }); },
  }]));
  const apartados = ["solicitudes", "catalogo", "correcciones", "notificaciones"].map((clave) => ({
    dataset: { cronosApartado: clave }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; },
  }));
  const paneles = apartados.map((apartado) => ({ id: `cronos-apartado-${apartado.dataset.cronosApartado}`, hidden: apartado.dataset.cronosApartado !== "solicitudes" }));
  const roles = ["cronos-persona", "cronos-responsable"].map((clave) => ({ dataset: { cronosRol: clave }, setAttribute() {} }));
  const etapas = roles.map((rol) => ({ id: rol.dataset.cronosRol, hidden: rol.dataset.cronosRol !== "cronos-persona" }));
  const contenedor = {
    dataset: {}, innerHTML: "", addEventListener(tipo, fn) { escuchas.set(tipo, fn); }, removeEventListener(tipo) { escuchas.delete(tipo); }, remove() {},
    querySelectorAll(selector) {
      if (selector === "[data-cronos-apartado]") return apartados;
      if (selector === "[id^='cronos-apartado-']") return paneles;
      if (selector === "[data-cronos-rol]") return roles;
      if (selector === ".cronos-recorrido-etapa") return etapas;
      return [];
    },
    querySelector(selector) { return destinos[selector.match(/^\[data-cronos-hoja="(.*)"\]$/u)?.[1]] || null; },
  };
  documento.createElement = () => { documento.createElement = crearNodo; return contenedor; };
  const vista = montarVistaRecorridosCronos({ raiz: { ownerDocument: documento, append() {} } });
  const clic = (selector, nodo) => escuchas.get("click")({ target: { closest: (buscado) => buscado === selector ? nodo : null } });
  clic("[data-cronos-apartado]", apartados[1]);
  assert.match(hijos.at(-1).nodo.innerHTML, /data-cronos-c6-estado="no_configurado"/u);
  assert.match(hijos.at(-1).nodo.innerHTML, /Catálogo de permisos no disponible\./u);
  assert.doesNotMatch(hijos.at(-1).nodo.innerHTML, /Asuntos propios|Vacaciones|tipo_asuntos_propios/u);
  assert.equal(paneles[1].hidden, false);
  clic("[data-cronos-apartado]", apartados[2]);
  assert.equal(hijos.at(-2).nodo.eliminado, true);
  assert.match(hijos.at(-1).nodo.innerHTML, /Olvido de marcaje/u);
  clic("[data-cronos-apartado]", apartados[3]);
  assert.equal(hijos.at(-2).nodo.eliminado, true);
  assert.match(hijos.at(-1).nodo.innerHTML, /Notificaciones a RRHH/u);
  clic("[data-cronos-rol]", roles[1]);
  assert.equal(hijos.at(-1).nodo.eliminado, true);
  assert.equal(paneles[0].hidden, false);
  vista.desmontar();
  assert.equal(escuchas.size, 0);
});
