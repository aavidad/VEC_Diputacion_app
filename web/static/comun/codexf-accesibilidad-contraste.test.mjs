import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const css = await readFile(new URL("./tema-vec.css", import.meta.url), "utf8");
const bloques = [...css.matchAll(/([^{}]+)\{([^{}]+)\}/g)].map(([, selector, cuerpo]) => ({
  selector: selector.trim(),
  tokens: Object.fromEntries([...cuerpo.matchAll(/(--portal-[\w-]+):\s*([^;]+);/g)]
    .map(([, nombre, valor]) => [nombre, valor.trim()])),
}));
const base = bloques.find(({ selector }) => selector.endsWith(":root")).tokens;
function color(tokens, nombre) {
  const valor = tokens[`--portal-${nombre}`];
  const alias = /^var\(--portal-([\w-]+)\)$/.exec(valor);
  return alias ? color(tokens, alias[1]) : valor;
}
function luminancia(hex) {
  const rgb = /^#([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  assert.ok(rgb, `color opaco esperado: ${hex}`);
  return rgb.slice(1).reduce((suma, valor, indice) => {
    const canal = parseInt(valor, 16) / 255;
    return suma + (canal <= .04045 ? canal / 12.92 : ((canal + .055) / 1.055) ** 2.4)
      * [.2126, .7152, .0722][indice];
  }, 0);
}
function contraste(a, b) {
  const [alto, bajo] = [luminancia(a), luminancia(b)].sort((x, y) => y - x);
  return (alto + .05) / (bajo + .05);
}

test("campos y foco mantienen AA en base, granate, claro, oscuro, seis temas y alto contraste", () => {
  const variantes = [{ selector: "base/claro", tokens: {} }, ...bloques.filter(({ selector, tokens }) =>
    Object.keys(tokens).length > 10 && (selector.includes('html[data-tema="granate"]')
      || selector.includes('body[data-modo-color="') || selector.includes("body.alto-contraste")))];
  assert.equal(variantes.length, 11);
  for (const variante of variantes) {
    const oscuro = variante.selector.includes('html[data-tema="granate"] body')
      ? bloques.find(({ selector }) => selector.includes('body[data-modo-color="oscuro"]') && !selector.includes('html[data-tema=')).tokens : {};
    const paleta = { ...base, ...oscuro, ...variante.tokens };
    for (const fondo of ["superficie", "superficie-alterna", "cabecera-panel"]) {
      for (const token of ["tinta", "muted", "azul-700"]) {
        const ratio = contraste(color(paleta, token), color(paleta, fondo));
        assert.ok(ratio >= 4.5, `${variante.selector}: ${token}/${fondo} ${ratio.toFixed(2)}`);
      }
      for (const token of ["borde-control", "foco"]) {
        const ratio = contraste(color(paleta, token), color(paleta, fondo));
        assert.ok(ratio >= 3, `${variante.selector}: ${token}/${fondo} ${ratio.toFixed(2)}`);
      }
    }
  }
});
