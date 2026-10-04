import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { montarVistaHistoriaRelacionesPropia } from "./vista-historia-relaciones-propia.js";

function documento() {
  class Nodo {
    constructor(d, tipo = "div") { this.ownerDocument = d; this.tagName = tipo; this.children = []; this.dataset = {}; this.attributes = new Map(); this.listeners = new Map(); this.textContent = ""; }
    append(...hijos) { this.children.push(...hijos); for (const hijo of hijos) hijo.parent = this; }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    setAttribute(nombre, valor) { this.attributes.set(nombre, valor); }
    addEventListener(tipo, fn) { this.listeners.set(tipo, fn); }
    focus() { this.ownerDocument.activeElement = this; }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((h) => h !== this); }
  }
  const d = { createElement: (tipo) => new Nodo(d, tipo), activeElement: null, hasFocus: () => true };
  const raiz = new Nodo(d); d.body = raiz; return raiz;
}
const nodos = (n) => [n, ...n.children.flatMap(nodos)];
const texto = (n) => nodos(n).map((x) => x.textContent).join(" ");
const buscar = (raiz, clave) => nodos(raiz).find((n) => Object.hasOwn(n.dataset, clave));
const filtros = { efectosDesde: "2024-01-01", efectosHasta: "2025-01-01" };
function datos() {
  const revision = (version) => ({ relacion_ref: "rel_CCCCCCCCCCCCCCCCCCCCCC", estado: "vigente", regimen: "Funcionarial", modalidad: "Carrera", unidad: "Servicio de Gestión", puesto: "Administrativo", situacion: "<dato fuente>", traza: { desde: "2024-01-01", registrada_en: "2024-03-01T09:00:00.000000Z", version, acto_ref: "acto:uno", fuente_ref: "fuente:uno", fuente_version: 1 } });
  return { historia: { corte: { efectos_desde: filtros.efectosDesde, efectos_hasta: filtros.efectosHasta, conocido_en: "2026-10-04T08:00:00.000000Z" }, cobertura: "parcial", revisiones: [revision(2), revision(1)] }, consultada_en: "2026-10-04T08:00:01.000000Z", recibo_ref: "aud_v3_abcdef0123456789abcdef0123456789" };
}
const completar = () => new Promise((r) => setImmediate(r));
const enviar = (raiz) => nodos(raiz).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
function montar(cliente) {
  const raiz = documento(); const montaje = montarVistaHistoriaRelacionesPropia({ raiz, cliente, ...filtros }); return { raiz, montaje };
}

test("sin cliente nominal no activa consulta ni inventa registros; al montar no hay red", () => {
  const { raiz } = montar(); assert.equal(buscar(raiz, "personalHistoriaConsultar").disabled, true);
  assert.match(texto(raiz), /aún no está disponible/u); assert.equal(nodos(raiz).filter((n) => n.tagName === "table").length, 0);
  let consultas = 0; montar({ consultar() { consultas++; } }); assert.equal(consultas, 0);
});

test("muestra revisiones anteriores y separa efectos, conocimiento y referencia propia", async () => {
  const llamadas = []; const { raiz } = montar({ async consultar(entrada) { llamadas.push(entrada); return datos(); } });
  const boton = buscar(raiz, "personalHistoriaConsultar"); boton.focus(); await enviar(raiz);
  assert.equal(llamadas.length, 1); assert.deepEqual(Object.keys(llamadas[0]), ["efectosDesde", "efectosHasta", "signal"]);
  const filas = nodos(raiz).filter((n) => n.tagName === "tbody")[0].children; assert.equal(filas.length, 2);
  assert.match(texto(raiz), /Información conocida hasta/u); assert.match(texto(raiz), /Consulta realizada/u);
  assert.match(texto(raiz), /Cobertura parcial/u); assert.match(texto(raiz), /no acreditan firma ni eficacia/u);
  assert.ok(nodos(raiz).some((n) => n.textContent === "<dato fuente>")); assert.equal(nodos(raiz).some((n) => n.innerHTML), false);
  assert.equal(raiz.ownerDocument.activeElement, buscar(raiz, "personalHistoriaResultado"));
  assert.ok(nodos(raiz).some((n) => n.className === "tabla-contenedor personal-ficha-tabla" && n.attributes.get("tabindex") === "0"));
});

test("error, denegación, sesión finalizada y exceso retiran filas y recibo anteriores sin otra consulta", async () => {
  for (const codigo of ["denegado", "sesion_caducada", "no_disponible", "excede_limite"]) {
    let consultas = 0; const { raiz } = montar({ async consultar() { if (++consultas === 1) return datos(); throw { codigo }; } });
    await enviar(raiz); assert.equal(nodos(raiz).some((n) => n.tagName === "table"), true);
    await enviar(raiz); assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    assert.doesNotMatch(texto(raiz), /aud_v3_/u); assert.equal(consultas, 2);
    assert.ok(nodos(raiz).some((n) => n.attributes.get("role") === "alert"));
    if (codigo === "sesion_caducada") assert.match(texto(raiz), /Identifíquese de nuevo/u);
  }
});

test("cancelar y desmontar ignoran respuestas tardías aunque el proveedor ignore abort", async () => {
  for (const accion of ["cancelar", "desmontar"]) {
    let resolver, señal; const { raiz, montaje } = montar({ consultar(entrada) { señal = entrada.signal; return new Promise((r) => { resolver = r; }); } });
    const consulta = enviar(raiz);
    if (accion === "cancelar") buscar(raiz, "personalHistoriaCancelar").listeners.get("click")(); else montaje.desmontar();
    assert.equal(señal.aborted, true); resolver(datos()); await consulta; await completar();
    assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    if (accion === "cancelar") assert.match(texto(raiz), /Consulta cancelada/u);
    else assert.equal(raiz.children.length, 0);
  }
});

test("fechas iguales no consultan y señalan el campo con foco; vacío no cambia cobertura", async () => {
  let llamadas = 0; const { raiz } = montar({ async consultar() { llamadas++; const r = datos(); r.historia.revisiones = []; r.historia.cobertura = "no_acreditada"; return r; } });
  const campos = nodos(raiz).filter((n) => n.tagName === "input"); campos[1].value = campos[0].value;
  await enviar(raiz); assert.equal(llamadas, 0); assert.equal(raiz.ownerDocument.activeElement, campos[0]);
  assert.equal(campos[0].attributes.get("aria-invalid"), "true"); campos[1].value = filtros.efectosHasta;
  await enviar(raiz); assert.equal(llamadas, 1); assert.match(texto(raiz), /Cobertura no acreditada/u); assert.match(texto(raiz), /No hay revisiones/u);
});

test("catálogos ES/EN completos: mismos campos, estados y mensajes de recuperación", () => {
  const cargar = (idioma) => JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal-historia-relaciones.json`, import.meta.url), "utf8"));
  const claves = (valor) => Object.entries(valor).flatMap(([k, v]) => typeof v === "string" ? [k] : claves(v).map((c) => `${k}.${c}`)).sort();
  const es = cargar("es"), en = cargar("en"); assert.deepEqual(claves(es), claves(en));
  assert.equal(es.general.hasta, "Efectos hasta (no incluido)"); assert.match(en.general.sesion_caducada, /Sign in again/u);
});
