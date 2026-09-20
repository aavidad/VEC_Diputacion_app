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
});
