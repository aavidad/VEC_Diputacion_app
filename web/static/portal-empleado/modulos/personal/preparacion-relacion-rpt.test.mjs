import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { crearPanelRelacionParaRPT } from "./preparacion-relacion-rpt.js";
import { crearTextos } from "../../../comun/textos.js";

function documentoFalso() {
  return { createElement(tagName) { return { tagName, textContent: "", children: [], attrs: {}, append(...n) { this.children.push(...n); }, setAttribute(k, v) { this.attrs[k] = v; } }; } };
}
function texto(n, visible = false) {
  const hijos = visible && n.tagName === "details" ? n.children.slice(0, 1) : n.children;
  return [n.textContent, ...hijos.map((h) => texto(h, visible))].join(" ");
}
function modelo() {
  return { esquema: "vec.personal.preparacion-relacion-rpt.v1", uso: "preparacion", empleado_ref: "emp_" + "e".repeat(24), version_ficha: 7,
    corte: { vigente_en: "2026-10-01", conocido_en: "2026-10-01T12:00:00Z" }, cobertura: "no_acreditada", estado_rpt: "pendiente_fuente_rpt",
    relaciones: [{ relacion_ref: "rel_" + "a".repeat(24), estado: "finalizada", en_intervalo: false,
      traza: { desde: "2024-01-01", hasta: "2026-10-01", registrada_en: "2026-10-01T10:00:00Z", version: 2, fuente_ref: "fuente:sintetica", fuente_version: 3, acto_ref: "acto:sintetico" } }] };
}
function panel(m = modelo(), extra = {}) {
  return crearPanelRelacionParaRPT({ documento: documentoFalso(), modelo: m, relacionRef: m.relaciones[0]?.relacion_ref, ...extra });
}
test("representa estado y límite de Organización sin inferir ocupación", () => {
  const p = panel();
  assert.match(texto(p, true), /Finalizada/);
  assert.match(texto(p, true), /Ocupación y reserva pendientes/);
  assert.match(texto(p, true), /Fecha consultada dentro del periodo No/);
  assert.doesNotMatch(texto(p, true), /rel_|fuente:sintetica|acto:sintetico|vacante|cubrible|titular/);
  assert.match(texto(p), /fuente:sintetica/);
});
test("sin selección y referencia ajena nunca seleccionan otra relación", () => {
  assert.match(texto(panel(modelo(), { relacionRef: "" })), /Seleccione una relación/);
  const ajena = panel(modelo(), { relacionRef: "rel_" + "b".repeat(24) });
  assert.match(texto(ajena), /no está disponible/);
  assert.doesNotMatch(texto(ajena), /Finalizada|fuente:sintetica/);
  const m = modelo(); m.relaciones = [];
  assert.match(texto(panel(m)), /No hay relaciones/);
});
test("la vista usa en_intervalo recibido, sin reconstruir reglas temporales", () => {
  const m = modelo(); m.relaciones[0].en_intervalo = true;
  assert.match(texto(panel(m)), /Fecha consultada dentro del periodo Sí/);
});
test("rechaza modelos incompletos, duplicados y supuestas fuentes completas", () => {
  for (const mutar of [
    (m) => { m.uso = "oficial"; }, (m) => { m.cobertura = "completa"; },
    (m) => { m.relaciones.push(m.relaciones[0]); }, (m) => { delete m.relaciones[0].en_intervalo; },
    (m) => { m.relaciones[0].traza.desde = "2026-02-30"; }, (m) => { m.relaciones[0].estado = "vacante"; },
    (m) => { m.relaciones[0].traza.fuente_ref = "<img src=x>"; }, (m) => { m.relaciones[0].traza.fuente_version = 0; },
    (m) => { delete m.relaciones[0].traza.fuente_ref; }, (m) => { delete m.empleado_ref; },
  ]) { const m = modelo(); mutar(m); assert.throws(() => panel(m), /preparacion_rpt_invalida/); }
});
test("catálogos completos y renderer con traductor común en ambos idiomas", async () => {
  const es = JSON.parse(await readFile(new URL("../../../textos/es/preparacion-relacion-rpt.json", import.meta.url)));
  const en = JSON.parse(await readFile(new URL("../../../textos/en/preparacion-relacion-rpt.json", import.meta.url)));
  assert.deepEqual(Object.keys(es.general).sort(), Object.keys(en.general).sort());
  for (const catalogo of [es, en]) {
    const t = crearTextos({ modulo: "preparacion-relacion-rpt", idioma: "prueba", localizacion: "en-GB", respaldo: catalogo });
    const p = panel(modelo(), { traducir: t.traducir, localizacion: "en-GB" });
    assert.match(texto(p), new RegExp(catalogo.general.finalizada));
    assert.match(texto(p), new RegExp(catalogo.general.pendiente.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
  }
});
