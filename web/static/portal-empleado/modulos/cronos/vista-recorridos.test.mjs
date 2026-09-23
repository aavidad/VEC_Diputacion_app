import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { montarVistaRecorridosCronos, renderizarRecorridosCronos } from "./vista-recorridos.js";

const directorio = new URL("./", import.meta.url);

test("Cronos presenta una bandeja primero y conserva los tres recorridos de revisión", () => {
  const html = renderizarRecorridosCronos();
  for (const id of ["cronos-persona", "cronos-responsable", "cronos-rrhh"]) assert.match(html, new RegExp(`id="${id}"`));
  for (const texto of ["Antonio López Fernández", "María del Carmen Ruiz Moreno", "Jornada prevista hoy", "Bandeja del responsable", "Incidencias y correcciones", "Pendiente de conexión"]) assert.match(html, new RegExp(texto));
  assert.match(html, /data-estado-entrega="visual_pendiente_backend"/);
  assert.match(html, /id="cronos-responsable"[^>]+hidden/);
  assert.match(html, /id="cronos-rrhh"[^>]+hidden/);
  assert.doesNotMatch(html, /Identidad y relación Persona–Empleado verificadas por servidor/);
  assert.doesNotMatch(html, /Persistencia, auditoría, recibo e historia recuperables/);
  assert.doesNotMatch(html, /\b(?:DEMO|prueba|usuario de prueba)\b/i);
});

test("alta y detalles empiezan plegados y los efectos siguen deshabilitados", () => {
  const html = renderizarRecorridosCronos();
  assert.match(html, /data-cronos-control="persona-periodo"/);
  assert.match(html, /data-cronos-alta aria-expanded="false"/);
  assert.match(html, /data-cronos-alta-panel hidden/);
  assert.match(html, /data-cronos-detalle="cronos-solicitudes-detalle-0" aria-expanded="false"/);
  assert.match(html, /id="cronos-solicitudes-detalle-0" class="cronos-fila-detalle" hidden/);
  for (const tono of ["aviso", "exito", "peligro"]) assert.match(html, new RegExp(`cronos-estado-${tono}`));
  assert.match(html, /disabled aria-disabled="true" title="Acción pendiente de backend"/);
  for (const texto of ["Nueva solicitud", "Corrección de marcaje", "Aprobar", "Notificar", "Guardar calendario"]) assert.match(html, new RegExp(texto));
  for (const paso of ["Tipo y periodo", "Aclaración", "Revisión"]) assert.match(html, new RegExp(paso));
  assert.match(html, /data-cronos-paso="3"[^>]+hidden/u);
  assert.match(html, /Registrar solicitud<\/button>/u);
  assert.match(html, /disabled aria-disabled="true" title="Registro pendiente: faltan backend, autorización y recibo\."/u);
  assert.match(html, /Tipos, cuantías y responsables sujetos a confirmación de RRHH/u);
  assert.match(html, /<details class="cronos-permisos-ayuda"><summary>\? Ayuda para esta solicitud<\/summary>/u);
});

test("los pasos validan el periodo y actualizan el resumen local sin registrar", () => {
  const listeners = new Map();
  const fechas = { desde: { value: "", validity: { valid: true }, focus() { this.enfocado = true; } }, hasta: { value: "", validity: { valid: true } }, tipo: { value: "vacaciones" } };
  const formulario = { elements: { namedItem(nombre) { return fechas[nombre]; } } };
  const pasos = [1, 2, 3].map((numero) => ({ dataset: { cronosPaso: String(numero) }, hidden: numero !== 1 }));
  const indicadores = [1, 2, 3].map((numero) => ({ dataset: { cronosPasoIndicador: String(numero) }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; }, removeAttribute(nombre) { delete this.atributos[nombre]; } }));
  const resumen = Object.fromEntries(["tipo", "periodo", "saldo"].map((clave) => [clave, { textContent: "" }]));
  const error = { hidden: true };
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(tipo, fn) { listeners.set(tipo, fn); }, removeEventListener(tipo) { listeners.delete(tipo); }, remove() {}, querySelectorAll(selector) { if (selector === "[data-cronos-paso]") return pasos; if (selector === "[data-cronos-paso-indicador]") return indicadores; return []; }, querySelector(selector) { if (selector === "[data-cronos-permisos-formulario]") return formulario; if (selector === "[data-cronos-permisos-error]") return error; const clave = selector.match(/^\[data-cronos-resumen="(.+)"\]$/u)?.[1]; return clave ? resumen[clave] : null; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  const anuncios = [];
  const vista = montarVistaRecorridosCronos({ raiz, anunciar: (mensaje) => anuncios.push(mensaje) });
  const siguiente = { closest: (selector) => selector === "[data-cronos-paso-siguiente]" ? {} : null };
  listeners.get("click")({ target: siguiente });
  assert.equal(pasos[0].hidden, false);
  assert.equal(error.hidden, false);
  assert.match(anuncios[0], /periodo válido/u);
  fechas.desde.value = "2026-10-09"; fechas.hasta.value = "2026-10-05";
  listeners.get("click")({ target: siguiente });
  assert.equal(pasos[0].hidden, false);
  fechas.hasta.value = "2026-10-12";
  listeners.get("input")({ target: { closest: (selector) => selector === "[data-cronos-permisos-formulario]" ? formulario : null } });
  assert.equal(resumen.periodo.textContent, "09/10/2026 – 12/10/2026");
  fechas.tipo.value = "correccion";
  listeners.get("change")({ target: { closest: (selector) => selector === "[data-cronos-permisos-formulario]" ? formulario : null } });
  assert.match(resumen.saldo.textContent, /No aplica/u);
  listeners.get("click")({ target: siguiente });
  assert.equal(pasos[1].hidden, false);
  assert.equal(indicadores[0].atributos["data-estado"], "hecho");
  assert.equal(indicadores[1].atributos["aria-current"], "step");
  listeners.get("click")({ target: siguiente });
  assert.equal(pasos[2].hidden, false);
  let impedido = false;
  listeners.get("submit")({ target: { matches: () => true }, preventDefault() { impedido = true; } });
  assert.equal(impedido, true);
  vista.desmontar();
  assert.equal(listeners.size, 0);
});

