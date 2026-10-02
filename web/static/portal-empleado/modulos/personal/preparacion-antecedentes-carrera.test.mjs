import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { renderizarPreparacionAntecedentesCarrera, validarPreparacionAntecedentesCarrera } from "./preparacion-antecedentes-carrera.js";

function documentoFalso() {
  class Nodo {
    constructor(tagName) { this.tagName = tagName; this.children = []; this.attrs = new Map(); this.textContent = ""; }
    append(...hijos) { this.children.push(...hijos); }
    setAttribute(nombre, valor) { this.attrs.set(nombre, valor); }
  }
  return { createElement: (tipo) => new Nodo(tipo) };
}
const nodos = (n) => [n, ...n.children.flatMap(nodos)];
const textoVisible = (n) => [n.textContent, ...(n.tagName === "details" && !n.open ? n.children.slice(0,1) : n.children).map(textoVisible)].join(" ");
const texto = (n) => nodos(n).map((x) => x.textContent).join(" ");
const traza = { desde: "2020-01-01", registrada_en: "2026-10-01T10:00:00Z", version: 2, acto_ref: "acto:reservado", fuente_ref: "fuente:reservada", fuente_version: 3 };
function modelo() {
  return {
    esquema: "vec.personal.preparacion-antecedentes-carrera.v1", alcance: "preparacion", empleado_ref: "emp_eeeeeeeeeeeeeeeeeeeeeeee", version: 4,
    corte: { vigente_en: "2026-10-01", conocido_en: "2026-10-01T10:00:00Z" }, pendientes: ["fuente_institucional", "cobertura_antecedentes", "politica_carrera"],
    relaciones: [{ relacion_ref: "rel_rrrrrrrrrrrrrrrrrrrrrrrr", regimen_ref: "regimen:funcionario", regimen: "Régimen de la fuente", estado: "vigente", traza: structuredClone(traza), historia: [{ estado: "vigente", regimen_ref: "regimen:funcionario", regimen: "Régimen de la fuente", traza: structuredClone(traza) }],
      pendientes: ["grupo_subgrupo", "puesto_nivel_m", "grado_personal_h"], situaciones: [{ situacion_ref: "situacion:uno", codigo_ref: "situacion:activo", estado: "vigente", traza: structuredClone(traza) }],
      servicios: [{ servicio_ref: "servicio:uno", ultima_revision_conocida: true, estado: "reconocido", periodo_desde: "2020-01-01", periodo_hasta: "2020-12-31", dias_reconocidos: 365, traza: structuredClone(traza) }],
    }],
  };
}

test("muestra preparación, pendientes y cortes calculados sin referencias personales", () => {
  const m = modelo(); m.persona_nombre = "Nombre que no debe copiarse";
  const panel = renderizarPreparacionAntecedentesCarrera({ modelo: m, documento: documentoFalso() });
  assert.match(texto(panel), /Antecedentes para Carrera/);
  assert.match(texto(panel), /Preparación pendiente de completar/);
  assert.match(texto(panel), /Falta acreditar la fuente institucional/);
  assert.match(texto(panel), /Organización y RPT/);
  assert.match(texto(panel), /reconocimiento del grado personal/);
  assert.match(texto(panel), /365/);
  assert.doesNotMatch(textoVisible(panel), /emp_|rel_|acto:|fuente:|situacion:|servicio:|Nombre que no/);
  assert.equal(nodos(panel).find((x) => x.tagName === "table").className, "tabla-datos");
  assert(nodos(panel).filter((x) => x.tagName === "th").every((x) => x.attrs.get("scope") === "col"));
  assert.equal(nodos(panel).find((x) => x.attrs.get("role") === "region").attrs.get("tabindex"), "0");
});

test("no deriva reconocimientos ni modifica los datos recibidos", () => {
  const m = modelo(); m.relaciones[0].servicios[0].estado = "declarado"; m.relaciones[0].pendientes.push("servicios_no_reconocidos");
  const antes = structuredClone(m);
  const panel = renderizarPreparacionAntecedentesCarrera({ modelo: m, documento: documentoFalso() });
  assert.match(texto(panel), /Declarado/);
  assert.match(texto(panel), /todavía no están reconocidos/);
  assert.deepEqual(m, antes);
  assert.doesNotMatch(texto(panel), /Reconocido en el registro|Total|Antigüedad/);
});

test("ficha ausente se presenta como error, con recuperación", () => {
  const panel = renderizarPreparacionAntecedentesCarrera({ modelo: undefined, documento: documentoFalso() });
  assert.match(texto(panel), /Vuelva a consultar la ficha/);
  assert.equal(nodos(panel).find((x) => x.attrs.get("role") === "alert").tagName, "p");
  assert.doesNotMatch(texto(panel), /Preparación pendiente de completar/);
});

