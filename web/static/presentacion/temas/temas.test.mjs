import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { TEMAS_VEC } from "./temas.js";

const hex = (valor) => valor.match(/[0-9a-f]{2}/gi).map((canal) => Number.parseInt(canal, 16));
const luminancia = (valor) => {
  const canales = hex(valor).map((canal) => {
    const normal = canal / 255;
    return normal <= 0.04045 ? normal / 12.92 : ((normal + 0.055) / 1.055) ** 2.4;
  });
  return (0.2126 * canales[0]) + (0.7152 * canales[1]) + (0.0722 * canales[2]);
};
const contraste = (a, b) => {
  const [claro, oscuro] = [luminancia(a), luminancia(b)].sort((x, y) => y - x);
  return (claro + 0.05) / (oscuro + 0.05);
};

test("ofrece exactamente quince temas únicos sobre un contrato de tokens común", () => {
  assert.equal(TEMAS_VEC.length, 15);
  assert.equal(new Set(TEMAS_VEC.map(({ id }) => id)).size, 15);
  assert.equal(new Set(TEMAS_VEC.map(({ nombre }) => nombre)).size, 15);
  const claves = Object.keys(TEMAS_VEC[0]).sort();
  TEMAS_VEC.forEach((tema) => assert.deepEqual(Object.keys(tema).sort(), claves));
});

test("texto, navegación y acción principal mantienen contraste legible", () => {
  for (const tema of TEMAS_VEC) {
    assert.ok(contraste(tema.text, tema.surface) >= 4.5, `${tema.nombre}: texto/superficie`);
    assert.ok(contraste("#ffffff", tema.rail) >= 4.5, `${tema.nombre}: lateral`);
    assert.ok(contraste("#ffffff", tema.primary) >= 4.5, `${tema.nombre}: botón`);
  }
});

test("la galería compara el mismo mini-recorrido y no conserva preferencias", async () => {
  const base = new URL(".", import.meta.url);
  const [html, js, css] = await Promise.all([
    readFile(new URL("index.html", base), "utf8"),
    readFile(new URL("temas.js", base), "utf8"),
    readFile(new URL("temas.css", base), "utf8"),
  ]);
  assert.match(html, /<template id="plantilla-tema">/);
  assert.match(html, /Mapa vacío centrado en Granada/);
  assert.match(css, /grid-template-columns:\s*repeat\(3,/);
  assert.match(css, /@media \(max-width: 46rem\)[\s\S]*?grid-template-columns:\s*1fr/);
  assert.doesNotMatch(js, /localStorage|sessionStorage|document\.cookie|fetch\(/);
});
