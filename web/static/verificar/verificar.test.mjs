import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { cotejarDocumentoPresentacion } from "./adaptador-presentacion.js";

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
});

test("el adaptador de presentación reconoce solo recibos DEMO cerrados", () => {
  const valido = cotejarDocumentoPresentacion("DEMO-REC-DIE-0073-06");
  assert.equal(valido.valido, true);
  assert.match(valido.estado, /sin validez administrativa/);
  assert.equal(cotejarDocumentoPresentacion("DOC-REAL-001").valido, false);
  assert.equal(cotejarDocumentoPresentacion("DEMO-REC-DIE-../../clave").valido, false);
});
