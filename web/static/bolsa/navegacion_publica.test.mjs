import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const directorio = dirname(fileURLToPath(import.meta.url));
const html = readFileSync(join(directorio, "index.html"), "utf8");
const css = readFileSync(join(directorio, "bolsa.css"), "utf8");

test("la bolsa pública conserva una navegación lateral limitada a contenido público", () => {
  const menu = html.match(/<aside class="menu-lateral-publico"[\s\S]*?<\/aside>/)?.[0] || "";
  assert.match(menu, /Navegación del portal público de empleo/);
  assert.match(menu, /href="#contenido-principal"/);
  assert.match(menu, /href="#directorio-categorias"/);
  assert.match(menu, /href="#ayuda-publica"/);
  assert.doesNotMatch(menu, /Cronos|Nóminas|Dietas|Administración|Auditoría/);
});

test("todos los destinos del menú público existen en el documento", () => {
  const menu = html.match(/<aside class="menu-lateral-publico"[\s\S]*?<\/aside>/)?.[0] || "";
  const destinos = [...menu.matchAll(/href="#([^"]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.deepEqual(destinos, ["contenido-principal", "filtros-convocatorias", "directorio-categorias", "ayuda-publica"]);
  destinos.forEach((id) => assert.match(html, new RegExp(`id="${id}"`)));
});

test("el menú es lateral en escritorio y se adapta sin ocultarse en móvil", () => {
  assert.match(css, /\.portal-publico-shell\s*\{[^}]*grid-template-columns:\s*272px minmax\(0, 1fr\)/s);
  assert.match(css, /@media \(max-width: 800px\)[\s\S]*?\.portal-publico-shell\s*\{\s*grid-template-columns:\s*1fr;/);
  assert.match(css, /@media \(max-width: 800px\)[\s\S]*?\.navegacion-publica\s*\{[^}]*grid-template-columns:/);
});
