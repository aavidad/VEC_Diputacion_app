import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosPersonal } from "./i18n.js?v=20261007-pantallas-textos-final-v1";

test.before(async () => { await prepararTextosPersonal(); });
import { compararFichasB2, montarComparacionFichaB2, instanteDesdeMadrid, fechaHoraMadrid } from "./comparacion-ficha-b2.js";
import { montarRegistroB2 } from "./registro-b2.js";

const empleadoRef = "emp_aaaaaaaaaaaaaaaaaaaaaa";
const personaRef = "per_bbbbbbbbbbbbbbbbbbbbbb";
const relacionRef = "rel_cccccccccccccccccccccc";
const primero = { vigente_en: "2026-10-03", conocido_en: "2026-10-03T10:00:00.000000Z" };
const segundo = { vigente_en: "2026-09-01", conocido_en: "2026-10-03T10:00:00.000000Z" };
const traza = { desde: "2024-01-01", registrada_en: "2024-01-01T10:00:00Z", version: 1, acto_ref: "acto:registro", fuente_ref: "fuente:personal", fuente_version: 1 };
function respuesta(corte = primero) {
  return { ficha: { empleado_ref: empleadoRef, persona_ref: personaRef, organismo_ref: "org:diputacion", corte: { ...corte }, version: 2, eficacia_administrativa: false, firma_oficial: false,
    relaciones: [{ relacion_ref: relacionRef, organismo_ref: "org:diputacion", unidad_denominacion: "Servicios Generales", estado: "vigente", traza: { ...traza }, catalogo_snapshot: { modalidad: { denominacion: "Temporal", version: 1 } } }],
    ocupaciones: [], servicios: [], situaciones: [] },
    evidencia: { recibo_ref: "recibo:uno", decision_ref: "decision:uno", efecto_ref: empleadoRef, auditoria_ref: "audit:uno", consumo_huella_sha256: "a".repeat(64), consultada_en: "2026-10-03T10:01:00Z" } };
}
const opciones = { empleadoRef, cortePrimero: primero, corteSegundo: segundo };
function raizFalsa() {
  class Nodo {
    constructor(d, tag) { this.ownerDocument = d; this.tagName = tag; this.children = []; this.dataset = {}; this.listeners = new Map(); this.attrs = new Map(); this.parent = null; this.textContent = ""; this.value = ""; }
    append(...hijos) { for (const n of hijos) { n.remove(); n.parent = this; this.children.push(n); } }
    replaceChildren(...hijos) { for (const n of this.children) n.parent = null; this.children = []; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((n) => n !== this); this.parent = null; }
    setAttribute(k, v) { this.attrs.set(k, String(v)); }
    addEventListener(k, fn) { this.listeners.set(k, fn); }
    focus() { this.enfocado = true; }
  }
  const d = { createElement(tag) { return new Nodo(d, tag); } };
  return new Nodo(d, "root");
}
const nodos = (n) => [n, ...n.children.flatMap(nodos)];
const texto = (n) => nodos(n).map((x) => x.textContent).join(" ");
const visible = (n) => [n.textContent, ...(n.tagName === "details" && !n.open ? n.children.slice(0, 1) : n.children).map(visible)].join(" ");
const buscar = (n, f) => nodos(n).find(f);
const pausa = () => new Promise((r) => setImmediate(r));
function controles(raiz) {
  const form = buscar(raiz, (n) => n.className === "personal-comparacion-formulario");
  return { form, efectos: buscar(form, (n) => n.dataset.comparacionEfectos !== undefined), conocimiento: buscar(form, (n) => n.dataset.comparacionConocimiento !== undefined), boton: buscar(form, (n) => n.type === "submit"), cancelar: buscar(form, (n) => n.type === "button") };
}
async function enviar(c) { c.efectos.value = segundo.vigente_en; await c.form.listeners.get("submit")({ preventDefault() {} }); }
function montar(cliente, extra = {}) {
  const raiz = raizFalsa(); const montaje = montarComparacionFichaB2({ raiz, cliente, empleadoRef, corteBase: primero, ...extra }); return { raiz, montaje, c: controles(raiz) };
}

test("compara versiones, periodos y origen sin sumar días ni perder historia", () => {
  const a = respuesta(); const b = respuesta(segundo);
  b.ficha.relaciones.push({ ...b.ficha.relaciones[0], traza: { ...traza, version: 2, fuente_version: 3 } });
  a.ficha.servicios = [{ servicio_ref: "srv:uno", relacion_ref: relacionRef, estado: "declarado", dias_reconocidos: 1, periodo_desde: "2020-01-01", periodo_hasta: "2021-01-01", traza }];
  const result = compararFichasB2(a, b, opciones);
  assert.equal(result.diferencias.relaciones[0].estado, "distinto");
  assert.deepEqual(result.diferencias.relaciones[0].segundo.map((x) => x.traza.version), [1, 2]);
  assert.equal(result.diferencias.servicios[0].estado, "solo_primero");
  assert.equal(result.primero.cobertura, "no_acreditada");
  assert.equal(result.primero.evidencia.recibo_ref, a.evidencia.recibo_ref);
  result.primero.relaciones[0].traza.version = 99;
  assert.equal(a.ficha.relaciones[0].traza.version, 1);
});

