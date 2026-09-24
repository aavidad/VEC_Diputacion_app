import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const [html, css, tema] = await Promise.all([
  readFile(new URL("./index.html", import.meta.url), "utf8"),
  readFile(new URL("./verificar.css", import.meta.url), "utf8"),
  readFile(new URL("../comun/tema-vec.css", import.meta.url), "utf8"),
]);

function bloque(fuente, selector) {
  const inicio = fuente.indexOf(`${selector} {`);
  assert.ok(inicio >= 0, `Falta ${selector}`);
  return fuente.slice(inicio + selector.length + 2, fuente.indexOf("}", inicio));
}

function valores(fuente, selector) {
  return Object.fromEntries([...bloque(fuente, selector).matchAll(/(--portal-[\w-]+):\s*([^;]+);/g)]
    .map(([, nombre, valor]) => [nombre, valor.trim()]));
}

function color(nombre, paleta) {
  const valor = paleta[`--portal-${nombre}`];
  assert.ok(valor, `Falta --portal-${nombre}`);
  const referencia = /^var\(--portal-([\w-]+)\)$/.exec(valor);
  return referencia ? color(referencia[1], paleta) : valor;
}

function luminancia(hex) {
  const rgb = /^#([\da-f]{2})([\da-f]{2})([\da-f]{2})$/i.exec(hex);
  assert.ok(rgb, `Color hexadecimal opaco esperado: ${hex}`);
  const [rojo, verde, azul] = rgb.slice(1).map((canal) => {
    const normalizado = Number.parseInt(canal, 16) / 255;
    return normalizado <= 0.04045 ? normalizado / 12.92 : ((normalizado + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * rojo + 0.7152 * verde + 0.0722 * azul;
}

function contraste(a, b) {
  const [alto, bajo] = [luminancia(a), luminancia(b)].sort((x, y) => y - x);
  return (alto + 0.05) / (bajo + 0.05);
}

const base = valores(tema, ":root");
const granate = { ...base, ...valores(tema, 'html[data-tema="granate"]') };
const contrasteAlto = { ...granate, ...valores(tema, "body.alto-contraste") };

test("Verificar carga la hoja común antes de su CSS versionado y conserva el cotejo público", () => {
  const hojas = [...html.matchAll(/<link rel="stylesheet" href="([^"]+)"/g)].map((coincidencia) => coincidencia[1]);
  assert.deepEqual(hojas.map((href) => href.split("?", 1)[0]), ["/comun/tema-vec.css", "/verificar/verificar.css"]);
  assert.equal(hojas[1], "/verificar/verificar.css?v=20260924-f2-tema-verificar-v1");
  // tema-vec.css cambió después de esta versión publicada: pide una URL nueva.
  const versionTema = new URL(hojas[0], "https://vec.example").searchParams.get("v");
  assert.ok(versionTema && versionTema !== "20260924-f2-tema-base-v2", "tema-vec.css renueva su URL");
  assert.match(html, /<form id="formulario-cotejo">/);
  assert.match(html, /Resultado DEMO\./);
  assert.match(html, /no acredita autenticidad, registro ni firma/);
});

test("el patrón HTML admite las mismas referencias bajo RegExp v de Chromium", () => {
  const campo = /<input id="referencia"[^>]+>/.exec(html)?.[0];
  assert.ok(campo, "Falta el campo de referencia");
  const patron = /\bpattern="([^"]+)"/.exec(campo)?.[1];
  assert.ok(patron, "Falta el patrón HTML");
  const regex = new RegExp(`^(?:${patron})$`, "v");
  const minimo = Number(/\bminlength="(\d+)"/.exec(campo)?.[1]);
  const maximo = Number(/\bmaxlength="(\d+)"/.exec(campo)?.[1]);
  const valida = (referencia) => referencia.length >= minimo && referencia.length <= maximo && regex.test(referencia);
  for (const referencia of ["ABCDEFGH", "SINTETICO-REC-0001", "A._:-123", "A" + "a".repeat(79)]) {
    assert.equal(valida(referencia), true, referencia);
  }
  for (const referencia of ["ABCDEFG", "-ABCDEFG", "A/B/CDEF", "A\\B-CDEF", "A B-CDEF", "A" + "a".repeat(80)]) {
    assert.equal(valida(referencia), false, referencia);
  }
});

test("los controles, resultados y aviso DEMO heredan tokens y admiten alto contraste", () => {
  assert.doesNotMatch(css, /#[\da-f]{3,8}\b|\brgba?\(/i, "No debe haber colores locales fijos");
  assert.doesNotMatch(css, /--portal-[\w-]+\s*:/, "La paleta solo se define en la hoja común");
  for (const referencia of css.matchAll(/var\((--portal-[\w-]+)\)/g)) {
    assert.ok(referencia[1] in base, `Token común desconocido: ${referencia[1]}`);
  }
  for (const [selector, texto, fondo] of [
    ["body", "tinta", "fondo"],
    ["input", "tinta", "superficie"],
    ["button", "texto-inverso", "azul-700"],
    ['.resultado[data-estado="valido"]', "exito", "exito-suave"],
    ['.resultado[data-estado="error"]', "peligro", "peligro-suave"],
    ["#aviso-presentacion", "aviso", "aviso-suave"],
  ]) {
    const regla = bloque(css, selector);
    assert.match(regla, new RegExp(`color:\\s*var\\(--portal-${texto}\\)`));
    assert.match(regla, new RegExp(`background:\\s*var\\(--portal-${fondo}\\)`));
    for (const [nombre, paleta] of [["institucional", base], ["granate", granate], ["alto contraste", contrasteAlto]]) {
      assert.ok(contraste(color(texto, paleta), color(fondo, paleta)) >= 4.5,
        `${selector} debe ser legible en ${nombre}`);
    }
  }
  for (const selector of [".salto:focus", "a:focus-visible", "input:focus", "button:focus-visible"]) {
    assert.match(bloque(css, selector), /outline:\s*3px solid var\(--portal-foco\)/);
  }
});
