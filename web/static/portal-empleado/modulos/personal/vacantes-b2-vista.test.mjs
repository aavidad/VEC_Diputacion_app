import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { crearVistaVacantesB2 } from "./vacantes-b2-vista.js";

function documento() {
  return { createElement(tipo) { return { tagName: tipo, textContent: "", children: [], dataset: {}, atributos: {}, eventos: {}, value: "", addEventListener(tipo, accion) { this.eventos[tipo] = accion; }, replaceChildren(...hijos) { this.children = hijos; }, focus() { this.enfocado = true; }, append(...hijos) { this.children.push(...hijos); }, setAttribute(nombre, valor) { this.atributos[nombre] = valor; } }; } };
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

function escribir(hoja, consulta) {
  const campo = nodos(hoja).find((n) => n.tagName === "input" && n.type === "search");
  campo.value = consulta; campo.eventos.input(); return campo;
}
function filas(hoja) { return nodos(hoja).filter((n) => n.dataset.personalVacante !== undefined); }

test("búsqueda literal por campos visibles de esta página, sin interpretar expresión ni referencias privadas", () => {
  const pagina = dto(); pagina.vacantes.push({ ...pagina.vacantes[0], plaza_ref: "plaza_segunda", codigo_plaza_fuente: "00902", unidad_denominacion: "Archivo Provincial", puesto_denominacion: "Administrativo/a" });
  const entrada = JSON.stringify(pagina); const hoja = crearVistaVacantesB2(opciones(pagina));
  const campo = escribir(hoja, "  00902  ");
  assert.equal(filas(hoja).length, 1); assert.match(visible(hoja), /00902/u);
  assert.match(visible(hoja), /1 coincidencia de 2 plazas en esta página/u);
  escribir(hoja, "archivo"); assert.equal(filas(hoja).length, 1);
  escribir(hoja, "ADMINISTRATIVO"); assert.equal(filas(hoja).length, 1);
  escribir(hoja, "te\u0301cnico"); assert.equal(filas(hoja).length, 1, "normaliza Unicode sin cambiar la fuente");
  escribir(hoja, "<script>"); assert.equal(filas(hoja).length, 1); assert.ok(!nodos(hoja).some((n) => n.tagName === "script"));
  for (const consulta of [".*", "plaza_privada", "acto_privado"]) {
    escribir(hoja, consulta); assert.equal(filas(hoja).length, 0);
    assert.match(visible(hoja), /Ninguna plaza de esta página coincide/u);
    assert.doesNotMatch(visible(hoja), /No hay resultados para el ámbito/u);
  }
  const limpiar = nodos(hoja).find((n) => n.tagName === "button");
  limpiar.eventos.click(); assert.equal(campo.value, ""); assert.ok(campo.enfocado); assert.ok(limpiar.disabled);
  assert.equal(filas(hoja).length, 2); assert.equal(JSON.stringify(pagina), entrada);
  assert.match(visible(hoja), /fecha 2026-10-01.*instante 2026-10-01T12:00:00Z/u);
  assert.ok(nodos(hoja).some((n) => n.atributos["aria-live"] === "polite"));
  assert.ok(nodos(hoja).some((n) => n.tagName === "label" && n.children.includes(campo)), "etiqueta visible asociada");
});

test("una página o corte nuevo arranca sin búsqueda; vacío y denegación no reutilizan resultados", () => {
  const hoja = crearVistaVacantesB2(opciones()); escribir(hoja, "no existe");
  const siguiente = dto(); siguiente.corte.vigente_en = "2026-11-01"; siguiente.cursor_siguiente = "";
  const nueva = crearVistaVacantesB2(opciones(siguiente));
  assert.equal(nodos(nueva).find((n) => n.type === "search").value, ""); assert.equal(filas(nueva).length, 1);
  assert.match(visible(nueva), /fecha 2026-11-01/u);
  for (const estado of ["cargando", "denegado", "error", "cobertura_no_acreditada"]) {
    const fallo = crearVistaVacantesB2({ documento: documento(), estado, pagina: siguiente });
    assert.equal(filas(fallo).length, 0); assert.ok(!nodos(fallo).some((n) => n.type === "search"));
  }
  const vacia = dto(); vacia.vacantes = [];
  const sinRegistros = crearVistaVacantesB2(opciones(vacia));
  assert.ok(nodos(sinRegistros).find((n) => n.type === "search").disabled);
  assert.match(visible(sinRegistros), /No hay resultados para el ámbito/u);
  assert.doesNotMatch(visible(sinRegistros), /Ninguna plaza de esta página coincide/u);
});

test("búsqueda y recuento traducidos al inglés conservan el número original de fila y el origen", async () => {
  const en = await cargarTextos("personal-vacantes", { idioma: "en" });
  const pagina = dto(); pagina.vacantes.push({ ...pagina.vacantes[0], plaza_ref: "plaza_segunda", codigo_plaza_fuente: "00902", unidad_denominacion: "Archivo" });
  const hoja = crearVistaVacantesB2({ ...opciones(pagina), traducir: (clave, variables) => en.traducir(`general.${clave}`, variables) });
  escribir(hoja, "00902");
  assert.match(visible(hoja), /1 match out of 2 posts on this page/u);
  assert.ok(nodos(hoja).some((n) => n.atributos["aria-label"]?.includes("00902") && n.atributos["aria-label"]?.includes("2")), "fila original de la página");
  escribir(hoja, "no match"); assert.match(visible(hoja), /No posts on this page match the search/u);
  assert.equal(en.traducir("general.coincidencias_pagina", { cuenta: 2, total: 2 }), "2 matches out of 2 posts on this page");
});