test("orden de filas y grafía equivalente de instantes no crean diferencias", () => {
  const a = respuesta(); const b = respuesta(segundo);
  a.ficha.corte.conocido_en = "2026-10-03T10:00:00Z";
  assert.deepEqual(compararFichasB2(a, b, opciones).diferencias, { relaciones: [], servicios: [], situaciones: [] });
});

test("rechaza empleado, persona, organismo, corte o evidencia cruzados y duplicados", () => {
  for (const cambiar of [
    (x) => { x.ficha.empleado_ref = "emp_dddddddddddddddddddddd"; },
    (x) => { x.ficha.persona_ref = "per_dddddddddddddddddddddd"; },
    (x) => { x.ficha.organismo_ref = "org:otro"; },
    (x) => { x.ficha.corte.conocido_en = "2026-10-03T10:00:00.000001Z"; },
    (x) => { delete x.evidencia; },
    (x) => { x.evidencia.efecto_ref = "emp_dddddddddddddddddddddd"; },
    (x) => { x.ficha.relaciones.push(x.ficha.relaciones[0]); },
    (x) => { x.ficha.relaciones[0].estado = "inventado"; },
    (x) => { x.ficha.firma_oficial = true; },
  ]) { const b = respuesta(segundo); cambiar(b); assert.throws(() => compararFichasB2(respuesta(), b, opciones), /incompatible/); }
});

test("Madrid: conserva el día, rechaza saltos y ambigüedad de horario", () => {
  assert.equal(fechaHoraMadrid("2026-10-03T10:00:00Z"), "2026-10-03T12:00:00");
  assert.equal(instanteDesdeMadrid("2026-01-03T12:00"), "2026-01-03T11:00:00.000000Z");
  assert.equal(instanteDesdeMadrid("2026-03-29T02:30"), "");
  assert.equal(instanteDesdeMadrid("2026-10-25T02:30"), "");
  assert.equal(instanteDesdeMadrid("2026-02-30T12:00"), "");
});

test("dos consultas frescas y evidencia plegada; resultados limitados a datos devueltos", async () => {
  const llamadas = []; let inicia = 0;
  const { raiz, c } = montar({ async consultarFicha(o) { llamadas.push(o); const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn }); if (llamadas.length === 2) r.ficha.relaciones = []; return r; } }, { alIniciar() { inicia++; } });
  await enviar(c);
  assert.equal(inicia, 1); assert.equal(llamadas.length, 2);
  assert.equal(llamadas[0].empleadoRef, llamadas[1].empleadoRef);
  assert.match(visible(raiz), /Sólo en el primer corte/);
  assert.match(visible(raiz), /Un registro que no aparece puede existir/);
  assert.doesNotMatch(visible(raiz), /recibo:uno|fuente:personal|acto:registro|emp_/);
  assert.match(texto(raiz), /recibo:uno/);
});

test("denegación, error o cruce retiran comparación previa; primera denegada evita el segundo GET", async () => {
  for (const causa of [Object.assign(new Error(), { estado: 403 }), new Error(), "cruce"]) {
    let modo = "bien"; let errores = 0; let n = 0;
    const { raiz, c } = montar({ async consultarFicha(o) { n++; if (modo !== "bien" && causa !== "cruce") throw causa; const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn }); if (modo !== "bien") r.ficha.persona_ref = "per_dddddddddddddddddddddd"; return r; } }, { alError() { errores++; } });
    await enviar(c); assert.match(texto(raiz), /Ver evidencia/);
    modo = "error";
    // A consistent person replacement in both responses cannot be identified
    // without the server's authority; cross-person response pairs are rejected.
    if (causa === "cruce") {
      let i = 0; const cliente = { async consultarFicha(o) { const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn }); if (++i === 2) r.ficha.persona_ref = "per_dddddddddddddddddddddd"; return r; } };
      const otro = montar(cliente); await enviar(otro.c); assert.match(texto(otro.raiz), /retirado los datos/); continue;
    }
    await enviar(c);
    assert.equal(errores, 1); assert.equal(n, 3); assert.doesNotMatch(texto(raiz), /recibo:uno|Servicios Generales/);
    assert.match(texto(raiz), causa.estado === 403 ? /No tiene acceso/ : /no se han podido/i);
  }
});