test("cero relaciones conserva el pendiente del backend", () => {
  const m = modelo(); m.relaciones = []; m.pendientes.push("sin_relacion");
  assert.match(texto(renderizarPreparacionAntecedentesCarrera({ modelo: m, documento: documentoFalso() })), /No consta una relación de servicio/);
});

for (const [nombre, cambiar] of Object.entries({
  esquema: (m) => { m.esquema = "otro"; },
  alcance: (m) => { m.alcance = "acreditado"; },
  eficacia: (m) => { m.eficacia_administrativa = true; },
  firma: (m) => { m.firma_oficial = true; },
  corte: (m) => { m.corte.vigente_en = "2026-02-30"; },
  version: (m) => { m.version = 0; },
  pendiente: (m) => { m.pendientes.push("aprobado"); },
  limite: (m) => { m.relaciones = Array(201).fill(m.relaciones[0]); },
  dias: (m) => { m.relaciones[0].servicios[0].dias_reconocidos = -1; },
  fuente: (m) => { m.relaciones[0].traza.fuente_ref = ""; },
  estado: (m) => { m.relaciones[0].servicios[0].estado = "acreditado"; },
})) test(`rechaza DTO incompatible: ${nombre}`, () => {
  const m = modelo(); cambiar(m);
  assert.throws(() => validarPreparacionAntecedentesCarrera(m), /invalidos/);
  assert.match(texto(renderizarPreparacionAntecedentesCarrera({ modelo: m, documento: documentoFalso() })), /Vuelva a consultar/);
});

test("escapa el contenido de la fuente mediante textContent", () => {
  const m = modelo(); m.relaciones[0].regimen = "<img src=x onerror=alert(1)>";
  const panel = renderizarPreparacionAntecedentesCarrera({ modelo: m, documento: documentoFalso() });
  assert.match(texto(panel), /<img src=x onerror=alert\(1\)>/);
  assert.equal(nodos(panel).filter((x) => x.tagName === "img").length, 0);
});

test("usa ambos catálogos completos con el traductor común", async () => {
  const es = JSON.parse(await readFile(new URL("../../../textos/es/personal-antecedentes-carrera.json", import.meta.url), "utf8"));
  const en = JSON.parse(await readFile(new URL("../../../textos/en/personal-antecedentes-carrera.json", import.meta.url), "utf8"));
  for (const section of Object.keys(es)) assert.deepEqual(Object.keys(es[section]).sort(), Object.keys(en[section]).sort());
  const textos = await cargarTextos("personal-antecedentes-carrera", { idioma: "en", porDefecto: "es" });
  assert.deepEqual(textos.faltantes, []);
  const panel = renderizarPreparacionAntecedentesCarrera({ modelo: modelo(), documento: documentoFalso(), t: textos.traducir, localizacion: textos.localizacion });
  assert.match(texto(panel), /Career progression background/);
  assert.match(texto(panel), /Position and level/);
  assert.doesNotMatch(texto(panel), /Antecedentes para Carrera|Falta acreditar/);
});

test("acepta empleado y relación nominales de B2 con mayúsculas", () => {
 const m=modelo(); m.empleado_ref="emp_"+"A".repeat(24); m.relaciones[0].relacion_ref="rel_"+"R".repeat(24);
 assert.equal(validarPreparacionAntecedentesCarrera(m),m);
 assert.doesNotMatch(texto(renderizarPreparacionAntecedentesCarrera({modelo:m,documento:documentoFalso()})),/Vuelva a consultar/);
});
test("conserva acto y fuente reales en el origen plegado", () => {
 const panel=renderizarPreparacionAntecedentesCarrera({modelo:modelo(),documento:documentoFalso()});
 assert.match(texto(panel),/acto:reservado/); assert.match(texto(panel),/fuente:reservada/);
 assert.doesNotMatch(textoVisible(panel),/acto:reservado|fuente:reservada/);
});
test("distingue la versión sustituida del último estado conocido", () => {
 const m=modelo(); const antigua=structuredClone(m.relaciones[0].servicios[0]); antigua.ultima_revision_conocida=false; m.relaciones[0].servicios[0].estado="comprobado"; m.relaciones[0].servicios[0].traza.version=3;
 m.relaciones[0].servicios.unshift(antigua);
 const panel=renderizarPreparacionAntecedentesCarrera({modelo:m,documento:documentoFalso()});
 assert.match(texto(panel),/Reconocido en el registro · Versión sustituida/);
 assert.match(texto(panel),/Comprobado · Última versión conocida/);
});
