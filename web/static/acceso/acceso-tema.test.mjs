import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const [html, acceso, comun] = await Promise.all([
  readFile(new URL("./index.html", import.meta.url), "utf8"),
  readFile(new URL("./acceso.css", import.meta.url), "utf8"),
  readFile(new URL("../comun/tema-vec.css", import.meta.url), "utf8"),
]);

function tokens(selector) {
  const inicio = comun.indexOf(`${selector} {`);
  assert.ok(inicio >= 0, `falta ${selector}`);
  const cuerpo = comun.slice(inicio + selector.length + 2, comun.indexOf("}", inicio));
  return Object.fromEntries([...cuerpo.matchAll(/(--portal-[\w-]+):\s*([^;]+);/gu)]
    .map(([, clave, valor]) => [clave, valor.trim()]));
}

function resolver(nombre, paleta) {
  const valor = paleta[nombre];
  assert.ok(valor, `falta ${nombre}`);
  const alias = /^var\((--portal-[\w-]+)\)$/u.exec(valor);
  return alias ? resolver(alias[1], paleta) : valor;
}

function luminancia(hex) {
  const canales = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/iu.exec(hex);
  assert.ok(canales, `se esperaba un color opaco: ${hex}`);
  const [r, g, b] = canales.slice(1).map((canal) => {
    const valor = Number.parseInt(canal, 16) / 255;
    return valor <= 0.04045 ? valor / 12.92 : ((valor + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contraste(a, b) {
  const [claro, oscuro] = [luminancia(a), luminancia(b)].sort((x, y) => y - x);
  return (claro + 0.05) / (oscuro + 0.05);
}

test("acceso carga el tema común versionado antes de su hoja local", () => {
  const estilos = [...html.matchAll(/<link rel="stylesheet" href="([^"]+)"/gu)].map(([, href]) => href);
  assert.deepEqual(estilos, [
    "/styles.css?v=20260715-theme",
    "/comun/tema-vec.css?v=20260924-f2-tema-base-v2",
    "/acceso/acceso.css?v=20260924-f1-acceso-ayuda-v2",
  ]);
  assert.match(html, /<script type="module" src="\/acceso\/acceso-i18n\.js\?v=20260924-f1-acceso-ayuda-v1"><\/script>/u);
});

test("el texto de ayuda no aparece hasta activar el único botón ?", async () => {
  const catalogo = JSON.parse(await readFile(new URL("./locales/es.json", import.meta.url), "utf8"));
  assert.match(html, /<button[^>]*id="boton-ayuda-acceso"[^>]*aria-expanded="false"[^>]*aria-controls="acceso-autorizacion"[^>]*>\?<\/button>/u);
  assert.match(html, /<p[^>]*id="acceso-autorizacion"[^>]*hidden[^>]*data-i18n="acceso\.ayuda"[^>]*>/u);
  assert.equal((html.match(/id="boton-ayuda-acceso"/gu) ?? []).length, 1);
  assert.equal((html.match(/aria-controls="acceso-autorizacion"/gu) ?? []).length, 1);
  assert.equal(catalogo["acceso.ayuda.boton"], "Información sobre identidad y autorización");
  assert.ok(catalogo["acceso.ayuda"].includes("no conceden acceso por sí solos"));
  assert.doesNotMatch(html, /aria-describedby="[^"]*acceso-autorizacion/u);
  assert.match(acceso, /\.acceso-ayuda\[hidden\]\s*\{\s*display:\s*none;/u);
});

test("acceso solo consume tokens cromáticos declarados por la autoridad común", () => {
  const base = tokens(":root");
  const referencias = [...acceso.matchAll(/var\((--portal-[\w-]+)\)/gu)].map(([, nombre]) => nombre);
  assert.ok(referencias.length > 25);
  for (const nombre of referencias) assert.ok(nombre in base, `${nombre} ausente en tema común`);
  assert.doesNotMatch(acceso, /#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(/iu);
  assert.match(acceso, /\.acceso-publico :is\(button, a, \[tabindex\]\):focus-visible\s*\{\s*outline-color: var\(--portal-foco\);/u);
});

test("lienzo, portada y estados conservan contraste AA en las tres variantes", () => {
  const base = tokens(":root");
  const granate = tokens('html[data-tema="granate"]');
  const altoContraste = tokens("body.alto-contraste");
  const pares = [
    ["tinta", "fondo"],
    ["tinta", "superficie"],
    ["azul-900", "superficie"],
    ["texto-inverso", "azul-950"],
    ["lateral-muted", "azul-950"],
    ["azul-950", "azul-100"],
    ["aviso", "aviso-suave"],
    ["texto-inverso", "aviso"],
    ["muted", "neutro-suave"],
  ];
  for (const [nombre, paleta] of [
    ["institucional", base],
    ["granate", { ...base, ...granate }],
    ["alto contraste", { ...base, ...granate, ...altoContraste }],
  ]) {
    for (const [tinta, fondo] of pares) {
      assert.ok(contraste(resolver(`--portal-${tinta}`, paleta), resolver(`--portal-${fondo}`, paleta)) >= 4.5,
        `${nombre}: ${tinta} sobre ${fondo}`);
    }
  }
});
