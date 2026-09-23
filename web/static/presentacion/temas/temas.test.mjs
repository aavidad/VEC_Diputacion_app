import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { DISENOS_DIETAS } from "./temas.js";

test("ofrece exactamente quince estructuras de Dietas únicas", () => {
  assert.equal(DISENOS_DIETAS.length, 15);
  assert.equal(new Set(DISENOS_DIETAS.map(({ id }) => id)).size, 15);
  assert.equal(new Set(DISENOS_DIETAS.map(({ nombre }) => nombre)).size, 15);
  DISENOS_DIETAS.forEach(({ criterio }) => assert.ok(criterio.length >= 45));
});

test("compara la misma información sin repetir ruido técnico en la tarea", async () => {
  const html = await readFile(new URL("index.html", import.meta.url), "utf8");
  assert.match(html, /<template id="plantilla-diseno">/);
  assert.match(html, /Mis comisiones de servicio/);
  assert.match(html, /class="alta-rutas" hidden/);
  assert.match(html, /class="abrir-alta" aria-expanded="false"/);
  assert.match(html, /\+ Añadir ruta/);
  assert.doesNotMatch(html, /Añadir parada/);
  assert.equal((html.match(/class="comision-detalle" hidden/g) || []).length, 3);
  assert.match(html, /Pendiente de jefatura/);
  assert.match(html, /class="chip chip-gastos">Liquidada/);
  assert.match(html, /Granada/);
  assert.match(html, /Baza/);
  assert.match(html, /Purchil/);
  assert.doesNotMatch(html, /No acredita autorización|no valida kilómetros|no liquida|no ordena pagos|Funcionario DEMO 01/);
});

test("cada alternativa cambia composición y la galería no conserva preferencias", async () => {
  const base = new URL(".", import.meta.url);
  const [js, css] = await Promise.all([
    readFile(new URL("temas.js", base), "utf8"),
    readFile(new URL("temas.css", base), "utf8"),
  ]);
  for (let indice = 1; indice <= 15; indice += 1) {
    const id = String(indice).padStart(2, "0");
    assert.match(css, new RegExp(`data-diseno=\\"${id}\\"`));
  }
  assert.match(css, /grid-template-columns:\s*repeat\(2,/);
  assert.match(css, /@media \(max-width: 74rem\)[^}]*galeria-disenos[^}]*grid-template-columns:\s*1fr/s);
  assert.doesNotMatch(js, /localStorage|sessionStorage|document\.cookie|fetch\(/);
  assert.match(js, /altaRutas\.hidden/u);
  assert.match(js, /comision-detalle/u);
});

test("la galería conserva composición al recibir los colores del tema común", async () => {
  const css = await readFile(new URL("temas.css", import.meta.url), "utf8");
  for (const token of ["azul-950", "azul-900", "azul-800", "azul-700", "azul-100", "fondo", "borde", "superficie", "verde", "ambar"]) {
    assert.match(css, new RegExp(`--${token}:\\s*var\\(--portal-`), token);
  }
  const reglas = css.slice(css.indexOf("* { box-sizing: border-box; }"));
  assert.doesNotMatch(reglas, /#073b6c|#e7f0fb|rgb\(7 95 202/i);
  assert.match(reglas, /\.app-lateral[^}]*var\(--azul-900\)/s);
  assert.match(reglas, /\.mapa-lienzo[^}]*color-mix\(in srgb, var\(--azul-700\)/s);
  assert.match(reglas, /\.propuesta\[data-seleccionado="true"\][^}]*border-color: var\(--azul-700\)/s);
});
