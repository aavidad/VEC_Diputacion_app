import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { montarVistaRecorridosCronos, renderizarRecorridosCronos } from "./vista-recorridos.js";

const directorio = new URL("./", import.meta.url);

test("el recorrido de Cronos presenta las tres etapas y no inventa hechos", () => {
  const html = renderizarRecorridosCronos();
  for (const id of ["cronos-persona", "cronos-responsable", "cronos-rrhh"]) assert.match(html, new RegExp(`id="${id}"`));
  for (const texto of ["Saldo horario", "Solicitar corrección", "Bandeja de equipo", "Evaluación de solicitudes", "Revisión de incidencias", "Calendarios y políticas"]) assert.match(html, new RegExp(texto));
  assert.match(html, /Pendiente de conexión/);
  assert.match(html, /Sin datos: el servicio todavía no está conectado/);
  assert.match(html, /disabled aria-disabled="true"/);
  assert.match(html, /href="#cronos-persona"/);
  for (const texto of ["Semana actual", "Mes actual", "Año", "Selección", "Resumen", "Detalle", "Olvidos de marcaje", "Absentismos", "Calendario anual", "Pendientes de justificar", "Pendientes de conceder", "Consulta de mensajes", "Archivar mensajes", "Selección de notificaciones"]) assert.match(html, new RegExp(texto));
  assert.match(html, /data-cronos-control="saldo-periodo" aria-pressed="true"/);
  assert.match(html, /data-cronos-control="mensajes-vista"/);
  assert.match(html, /Movimientos e incidencias del periodo/);
  assert.match(html, /Permisos disponibles y solicitados/);
  assert.match(html, /Días u horas/);
  assert.doesNotMatch(html, /DEMO|recibo:|fichaje registrado/i);
});

test("el montaje se desmonta y registra la limpieza sin peticiones", async () => {
  const documento = { createElement: () => ({ dataset: {}, innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() { this.eliminado = true; } }) };
  const raiz = { ownerDocument: documento, actual: null, append(nodo) { this.actual = nodo; }, querySelector() { return this.actual; } };
  let registrado;
  const vista = montarVistaRecorridosCronos({ raiz, registrarDesmontar: (desmontar) => { registrado = desmontar; } });
  assert.equal(typeof vista.desmontar, "function");
  assert.equal(registrado, vista.desmontar);
  vista.desmontar();
  assert.equal(raiz.actual.eliminado, true);
});

test("los nuevos assets no introducen almacenamiento, red ni geolocalización", async () => {
  const [vista, css] = await Promise.all([readFile(new URL("vista-recorridos.js", directorio), "utf8"), readFile(new URL("cronos.css", directorio), "utf8")]);
  assert.doesNotMatch(vista, /fetch\(|XMLHttpRequest|localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation/i);
  assert.match(css, /cronos-recorrido-etapas/);
});

test("el catálogo inyectable se trata siempre como texto", () => {
  const mensajes = { ...MENSAJES_CRONOS_ES, recorridos_titulo: '<img src=x onerror="alert(1)">' };
  const html = renderizarRecorridosCronos({ mensajes });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
});
