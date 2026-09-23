import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const css = readFileSync(new URL("./dietas.css", import.meta.url), "utf8");
const vista = readFileSync(new URL("./vista-borradores-propios.js", import.meta.url), "utf8");
const inicio = css.indexOf("/* Mis comisiones:");
const corte = inicio < 0 ? "" : css.slice(inicio);

test("la vista de borradores conserva el contrato visual de paneles, lista y acciones", () => {
  assert.ok(corte, "falta el corte CSS de Mis comisiones");
  for (const clase of [
    "dietas-borradores-propios",
    "dietas-borradores-espacio",
    "dietas-borradores-principal",
    "dietas-borradores-lateral",
    "dietas-borradores-lista",
    "dietas-borradores-fila",
    "dietas-borradores-paginacion",
    "dietas-borradores-campos",
    "dietas-borradores-acciones",
  ]) {
    assert.match(vista, new RegExp(`\\b${clase}\\b`, "u"), `la vista perdió ${clase}`);
    assert.match(corte, new RegExp(`\\.${clase}\\b`, "u"), `falta CSS para ${clase}`);
  }
  assert.match(vista, /className = "cabecera-panel"/u);
  assert.match(vista, /className = "cuerpo-panel/u);
});

test("Mis comisiones usa tokens compartidos y scroll interno en escritorio", () => {
  assert.doesNotMatch(corte, /#[0-9a-fA-F]{3,8}\b/u);
  assert.match(corte, /background: var\(--portal-fondo\)/u);
  assert.match(corte, /background: var\(--portal-cabecera-panel\)/u);
  assert.match(corte, /@media \(min-width: 1024px\)/u);
  assert.match(corte, /\.modulo-dietas\.dietas-borradores-propios\s*\{[^}]*overflow: hidden/su);
  assert.match(corte, /\.dietas-borradores-propios \.dietas-borradores-lista,[^}]*overflow: auto/su);
  assert.match(corte, /@media \(max-width: 1023px\)/u);
  assert.match(corte, /@media \(max-width: 520px\)/u);
});
