import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const [html, css, aplicacion] = await Promise.all([
  readFile(new URL("./index.html", import.meta.url), "utf8"),
  readFile(new URL("./area-personal.css", import.meta.url), "utf8"),
  readFile(new URL("./aplicacion.js", import.meta.url), "utf8"),
]);

function bloque(selector) {
  const inicio = css.indexOf(`${selector} {`);
  assert.ok(inicio >= 0, `falta ${selector}`);
  return css.slice(inicio + selector.length + 2, css.indexOf("}", inicio));
}

test("el área personal acepta la paleta común sin cargar activos no autorizados", () => {
  assert.match(html, /<html lang="es" data-tema="institucional">/);
  assert.doesNotMatch(html + css + aplicacion, /(?:src|href)="\/comun\/|@import[^;]*\/comun\//);
  const alias = bloque("body.area-personal-app");
  for (const [propiedad, comun] of [
    ["azul-950", "azul-950"], ["azul-700", "azul-700"], ["fondo", "fondo"],
    ["superficie", "superficie"], ["superficie-alterna", "superficie-alterna"],
    ["cabecera-panel", "cabecera-panel"], ["borde", "borde"], ["sombra", "sombra-sm"],
    ["verde", "exito"], ["rojo", "peligro"], ["ambar", "aviso"],
  ]) {
    assert.match(alias, new RegExp(`--ap-${propiedad}:\\s*var\\(--portal-${comun},`));
  }
  assert.match(css, /\.ap-navegacion > a:hover, \.ap-navegacion > a\[aria-current="page"\][^}]*background: var\(--ap-azul-700\)/s);
  assert.match(css, /\.panel > header[^}]*background:var\(--ap-cabecera-panel\)/s);
  assert.match(css, /input\[type="checkbox"\], input\[type="radio"\] \{ accent-color: var\(--ap-azul-700\)/);
});

test("alto contraste mantiene una capa propia y el foco de teclado visible", () => {
  const contraste = bloque('body.area-personal-app[data-contraste="true"]');
  for (const token of ["azul-950", "azul-900", "azul-800", "azul-700", "azul-100", "tinta", "muted", "borde", "fondo", "superficie", "superficie-alterna", "cabecera-panel", "lateral-muted"]) {
    assert.match(contraste, new RegExp(`--ap-${token}:`), token);
  }
  assert.match(css, /outline: 3px solid var\(--ap-azul-700\)/);
  assert.match(css, /\.ap-lateral :focus-visible \{ outline-color: var\(--ap-texto-inverso\)/);
  assert.doesNotMatch(css, /\.campo input:focus[^}]*outline:none/s);
  assert.match(aplicacion, /destino\.dataset\[atributo\] = String\(activo\)/);
  assert.doesNotMatch(aplicacion, /localStorage|sessionStorage|document\.cookie/);
});
