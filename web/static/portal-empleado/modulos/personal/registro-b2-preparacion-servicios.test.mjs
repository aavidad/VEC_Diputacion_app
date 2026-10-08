import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosPersonal } from "./i18n.js?v=20261008-alta-rpt-circular-v4";

test.before(async () => { await prepararTextosPersonal(); });
import { montarRegistroB2 } from "./registro-b2.js?v=20261008-alta-rpt-circular-v4";
import { ErrorRegistroB2 } from "./registro-b2-cliente.js";

function raizFalsa() {
  class Nodo {
    constructor(d, tagName) { Object.assign(this, { ownerDocument: d, tagName, children: [], dataset: {}, listeners: new Map(), attrs: new Map(), textContent: "", value: "" }); }
    append(...hijos) { for (const hijo of hijos) { hijo.parent = this; this.children.push(hijo); } }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((n) => n !== this); }
    setAttribute(k, v) { this.attrs.set(k, String(v)); }
    addEventListener(k, fn) { this.listeners.set(k, fn); }
    focus() {}
  }
  const d = { createElement: (tag) => new Nodo(d, tag) };
  return new Nodo(d, "root");
}
const nodos = (n) => [n, ...n.children.flatMap(nodos)];
const texto = (n) => nodos(n).map((x) => x.textContent).join(" ");
const completar = () => new Promise((r) => setImmediate(r));
const empleado = `emp_${"a".repeat(24)}`;
const relacion = `rel_${"r".repeat(24)}`;
const otraRelacion = `rel_${"s".repeat(24)}`;
function respuesta(q, incluir = true) {
  const corte = { vigente_en: q.vigenteEn, conocido_en: q.conocidoEn };
  const traza = { desde: "2024-01-01", version: 1, registrada_en: "2024-01-01T10:00:00Z", acto_ref: "acto:uno", fuente_ref: "fuente:uno", fuente_version: 1 };
  const ficha = { empleado_ref: q.empleadoRef, version: 3, corte, eficacia_administrativa: false, firma_oficial: false,
    relaciones: [relacion, otraRelacion].map((ref) => ({ relacion_ref: ref, estado: "vigente", traza })), ocupaciones: [], situaciones: [], servicios: [] };
  const resultado = { ficha };
  if (incluir) resultado.preparacion_servicios = { empleado_ref: q.empleadoRef, version: ficha.version, corte: { ...corte }, estado: "preparacion_sintetica", cobertura: "no_acreditada", eficacia_administrativa: false, firma_oficial: false,
    servicios: [relacion, otraRelacion].map((ref, i) => ({ servicio_ref: `servicio:${i}`, relacion_ref: ref, periodo_desde: "2020-01-01", periodo_hasta: "2021-01-01", estado: "reconocido", clase: `Clase sintética ${i}`, seleccion_temporal: "incluido", solapado: true, faltantes: [], traza })) };
  return resultado;
}
function montar(consultarFicha) {
  const raiz = raizFalsa();
  const montaje = montarRegistroB2({ raiz, empleadoRef: empleado, cliente: { consultarFicha, listarVacantes: () => { throw Error("consulta inesperada"); } }, reloj: () => new Date("2026-10-01T10:00:00Z") });
  return { raiz, montaje };
}
const selector = (raiz) => nodos(raiz).find((n) => n.dataset.registroB2Relacion !== undefined);
const seleccionar = (raiz, ref) => { const s = selector(raiz); s.value = ref; s.listeners.get("change")(); };
const titulo = /Servicios para revisión/;

test("la ficha muestra la preparación validada y filtra la relación sin otra consulta", async () => {
  let llamadas = 0;
  const { raiz } = montar((q) => { llamadas++; return respuesta(q); });
  await completar();
  assert.match(texto(raiz), titulo);
  assert.doesNotMatch(texto(raiz), /Clase sintética/);
  seleccionar(raiz, relacion);
  assert.match(texto(raiz), /Clase sintética 0/);
  assert.doesNotMatch(texto(raiz), /Clase sintética 1/);
  seleccionar(raiz, otraRelacion);
  assert.match(texto(raiz), /Clase sintética 1/);
  assert.doesNotMatch(texto(raiz), /Clase sintética 0/);
  assert.equal(llamadas, 1);
});

test("un backend anterior sin preparación conserva la ficha compatible", async () => {
  const { raiz } = montar((q) => respuesta(q, false));
  await completar();
  assert.match(texto(raiz), /Relaciones de servicio/);
  assert.doesNotMatch(texto(raiz), titulo);
});

test("empleado, versión, corte o modelo incompatibles retiran toda la ficha", async () => {
  const cambios = [
    (m) => { m.empleado_ref = `emp_${"b".repeat(24)}`; },
    (m) => { m.version++; },
    (m) => { m.corte.vigente_en = "2026-09-30"; },
    (m) => { m.corte.conocido_en = "2026-10-01T00:00:00Z"; },
    (m) => { m.cobertura = "completa"; },
    (m) => { m.firma_oficial = true; },
    (m) => { m.servicios = null; },
  ];
  for (const cambio of cambios) {
    const { raiz } = montar((q) => { const r = respuesta(q); cambio(r.preparacion_servicios); return r; });
    await completar();
    assert.doesNotMatch(texto(raiz), titulo);
    assert.doesNotMatch(texto(raiz), /Relaciones de servicio|Clase sintética/);
    assert.ok(nodos(raiz).some((n) => n.attrs.get("role") === "alert"));
  }
});

test("la revocación elimina datos y un selector antiguo no repinta la ficha", async () => {
  let revocado = false;
  const { raiz } = montar((q) => { if (revocado) throw new ErrorRegistroB2("acceso_denegado", 403); return respuesta(q); });
  await completar(); seleccionar(raiz, relacion);
  const antiguo = selector(raiz); revocado = true;
  const formulario = nodos(raiz).find((n) => n.tagName === "form");
  formulario.listeners.get("submit")({ preventDefault() {} });
  assert.doesNotMatch(texto(raiz), titulo);
  await completar(); antiguo.listeners.get("change")();
  assert.doesNotMatch(texto(raiz), /Clase sintética|Relaciones de servicio/);
  assert.doesNotMatch(texto(raiz), titulo);
});

test("cambiar de empleado ignora la respuesta tardía de la ficha anterior", async () => {
  let resolver;
  const { raiz, montaje } = montar((q) => q.empleadoRef === empleado ? new Promise((r) => { resolver = () => r(respuesta(q)); }) : respuesta(q, false));
  montaje.cambiarEmpleado(`emp_${"b".repeat(24)}`);
  await completar(); resolver(); await completar();
  assert.match(texto(raiz), /Relaciones de servicio/);
  assert.doesNotMatch(texto(raiz), titulo);
  montaje.desmontar(); assert.equal(raiz.children.length, 0);
});
