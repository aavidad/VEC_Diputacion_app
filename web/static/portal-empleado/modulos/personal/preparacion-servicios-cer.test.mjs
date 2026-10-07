import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearPanelPreparacionServiciosCER, crearTraductorPreparacionServiciosCER } from "./preparacion-servicios-cer.js";

const ES = JSON.parse(await readFile(new URL("../../../textos/es/personal-servicios-cer.json", import.meta.url))).general;

function documento() {
  const d = { createElement: (tagName) => ({ tagName, children: [], attrs: {}, textContent: "", append(...hijos) { this.children.push(...hijos); }, setAttribute(k, v) { this.attrs[k] = v; } }) };
  return d;
}
const nodos = (n) => [n, ...n.children.flatMap(nodos)];
const texto = (n) => nodos(n).map((v) => v.textContent).join(" ");
const visible = (n) => [n.textContent, ...(n.tagName === "details" && !n.open ? n.children.slice(0, 1) : n.children).map(visible)].join(" ");
function modelo() {
  return { estado: "preparacion_sintetica", empleado_ref: "emp_" + "a".repeat(24), corte: { vigente_en: "2026-10-01", conocido_en: "2026-10-01T12:00:00Z" }, version: 3, cobertura: "no_acreditada", eficacia_administrativa: false, firma_oficial: false,
    servicios: [{ servicio_ref: "servicio:uno", relacion_ref: "rel_uno", periodo_desde: "2020-01-01", periodo_hasta: "2021-01-01", estado: "reconocido", clase: "Servicio previo", traza: { version: 2, fuente_version: 4, acto_ref: "acto:uno", fuente_ref: "fuente:uno" }, seleccion_temporal: "incluido", solapado: true, faltantes: [] }] };
}
const pintar = (m = modelo(), relacionRef = "rel_uno") => crearPanelPreparacionServiciosCER({ documento: documento(), modelo: m, relacionRef, t: crearTraductorPreparacionServiciosCER(ES) });

test("presenta la preparación y su límite sin emitir ni calcular certificado", () => {
  const panel = pintar();
  assert.match(visible(panel), /Estos periodos son un ejemplo.*Antes de solicitar un certificado.*confirme si la relación está completa/u);
  assert.match(texto(panel), /Reconocido en el ejercicio.*Periodos coincidentes/u);
  assert.equal(nodos(panel).filter((n) => n.tagName === "button" || n.tagName === "a").length, 0);
  assert.equal(nodos(panel).filter((n) => n.tagName === "th" && n.attrs.scope === "col").length, 5);
  assert.ok(nodos(panel).some((n) => n.attrs.role === "region" && n.attrs.tabindex === "0"));
  assert.doesNotMatch(visible(panel), /acto:uno|fuente:uno|servicio:uno|rel_uno/u);
});

test("usa la selección temporal y los solapes del modelo, sin recalcularlos", () => {
  const m = modelo(); m.servicios[0].seleccion_temporal = "fuera_corte"; m.servicios[0].solapado = false;
  assert.match(texto(pintar(m)), /Fuera de las fechas de consulta/u);
  assert.doesNotMatch(texto(pintar(m)), /Periodos coincidentes/u);
  m.servicios[0].estado = "declarado"; m.servicios[0].seleccion_temporal = "pendiente"; m.servicios[0].faltantes = ["fuente"];
  assert.match(texto(pintar(m)), /Declarado.*Fechas o procedencia incompletas.*Falta indicar la procedencia/u);
  assert.doesNotMatch(texto(pintar(m)), /Reconocido en el ejercicio/u);
  m.servicios[0].seleccion_temporal = "sustituido";
  assert.match(texto(pintar(m)), /Sustituido por otra revisión/u);
});

test("seleccionar otra relación no revela sus filas y vacío no acredita cobertura", () => {
  assert.doesNotMatch(texto(pintar(modelo(), "rel_otro")), /Servicio previo/u);
  assert.match(texto(pintar(modelo(), "rel_otro")), /No hay periodos.*Si esperaba ver alguno, consulte a Personal/u);
  assert.match(texto(pintar(modelo(), "")), /Seleccione una relación/u);
});

test("rechaza afirmaciones oficiales y DTO desconocidos", () => {
  for (const cambio of [{ eficacia_administrativa: true }, { firma_oficial: true }, { cobertura: "completa" }, { estado: "certificado" }]) assert.throws(() => pintar({ ...modelo(), ...cambio }), /modelo_invalido/u);
  const m = modelo(); m.servicios[0].estado = "oficial"; assert.throws(() => pintar(m), /modelo_invalido/u);
});

test("conserva procedencia en detalle técnico y trata nombres como texto", () => {
  const m = modelo(); m.servicios[0].clase = "<img src=x onerror=alert(1)>";
  const panel = pintar(m);
  assert.ok(nodos(panel).some((n) => n.tagName === "td" && n.textContent === m.servicios[0].clase));
  assert.equal(nodos(panel).filter((n) => n.tagName === "img" || n.tagName === "script").length, 0);
  assert.match(texto(panel), /Revisión del dato.*2.*Versión de procedencia.*4.*acto:uno.*fuente:uno/u);
});

test("catálogos completos ES y EN y traducción inglesa sin rótulos castellanos", async () => {
  const es = JSON.parse(await readFile(new URL("../../../textos/es/personal-servicios-cer.json", import.meta.url))).general;
  const en = JSON.parse(await readFile(new URL("../../../textos/en/personal-servicios-cer.json", import.meta.url))).general;
  assert.deepEqual(Object.keys(en).sort(), Object.keys(es).sort());
  const panel = crearPanelPreparacionServiciosCER({ documento: documento(), modelo: modelo(), relacionRef: "rel_uno", t: crearTraductorPreparacionServiciosCER(en) });
  assert.match(texto(panel), /Services for review.*ask HR to confirm whether the record is complete.*Recognised in the exercise/u);
  assert.doesNotMatch(texto(panel), /Revisar periodos|Periodo|Preparación/u);
});
