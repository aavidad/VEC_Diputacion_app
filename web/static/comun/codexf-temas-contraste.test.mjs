import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const css = await readFile(new URL("./tema-vec.css", import.meta.url), "utf8");
const portal = await readFile(new URL("../portal-empleado/portal.css", import.meta.url), "utf8");
const componentes = await readFile(new URL("../portal-empleado/portal-componentes.css", import.meta.url), "utf8");
const menuBolsa = await readFile(new URL("../portal-empleado/portal-menu-bolsa.css", import.meta.url), "utf8");
const modos = ["diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"];

function bloque(selector) {
  const inicio = css.indexOf(`${selector} {`);
  assert.ok(inicio >= 0, `falta ${selector}`);
  const cuerpo = css.slice(inicio + selector.length + 2, css.indexOf("}", inicio));
  return Object.fromEntries([...cuerpo.matchAll(/(--portal-[\w-]+):\s*([^;]+);/g)]
    .map(([, clave, valor]) => [clave, valor.trim()]));
}

const base = bloque(":root");

function color(paleta, token) {
  const valor = paleta[`--portal-${token}`];
  assert.ok(valor, `falta ${token}`);
  const variable = valor.match(/^var\(--portal-([\w-]+)\)$/);
  return variable ? color(paleta, variable[1]) : valor;
}

function luminancia(hex) {
  const rgb = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex);
  assert.ok(rgb, `se esperaba color opaco, recibido ${hex}`);
  const [rojo, verde, azul] = rgb.slice(1).map((canal) => {
    const valor = Number.parseInt(canal, 16) / 255;
    return valor <= 0.04045 ? valor / 12.92 : ((valor + 0.055) / 1.055) ** 2.4;
  });
  return rojo * 0.2126 + verde * 0.7152 + azul * 0.0722;
}

function contraste(paleta, primer, segundo) {
  const valores = [luminancia(color(paleta, primer)), luminancia(color(paleta, segundo))]
    .sort((a, b) => b - a);
  return (valores[0] + 0.05) / (valores[1] + 0.05);
}

test("seis modos solo cambian tokens de color y dejan libre el alto contraste", () => {
  assert.match(css, /body\[data-modo-color\]:not\(\[data-contraste="true"\]\):not\(\.alto-contraste\)\s*\{\s*--portal-lateral-fondo:\s*var\(--portal-azul-950\);/u);
  assert.match(css, /body\[data-contraste="true"\],\s*body\.alto-contraste\s*\{[^}]*--portal-lateral-fondo:\s*var\(--portal-azul-950\);/u);
  for (const modo of modos) {
    const selector = `body[data-modo-color="${modo}"]:not([data-contraste="true"]):not(.alto-contraste)`;
    const paleta = bloque(selector);
    assert.ok(Object.keys(paleta).length >= 17, `${modo}: paleta incompleta`);
    assert.ok(Object.keys(paleta).every((clave) => clave in base), `${modo}: token ajeno al sistema`);
    const contenido = css.slice(css.indexOf(`${selector} {`) + selector.length + 2,
      css.indexOf("}", css.indexOf(`${selector} {`)));
    assert.doesNotMatch(contenido, /(?:^|;)\s*(?:display|position|width|height|margin|padding|font|border-radius|grid-template)\s*:/m);
  }
  assert.deepEqual([...css.matchAll(/^body\[data-modo-color="([^\"]+)"\]:not\(\[data-contraste="true"\]\):not\(\.alto-contraste\) \{/gm)]
    .map((match) => match[1]).filter((modo) => modos.includes(modo)), modos);
});

test("texto, acciones, navegación, estados y foco cumplen WCAG 2.2 AA", () => {
  for (const modo of modos) {
    const paleta = { ...base, ...bloque(`body[data-modo-color="${modo}"]:not([data-contraste="true"]):not(.alto-contraste)`) };
    const exigir = (a, b, minimo) => {
      const razon = contraste(paleta, a, b);
      assert.ok(razon >= minimo, `${modo}: ${a} / ${b} = ${razon.toFixed(2)} < ${minimo}`);
    };
    for (const fondo of ["superficie", "superficie-alterna", "cabecera-panel"]) {
      exigir("tinta", fondo, 4.5);
      exigir("muted", fondo, 4.5);
      exigir("azul-700", fondo, 4.5);
    }
    for (const boton of ["azul-700", "azul-800", "azul-900"]) {
      exigir("texto-inverso", boton, 4.5);
    }
    exigir("texto-inverso", "azul-950", 4.5);
    exigir("texto-inverso", "peligro", 4.5);
    exigir("lateral-texto", "lateral-fondo", 4.5);
    exigir("lateral-muted", "lateral-fondo", 4.5);
    exigir("azul-900", "azul-100", 4.5);
    exigir("foco", "superficie", 3);
    exigir("foco", "fondo", 3);
    exigir("borde", "superficie", 3);
    for (const [texto, fondo] of [
      ["exito", "exito-suave"], ["aviso", "aviso-suave"],
      ["peligro", "peligro-suave"], ["violeta", "violeta-suave"],
      ["cian-fuerte", "cian-suave"], ["naranja", "naranja-suave"],
    ]) exigir(texto, fondo, 4.5);
  }
});

test("marca y enlaces del lateral usan el par de tokens comprobado", () => {
  const declaracion = (hoja, selector) => {
    const inicio = hoja.indexOf(`${selector} {`);
    assert.ok(inicio >= 0, `falta ${selector}`);
    return hoja.slice(inicio, hoja.indexOf("}", inicio));
  };
  for (const selector of [".portal-lateral", ".marca-portal"]) {
    assert.match(declaracion(portal, selector), /background: var\(--portal-lateral-fondo\)/u);
    assert.match(declaracion(portal, selector), /color: var\(--portal-lateral-texto\)/u);
  }
  for (const selector of [".enlace-lateral", ".enlace-lateral:hover:not(:disabled)"]) {
    assert.match(declaracion(portal, selector), /color: var\(--portal-lateral-texto\)/u);
  }
  assert.match(declaracion(menuBolsa, ".enlace-submenu:hover:not(:disabled)"), /color: var\(--portal-lateral-texto\)/u);
  for (const selector of [".boton-avisos span", ".avatar"]) {
    assert.match(declaracion(portal, selector), /color: var\(--portal-texto-inverso\)/u);
  }
  assert.match(declaracion(componentes, '.paginacion-marco button[aria-current="page"]'), /color: var\(--portal-texto-inverso\)/u);
  assert.match(css, /\.portal-lateral \.enlace-submenu\[aria-current="page"\]:focus-visible,[^}]*\.portal-lateral \.categoria-menu-bolsa\[data-categoria-activa="true"\]:focus-visible\s*\{\s*outline-color: var\(--portal-texto-inverso\);/u);
  assert.doesNotMatch(css, /\.portal-lateral \.enlace-lateral\[aria-current="page"\]:focus-visible/u);
});
