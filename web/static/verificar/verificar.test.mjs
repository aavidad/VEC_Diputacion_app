import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const directorio = new URL("./", import.meta.url);
const [html, javascript] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("verificar.js", directorio), "utf8"),
]);

test("la pantalla de cotejo usa POST, omite cookies y no revela documentos", () => {
  assert.match(javascript, /method: "POST"/);
  assert.match(javascript, /credentials: "omit"/);
  assert.match(javascript, /vec\.documentos\.cotejo\.publico\.solicitud\.v1/);
  assert.doesNotMatch(javascript, /localStorage|sessionStorage|document\.cookie/);
  assert.match(html, /El QR facilita el acceso al servicio, pero no acredita por sí mismo/);
  assert.match(html, /requieren autenticación adicional/);
  assert.match(html, /\/verificar\/verificar\.js\?v=20260925-cotejo-publico-v1/);
});

test("el cotejo no ofrece una rama de presentación ni un resultado local", () => {
  assert.doesNotMatch(javascript, /presentacion|adaptador-presentacion|DEMO/);
  assert.doesNotMatch(html, /aviso-presentacion|Resultado DEMO/);
  assert.match(javascript, /clave !== "ref"/);
});
