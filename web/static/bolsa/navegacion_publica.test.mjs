import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const directorio = dirname(fileURLToPath(import.meta.url));
const html = readFileSync(join(directorio, "index.html"), "utf8");
const css = readFileSync(join(directorio, "bolsa.css"), "utf8");
const menu = html.match(/<aside class="menu-lateral-publico"[\s\S]*?<\/aside>/)?.[0] || "";
const destinosPermitidos = [
  "/bolsa/",
  "#contenido-principal",
  "#filtros-convocatorias",
  "#directorio-categorias",
  "#ayuda-publica",
];

function sinBloquesDeImpresion(contenido) {
  let resultado = "";
  let cursor = 0;

  while (cursor < contenido.length) {
    const coincidencia = /@media\s+print\b/gi.exec(contenido.slice(cursor));
    if (!coincidencia) return resultado + contenido.slice(cursor);

    const inicio = cursor + coincidencia.index;
    const apertura = contenido.indexOf("{", inicio);
    assert.notEqual(apertura, -1, "el bloque @media print debe estar bien formado");
    resultado += contenido.slice(cursor, inicio);

    let profundidad = 1;
    let posicion = apertura + 1;
    while (posicion < contenido.length && profundidad > 0) {
      if (contenido[posicion] === "{") profundidad += 1;
      if (contenido[posicion] === "}") profundidad -= 1;
      posicion += 1;
    }
    assert.equal(profundidad, 0, "el bloque @media print debe estar cerrado");
    cursor = posicion;
  }

  return resultado;
}

test("la bolsa pública conserva una navegación lateral limitada a contenido público", () => {
  assert.match(menu, /Navegación del portal público de empleo/);
  const destinosPresentes = [...menu.matchAll(/<a\b[^>]*\bhref="([^"]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.deepEqual(destinosPresentes, destinosPermitidos);
  assert.doesNotMatch(menu, /Cronos|Nóminas|Dietas|Administración|Auditoría/);
});

test("todos los destinos del menú público existen en el documento", () => {
  const destinos = [...menu.matchAll(/href="#([^"]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.deepEqual(destinos, ["contenido-principal", "filtros-convocatorias", "directorio-categorias", "ayuda-publica"]);
  destinos.forEach((id) => assert.match(html, new RegExp(`id="${id}"`)));
});

test("el menú es lateral en escritorio y se adapta sin ocultarse en móvil", () => {
  assert.match(css, /\.portal-publico-shell\s*\{[^}]*grid-template-columns:\s*272px minmax\(0, 1fr\)/s);
  assert.match(css, /@media \(max-width: 800px\)[\s\S]*?\.portal-publico-shell\s*\{\s*grid-template-columns:\s*1fr;/);
  assert.match(css, /@media \(max-width: 800px\)[\s\S]*?\.navegacion-publica\s*\{[^}]*grid-template-columns:/);

  const cssDePantalla = sinBloquesDeImpresion(css);
  const reglasDelMenu = [...cssDePantalla.matchAll(/([^{}]+)\{([^{}]*)\}/g)]
    .filter((coincidencia) => coincidencia[1].includes(".menu-lateral-publico"));
  assert.ok(reglasDelMenu.length > 0, "debe existir al menos una regla de pantalla para el menú público");
  reglasDelMenu.forEach(([, selector, declaraciones]) => {
    assert.doesNotMatch(declaraciones, /\bdisplay\s*:\s*none\b/i, `${selector.trim()} no puede retirar el menú en pantalla`);
    assert.doesNotMatch(declaraciones, /\bvisibility\s*:\s*hidden\b/i, `${selector.trim()} no puede ocultar el menú en pantalla`);
    assert.doesNotMatch(declaraciones, /\bcontent-visibility\s*:\s*hidden\b/i, `${selector.trim()} no puede omitir el menú en pantalla`);
  });
});
