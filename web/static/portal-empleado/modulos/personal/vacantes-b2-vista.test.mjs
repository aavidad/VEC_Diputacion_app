import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { crearVistaVacantesB2 } from "./vacantes-b2-vista.js";

function documento() {
  return { createElement(tipo) { return { tagName: tipo, textContent: "", children: [], dataset: {}, atributos: {}, append(...hijos) { this.children.push(...hijos); }, setAttribute(nombre, valor) { this.atributos[nombre] = valor; } }; } };
}
function nodos(n) { return [n, ...n.children.flatMap(nodos)]; }
function visible(n) { return [n.textContent, ...(n.tagName === "details" && !n.open ? n.children.slice(0, 1) : n.children).map(visible)].join(" "); }
function dto() {
  return { organismo_ref: "organismo_privado", corte: { vigente_en: "2026-10-01", conocido_en: "2026-10-01T12:00:00Z" }, cobertura: "completa", cursor_siguiente: "cursor_privado", vacantes: [{
    plaza_ref: "plaza_privada", puesto_ref: "puesto_privado", unidad_ref: "unidad_privada", unidad_denominacion: "Unidad <script>alert(1)</script>", puesto_denominacion: "Técnico/a", codigo_plaza_fuente: "00041", estado_cobertura: "vacante_sin_ocupacion", version_plantilla_ref: "version_plantilla_privada", version_rpt_ref: "version_rpt_privada",
    traza: { desde: "2024-01-01", registrada_en: "2026-09-30T11:00:00Z", revision_estructural: 3, version_plantilla_ref: "version_plantilla_privada", acto_ref: "acto_privado", fuente_ref: "fuente_privada", fuente_huella_sha256: "a".repeat(64) },
  }] };
}
function opciones(pagina = dto()) {
  const d = documento(); const llamadas = [];
  return { documento: d, pagina, formatos: { fecha: (v) => `fecha ${v}`, instante: (v) => `instante ${v}`, numero: (v) => `número ${v}` },
    anadirDatoTraza(lista, titulo, valor) { llamadas.push([titulo, valor]); const etiqueta = d.createElement("dt"); etiqueta.textContent = titulo; const dato = d.createElement("dd"); dato.textContent = valor; lista.append(etiqueta, dato); }, llamadas };
}

test("la hoja muestra los tres ejes y el corte; las referencias quedan solo tras el segundo pliegue", () => {
  const opts = opciones(); const hoja = crearVistaVacantesB2(opts);
  assert.match(visible(hoja), /Sin ocupación registrada.*Consta un puesto vinculado.*Pendiente de determinar/u);
  assert.match(visible(hoja), /fecha 2026-10-01.*instante 2026-10-01T12:00:00Z/u);
  assert.doesNotMatch(visible(hoja), /plaza_privada|puesto_privado|organismo_privado|acto_privado|fuente_privada|version_rpt_privada|cursor_privado/u);
  const traza = nodos(hoja).find((n) => n.className === "personal-registro-b2-traza" && !n.open); traza.open = true;
  assert.match(visible(traza), /Revisión de la plaza.*número 3/u);
  assert.doesNotMatch(visible(traza), /acto_privado|fuente_privada/u);
  const tecnico = nodos(traza).find((n) => n.className === "personal-registro-b2-traza-tecnica"); tecnico.open = true;
  assert.match(visible(traza), /version_plantilla_privada.*version_rpt_privada.*fuente_privada.*acto_privado/u);
  assert.equal(nodos(hoja).filter((n) => n.tagName === "a" || n.tagName === "script").length, 0);
  assert.ok(opts.llamadas.some(([titulo]) => titulo === "Fecha de registro"), "reutiliza el helper de traza inyectado");
});

test("vacío significa sin resultados de la consulta y los fallos no exponen una página anterior", () => {
  const vacia = dto(); vacia.vacantes = [];
  assert.match(visible(crearVistaVacantesB2(opciones(vacia))), /No hay resultados para el ámbito y las fechas consultadas/u);
  for (const estado of ["cargando", "error", "denegado", "cobertura_no_acreditada"]) {
    const hoja = crearVistaVacantesB2({ documento: documento(), estado, pagina: dto() });
    assert.ok(!nodos(hoja).some((n) => n.tagName === "table")); assert.doesNotMatch(visible(hoja), /00041|Técnico|plaza_privada/u);
  }
  assert.throws(() => crearVistaVacantesB2(opciones({ ...dto(), cobertura: "parcial" })), TypeError);
});

test("catálogos reales ES/EN y plurales completos; inglés también distingue establecimiento y puesto", async () => {
  const es = await cargarTextos("personal-vacantes", { idioma: "es" }); const en = await cargarTextos("personal-vacantes", { idioma: "en" });
  assert.deepEqual(Object.keys(es.seccion("general")), Object.keys(en.seccion("general")));
  const hoja = crearVistaVacantesB2({ ...opciones(), traducir: (codigoMensaje, variables) => en.traducir(`general.${codigoMensaje}`, variables) });
  assert.match(visible(hoja), /Staff establishment post.*Linked position.*Staffing need/u);
  assert.equal(en.traducir("general.recuento", { cuenta: 1 }), "1 post on this page");
  assert.equal(es.traducir("general.recuento", { cuenta: 2 }), "2 plazas en esta página");
  const codigo = await readFile(new URL("./vacantes-b2-vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigo, /innerHTML|localStorage|sessionStorage|indexedDB|document\.cookie|fetch\(/u);
});


test("una denominación larga válida sigue llegando a las celdas, sin perder registros ni cambiar cobertura", () => {
  const pagina = dto(); const nombre = "😀".repeat(300);
  pagina.vacantes[0].puesto_denominacion = nombre; pagina.vacantes[0].unidad_denominacion = "U".repeat(257);
  const hoja = crearVistaVacantesB2(opciones(pagina));
  assert.ok(nodos(hoja).some((n) => n.textContent === nombre));
  assert.ok(nodos(hoja).some((n) => n.textContent === "U".repeat(257)));
  assert.equal(nodos(hoja).filter((n) => n.dataset.personalVacante !== undefined).length, 1);
});