test("la navegación entre roles es local y no muta el hash del portal", () => {
  const html = renderizarRecorridosCronos();
  assert.doesNotMatch(html, /href="#cronos-(?:persona|responsable|rrhh)"/u);
  assert.match(html, /data-cronos-rol="cronos-responsable" aria-controls="cronos-responsable" aria-selected="false"/u);
  const escuchar = new Map();
  const botones = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((rol) => ({ dataset: { cronosRol: rol }, atributos: {}, setAttribute(nombre, valor) { this.atributos[nombre] = valor; }, removeAttribute(nombre) { delete this.atributos[nombre]; } }));
  const etapas = ["cronos-persona", "cronos-responsable", "cronos-rrhh"].map((id) => ({ id, hidden: id !== "cronos-persona" }));
  const destino = { desplazado: false, scrollIntoView() { this.desplazado = true; } };
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(tipo, fn) { escuchar.set(tipo, fn); }, removeEventListener() {}, remove() {}, querySelectorAll(selector) { if (selector === "[data-cronos-rol]") return botones; if (selector === ".cronos-recorrido-etapa") return etapas; return []; }, querySelector(selector) { return selector === "#cronos-responsable" ? destino : null; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  montarVistaRecorridosCronos({ raiz });
  const hashPortal = "#portal-empleado/inicio";
  escuchar.get("click")({ target: { closest: (selector) => selector === "[data-cronos-rol]" ? botones[1] : null } });
  assert.equal(hashPortal, "#portal-empleado/inicio");
  assert.equal(botones[1].atributos["aria-selected"], "true");
  assert.equal(botones[0].atributos["aria-selected"], "false");
  assert.equal(etapas[0].hidden, true);
  assert.equal(etapas[1].hidden, false);
  assert.equal(destino.desplazado, true);
});

test("el detalle se abre bajo su fila y la alta sólo aparece tras la acción", () => {
  const escuchar = new Map();
  const botonDetalle = { dataset: { cronosDetalle: "cronos-solicitudes-detalle-0" }, atributos: { "aria-expanded": "false" }, getAttribute(nombre) { return this.atributos[nombre]; }, setAttribute(nombre, valor) { this.atributos[nombre] = valor; } };
  const filaDetalle = { hidden: true };
  const botonAlta = { atributos: { "aria-expanded": "false" }, getAttribute(nombre) { return this.atributos[nombre]; }, setAttribute(nombre, valor) { this.atributos[nombre] = valor; } };
  const panelAlta = { hidden: true };
  const contenedor = { dataset: {}, innerHTML: "", addEventListener(tipo, fn) { escuchar.set(tipo, fn); }, removeEventListener() {}, remove() {}, querySelectorAll(selector) { if (selector === "[data-cronos-detalle]") return [botonDetalle]; if (selector === ".cronos-fila-detalle") return [filaDetalle]; if (selector === "[data-cronos-alta]") return [botonAlta]; return []; }, querySelector(selector) { if (selector === "#cronos-solicitudes-detalle-0") return filaDetalle; if (selector === "[data-cronos-alta-panel]") return panelAlta; return null; } };
  const raiz = { ownerDocument: { createElement: () => contenedor }, append() {} };
  montarVistaRecorridosCronos({ raiz });
  escuchar.get("click")({ target: { closest: (selector) => selector === "[data-cronos-detalle]" ? botonDetalle : null } });
  assert.equal(botonDetalle.atributos["aria-expanded"], "true");
  assert.equal(filaDetalle.hidden, false);
  escuchar.get("click")({ target: { closest: (selector) => selector === "[data-cronos-alta]" ? botonAlta : null } });
  assert.equal(botonAlta.atributos["aria-expanded"], "true");
  assert.equal(panelAlta.hidden, false);
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