test("cancelar o desmontar descarta respuestas tardías y no consulta el segundo corte", async () => {
  for (const desmontar of [false, true]) {
    let resolver; let llamadas = 0;
    const { raiz, c, montaje } = montar({ consultarFicha(o) { llamadas++; return new Promise((r) => { resolver = () => r(respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn })); }); } });
    const pendiente = enviar(c); await pausa();
    if (desmontar) montaje.desmontar(); else c.cancelar.listeners.get("click")();
    resolver(); await pendiente;
    assert.equal(llamadas, 1); assert.doesNotMatch(texto(raiz), /recibo:uno/);
    if (!desmontar) assert.equal(c.boton.disabled, false);
  }
});

test("el montaje RRHH borra la ficha anterior al comparar y cancela al cambiar empleado", async () => {
  const raiz = raizFalsa(); let modo = "bien"; let resolver; let llamadas = 0;
  const cliente = { async consultarFicha(o) {
    llamadas++;
    if (modo === "denegado") throw Object.assign(new Error(), { estado: 403 });
    if (modo === "tardia") return new Promise((r) => { resolver = () => r(respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn })); });
    const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn }); r.ficha.empleado_ref = o.empleadoRef; return r;
  }, listarVacantes() {} };
  const montaje = montarRegistroB2({ raiz, cliente, empleadoRef, reloj: () => new Date("2026-10-03T10:00:00Z") });
  await pausa(); assert.match(texto(raiz), /Servicios Generales/);
  let c = controles(raiz); modo = "denegado"; await enviar(c);
  assert.doesNotMatch(texto(raiz), /Servicios Generales|Temporal/); assert.match(texto(raiz), /No tiene acceso/);
  modo = "bien"; montaje.cambiarEmpleado(empleadoRef); await pausa();
  c = controles(raiz); modo = "tardia"; const p = enviar(c); await pausa();
  modo = "bien"; montaje.cambiarEmpleado("emp_dddddddddddddddddddddd"); resolver(); await p; await pausa();
  assert.doesNotMatch(texto(raiz), /Ver evidencia/); assert.equal(llamadas, 5);
  montaje.desmontar(); assert.equal(raiz.children.length, 0);
});


test("un cambio de días reconocidos muestra ambos valores sin calcular antigüedad", async () => {
  let llamada = 0;
  const { raiz, c } = montar({ async consultarFicha(o) {
    const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn });
    r.ficha.servicios = [{ servicio_ref: "srv:uno", relacion_ref: relacionRef, estado: "reconocido", dias_reconocidos: ++llamada, periodo_desde: "2020-01-01", periodo_hasta: "2021-01-01", traza }];
    return r;
  } });
  await enviar(c);
  assert.match(visible(raiz), /Días reconocidos: 1.*Días reconocidos: 2/u);
  assert.match(visible(raiz), /Datos distintos/u);
});

test("el origen permite consultar los efectos modificados sin alterar el periodo servido", async () => {
  let llamada = 0;
  const { raiz, c } = montar({ async consultarFicha(o) {
    const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn });
    llamada++;
    r.ficha.servicios = [{ servicio_ref: "srv:uno", relacion_ref: relacionRef, estado: "reconocido", dias_reconocidos: 366,
      periodo_desde: "2020-01-01", periodo_hasta: "2021-01-01",
      traza: { ...traza, version: llamada, desde: llamada === 1 ? "2024-01-01" : "2025-02-03", hasta: "2025-12-31" } }];
    return r;
  } });
  await enviar(c);
  const origenes = nodos(raiz).filter((n) => n.tagName === "details" && n.children[0]?.textContent === "Ver origen");
  assert.ok(origenes.some((n) => /Efectos de esta versión.*2025/u.test(texto(n))));
  assert.ok(origenes.some((n) => /Fin de efectos de esta versión.*2025/u.test(texto(n))));
  assert.match(visible(raiz), /2020.*2021/u);
});

test("el origen muestra las versiones de catálogo que generan una diferencia", async () => {
  let llamada = 0;
  const { raiz, c } = montar({ async consultarFicha(o) {
    const r = respuesta({ vigente_en: o.vigenteEn, conocido_en: o.conocidoEn });
    r.ficha.relaciones[0].catalogo_snapshot.modalidad.version = ++llamada;
    return r;
  } });
  await enviar(c);
  const origenes = nodos(raiz).filter((n) => n.tagName === "details" && n.children[0]?.textContent === "Ver origen");
  assert.ok(origenes.some((n) => /Modalidad: versión 1/u.test(texto(n))));
  assert.ok(origenes.some((n) => /Modalidad: versión 2/u.test(texto(n))));
});
