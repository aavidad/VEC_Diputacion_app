import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const leer = (ruta) => readFile(new URL(ruta, raiz), "utf8");
const [patrones, pagina, tema] = await Promise.all([
  leer("portal-patrones.css"), leer("index.html"), leer("../comun/tema-vec.css"),
]);

test("los patrones comunes existen una sola vez en el tema del portal", () => {
  for (const clase of ["siguiente-paso", "linea-fases", "lista-documentos", "historial-frases",
    "filtros-quitables", "filtros-activos", "chip-quitar", "pasos-numerados", "tareas-pendientes",
    "tabla-apilable", "opciones-grandes", "resumen-errores", "revision-datos"]) {
    assert.match(patrones, new RegExp(`\\.${clase}\\b`), clase);
  }
  assert.match(pagina, /href="\/portal-empleado\/portal-patrones\.css\?v=[^"]+"/u);
});

test("los patrones solo usan tokens del tema común, sin colores propios", () => {
  assert.doesNotMatch(patrones, /#[0-9a-fA-F]{3,8}\b/u);
  assert.doesNotMatch(patrones, /\brgba?\(/u);
  const usados = new Set(patrones.match(/--portal-[a-z0-9-]+/gu));
  for (const token of usados) assert.match(tema, new RegExp(`${token}:`), token);
});

test("las fases y los documentos dicen su estado con símbolo o palabra, no solo con color", () => {
  assert.match(patrones, /\.linea-fases \.marca/u);
  assert.match(patrones, /\.lista-documentos \.simbolo/u);
  assert.match(patrones, /\.pasos-numerados > \.hecho::before \{ content: "✓"/u);
});

test("en móvil las tablas apilables pasan a fichas sin ensanchar la página", () => {
  assert.match(patrones, /@media \(max-width: 760px\)[\s\S]*\.tabla-apilable thead \{ position: absolute/u);
  assert.match(patrones, /\.tabla-apilable td\[data-etiqueta\]::before \{ content: attr\(data-etiqueta\)/u);
});
