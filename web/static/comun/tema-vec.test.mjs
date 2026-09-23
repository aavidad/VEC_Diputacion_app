import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const portal = await readFile(new URL("../portal-empleado/portal.css", import.meta.url), "utf8");
const tema = await readFile(new URL("./tema-vec.css", import.meta.url), "utf8");

function tokens(css, selector) {
  const inicio = css.indexOf(`${selector} {`);
  assert.ok(inicio >= 0, `falta ${selector}`);
  const cuerpo = css.slice(inicio + selector.length + 2, css.indexOf("}", inicio));
  return Object.fromEntries([...cuerpo.matchAll(/(--portal-[\w-]+):\s*([^;]+);/g)].map(([, clave, valor]) => [clave, valor.trim()]));
}

function luminancia(hex) {
  const canales = hex.match(/^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i);
  assert.ok(canales, `color opaco esperado: ${hex}`);
  const [r, g, b] = canales.slice(1).map((canal) => {
    const s = Number.parseInt(canal, 16) / 255;
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contraste(a, b) {
  const [claro, oscuro] = [luminancia(a), luminancia(b)].sort((x, y) => y - x);
  return (claro + 0.05) / (oscuro + 0.05);
}

const base = tokens(portal, ":root");
const granate = tokens(tema, 'html[data-tema="granate"]');
const altoContraste = tokens(portal, 'body.portal-empleado-app[data-contraste="true"]');

test("catálogo F2 cerrado: institucional heredado y granate cromático", () => {
  assert.deepEqual([...tema.matchAll(/html\[data-tema="([^"]+)"\]/g)].map((m) => m[1]), ["granate"]);
  assert.match(portal, /--portal-fondo:\s*#eaf1f8/);
  assert.ok(Object.keys(granate).every((clave) => clave in base));
  assert.deepEqual(Object.keys(granate).filter((clave) => /exito|aviso|peligro|violeta|cian|naranja/.test(clave)), []);
  assert.doesNotMatch(tema, /@import|url\(|(?:^|[;{]\s*)(?:display|position|grid-template|padding|margin|width|height|font-size)\s*:/im);
});

test("lienzo tintado, panel claro y texto/foco legibles en ambas paletas", () => {
  for (const paleta of [base, { ...base, ...granate }]) {
    assert.notEqual(paleta["--portal-fondo"], paleta["--portal-superficie"]);
    assert.notEqual(paleta["--portal-superficie-alterna"], paleta["--portal-superficie"]);
    assert.notEqual(paleta["--portal-cabecera-panel"], paleta["--portal-superficie"]);
    for (const tinta of ["--portal-tinta", "--portal-muted", "--portal-azul-700", "--portal-azul-600"]) {
      assert.ok(contraste(paleta[tinta], paleta["--portal-superficie"]) >= 4.5, `${tinta} sobre panel`);
    }
    assert.ok(contraste(paleta["--portal-texto-inverso"], paleta["--portal-azul-600"]) >= 4.5);
  }
});

test("alto contraste prevalece sobre todos los colores de paleta", () => {
  for (const clave of Object.keys(granate)) {
    assert.ok(clave in altoContraste, `${clave} debe prevalecer`);
  }
  assert.equal(altoContraste["--portal-foco"], "var(--portal-azul-700)");
  assert.equal(altoContraste["--portal-acento"], "var(--portal-azul-700)");
  assert.ok(contraste(altoContraste["--portal-azul-700"], altoContraste["--portal-superficie"]) >= 7);
  assert.ok(contraste(altoContraste["--portal-tinta"], altoContraste["--portal-superficie"]) >= 7);
  assert.match(portal, /outline:\s*3px solid var\(--portal-azul-700\)/);
});

function regla(css, selector) {
  const inicio = css.indexOf(`${selector} {`);
  assert.ok(inicio >= 0, `falta ${selector}`);
  return css.slice(inicio + selector.length + 2, css.indexOf("}", inicio));
}

test("navegación activa y botones usan tokens de ambas paletas también al interactuar", () => {
  const reglas = [
    regla(portal, '.enlace-lateral[aria-current="page"]'),
    regla(portal, ".boton-primario"),
    regla(portal, ".boton-primario:hover"),
    regla(portal, ".boton-primario:active"),
    regla(portal, ".boton-secundario"),
    regla(portal, ".boton-secundario:hover"),
    regla(portal, ".boton-secundario:active"),
  ];
  for (const cuerpo of reglas) {
    assert.doesNotMatch(cuerpo, /#[0-9a-f]{3,8}\b|rgba?\(/i, "color fijo en estado interactivo");
    assert.match(cuerpo, /var\(--portal-/);
  }
  assert.match(reglas[0], /background:\s*var\(--portal-azul-700\)/);
  assert.match(reglas[2], /background:\s*var\(--portal-azul-800\)/);
  assert.match(reglas[5], /background:\s*var\(--portal-azul-100\)/);
  for (const paleta of [base, { ...base, ...granate }, { ...base, ...granate, ...altoContraste }]) {
    for (const acento of ["--portal-azul-700", "--portal-azul-800", "--portal-azul-900"]) {
      assert.ok(contraste(paleta["--portal-texto-inverso"], paleta[acento]) >= 4.5, `${acento} en acción principal`);
    }
    assert.ok(contraste(paleta["--portal-azul-900"], paleta["--portal-azul-100"]) >= 4.5, "acción secundaria al pasar el ratón");
  }
});
