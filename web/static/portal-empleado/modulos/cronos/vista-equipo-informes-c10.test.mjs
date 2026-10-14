import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarEquipoInformesCronos, renderizarEquipoInformesCronos } from "./vista-equipo-informes-c10.js";
import { ErrorClienteEquipoInformesCronos } from "./cliente-equipo-informes-c10.js";

const base = { equipo_ref: "equipo_1", alcance: "jefatura", desde: "2026-10-01", hasta: "2026-10-02", vista: "presencia", version_ref: "v_1", siguiente_cursor: "",
  agregado: { total: 1, indeterminado: 1, registrado: 0, incidencias: 0 },
  filas: [{ empleado_ref: "emp_secreto", empleado_etiqueta: "Ana Ruiz", fecha: "2026-10-01", estado: "indeterminado", causa: "sin_marcajes", cobertura: "complete", hechos_registrados: 0 }] };

test("muestra nombre autorizado, recuentos de página y no deduce ausencia", () => {
  const html = renderizarEquipoInformesCronos({ estado: "listo", datos: base, desde: base.desde, hasta: base.hasta });
  assert.match(html, /Ana Ruiz/u); assert.match(html, /En esta página/u); assert.match(html, /La falta de fichajes no confirma una ausencia/u);
  assert.match(html, /aria-pressed="true"/u); assert.doesNotMatch(html, /emp_secreto|equipo_1/u);
  assert.match(html, /disabled aria-describedby="cronos-equipo-exportacion-estado"/u);
  assert.match(html, /publique su política|publique su política y/u);
});

test("solo ofrece combinaciones de informe autorizadas por el servidor", () => {
  const html = renderizarEquipoInformesCronos({ estado: "listo", datos: base, exportacionesPermitidas: ["saldo.pdf"] });
  assert.match(html, /value="saldo\.pdf"/u); assert.doesNotMatch(html, /value="permisos\.csv"/u);
  assert.doesNotMatch(html, /disabled aria-describedby="cronos-equipo-exportacion-estado"/u);
});

test("catálogo inglés real cambia todos los textos y escapa etiquetas", async () => {
  const en = JSON.parse(await readFile(new URL("../../../textos/en/cronos-equipo-informes-c10.json", import.meta.url), "utf8")).equipo;
  const html = renderizarEquipoInformesCronos({ estado: "listo", datos: { ...base, filas: [{ ...base.filas[0], empleado_etiqueta: '<img src=x onerror="x()">' }] },
    mensajes: en, locale: "en-GB" });
  assert.match(html, /On this page/u); assert.match(html, /Missing clock-ins do not confirm an absence/u);
  assert.doesNotMatch(html, /<img/u); assert.match(html, /&lt;img/u);
});

test("un fallo tras confirmar conserva la misma clave al reintentar", async () => {
  const claves = [];
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; }, removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() {} };
  const raiz = { ownerDocument: { createElement: () => nodo }, append() {} };
  const vista = montarEquipoInformesCronos({ raiz, equipo: "equipo_1", exportacionesPermitidas: ["saldo.pdf", "permisos.csv"], cliente: {
    consultar: async () => base,
    exportar: async (e) => { assert.equal(`${e.tipo}.${e.formato}`, "permisos.csv"); claves.push(e.clave_operacion); throw new ErrorClienteEquipoInformesCronos("red_no_disponible"); },
  } });
  await vista.consultar({ desde: base.desde, hasta: base.hasta });
  const form = { matches: (s) => s === "[data-cronos-exportar-equipo]", elements: { namedItem: () => ({ value: "permisos.csv" }) } };
  for (let i = 0; i < 2; i++) { nodo.eventos.submit({ target: form, preventDefault() {} }); await new Promise(setImmediate); assert.match(nodo.innerHTML, /value="permisos\.csv" selected/u); }
  assert.equal(claves.length, 2); assert.equal(claves[0], claves[1]);
  vista.desmontar();
});

test("el límite de 31 días se explica antes de consultar", async () => {
  let llamadas = 0;
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; }, removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() {}, querySelector() { return { focus() {} }; } };
  const raiz = { ownerDocument: { createElement: () => nodo }, append() {} };
  const vista = montarEquipoInformesCronos({ raiz, equipo: "equipo_1", cliente: { consultar: async () => { llamadas++; return base; }, exportar: async () => {} } });
  const form = { matches: (s) => s === "[data-cronos-periodo-equipo]", elements: { namedItem: (k) => ({ value: k === "desde" ? "2026-10-01" : "2026-11-03" }) } };
  nodo.eventos.submit({ target: form, preventDefault() {} }); await Promise.resolve();
  assert.equal(llamadas, 0); assert.match(nodo.innerHTML, /El periodo no puede superar 31 días/u);
  vista.desmontar();
});
