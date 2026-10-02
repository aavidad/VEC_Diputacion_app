import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { crearResumenServiciosCSV, descargarResumenServicios } from "./servicios-descarga.js";

const resultado = (items = [{ desde: "2020-01-01", procedencia: "Diputación; Granada", reconocimiento: 'Acto "reconocido"', estado: "Reconocido" }]) => ({ estado: items.length ? "disponible" : "vacio", fuente: "Registro de Personal", actualizado_en: "2026-10-02T08:00:00Z", fecha_referencia: "2026-10-01", items });

test("CSV Excel: UTF8 BOM, CRLF, delimitador y comillas; sólo campos visibles y corte real", () => {
  const datos = resultado(); datos.persona = "oculto"; datos.items[0].secreto = "privado";
  const csv = crearResumenServiciosCSV(datos);
  assert.equal(csv[0], "\uFEFF"); assert.match(csv, /"Inicio";"Fin";"Procedencia"/u);
  assert.match(csv, /"Diputación; Granada"/u); assert.match(csv, /"Acto ""reconocido"""/u);
  assert.match(csv, /2026-10-01/u); assert.match(csv, /Registro de Personal/u);
  assert.doesNotMatch(csv, /oculto|privado|undefined|null/u);
  assert.equal(csv.replaceAll("\r\n", "").includes("\n"), false);
});

test("fórmulas y controles iniciales nunca quedan al comienzo de una celda", () => {
  for (const valor of ["=1+1", "+1", "-1", "@SUM(A1)", " \t=1", "\r=1", "\u0000+1", "\u200B=1", "\u202E=1", "\uFEFF=1"]) {
    const csv = crearResumenServiciosCSV(resultado([{ procedencia: valor }]));
    assert.ok(csv.includes('"\'' + valor.replace(/[\p{Cc}\p{Cf}]/gu, " ").replaceAll('"','""') + '"'), valor);
  }
});

test("vacío y campos ausentes son explícitos; no se inventa corte ni se calculan periodos", () => {
  const datos = resultado([]); delete datos.fecha_referencia;
  assert.match(crearResumenServiciosCSV(datos), /No indicada en la respuesta/u);
  assert.match(crearResumenServiciosCSV(datos), /La consulta no devuelve servicios/u);
  for (const estado of ["denegado", "error", "no_configurado", "cargando", "excede_limite"]) assert.throws(() => crearResumenServiciosCSV({ ...datos, estado }));
});

test("el enlace temporal y su blob se retiran también si falla la descarga", () => {
  const registro = []; const enlace = { remove() { registro.push("remove"); }, click() { throw new Error("descarga no disponible"); } };
  const d = { createElement: () => enlace, body: { append() {} }, defaultView: { Blob, URL: { createObjectURL() { registro.push("create"); return "blob:local"; }, revokeObjectURL(url) { registro.push(url); } } } };
  assert.throws(() => descargarResumenServicios(d, resultado()));
  assert.deepEqual(registro, ["create", "remove", "blob:local"]);
  assert.equal(enlace.download, "resumen-consulta-servicios.csv");
});

test("los catálogos ES/EN tienen las mismas claves y todas se consumen", () => {
  const catalogo = (lang) => JSON.parse(readFileSync(new URL(`../../../textos/${lang}/personal-servicios-descarga.json`, import.meta.url))).general;
  const es = catalogo("es"), en = catalogo("en"); assert.deepEqual(Object.keys(es), Object.keys(en));
  for (const valor of Object.values(en)) assert.ok(typeof valor === "string" && valor.length > 0);
  assert.match(crearResumenServiciosCSV(resultado(), (clave) => en[clave]), /Service query summary/u);
});
