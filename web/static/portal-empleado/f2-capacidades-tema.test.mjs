import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const directorio = new URL("./", import.meta.url);
const [html, estilos, tema, vista] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal-capacidades.css", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-vistas-convocatorias.js", directorio), "utf8"),
]);

test("el formulario de capacidades montado sigue el fondo alterno del tema común", () => {
  assert.match(html, /<link rel="stylesheet" href="\/portal-empleado\/portal-capacidades\.css\?v=[^"]+">/);
  assert.match(vista, /formulario-gobernado/);
  assert.match(tema, /:root\s*\{[^}]*--portal-superficie-alterna\s*:/s);
  assert.match(tema, /body\.portal-empleado-app\[data-contraste="true"\]\s*\{[^}]*--portal-superficie-alterna\s*:/s);
  assert.match(estilos, /\.formulario-gobernado fieldset\s*\{[^}]*background:\s*var\(--portal-superficie-alterna\);/s);
});
