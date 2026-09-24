import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { renderizarErrorCargaAreaPersonal } from "./aplicacion.js";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";

const prefijo = "areaPersonal.estado.error.";

test("401, 403, 404 y 503 muestran textos del traductor sin exponer mensajes del transporte", () => {
  const casos = [
    ["autenticacion_requerida", "autenticacion.titulo", "autenticacion.detalle", true],
    ["acceso_denegado", "acceso.titulo", "acceso.detalle", false],
    ["recurso_no_encontrado", "recurso.titulo", "recurso.detalle", true],
    ["servicio_no_disponible", "titulo", "servicio.detalle", true],
  ];
  for (const [codigo, titulo, detalle, reintentar] of casos) {
    const error = Object.assign(new Error("Mensaje HTTP ajeno <script>alert(1)</script>"), { codigo });
    const html = renderizarErrorCargaAreaPersonal(error);
    assert.match(html, /role="alert"/);
    assert.ok(html.includes(traducir(`${prefijo}${titulo}`)), codigo);
    assert.ok(html.includes(traducir(`${prefijo}${detalle}`)), codigo);
    assert.equal(html.includes('data-accion="reintentar"'), reintentar, codigo);
    assert.doesNotMatch(html, /Mensaje HTTP ajeno|<script>|alert\(1\)/);
  }
});

test("una caída de red conserva un mensaje propio y no se confunde con el 503", () => {
  const error = Object.assign(new Error("fetch failed"), {
    codigo: "servicio_no_disponible", cause: new TypeError("socket cerrado"),
  });
  const html = renderizarErrorCargaAreaPersonal(error);
  assert.ok(html.includes(traducir(`${prefijo}red.detalle`)));
  assert.ok(!html.includes(traducir(`${prefijo}servicio.detalle`)));
  assert.doesNotMatch(html, /fetch failed|socket cerrado/);
});

test("errores desconocidos usan claves genéricas y escapan el catálogo real", async () => {
  const documento = { querySelectorAll: () => [] };
  await iniciarI18nAreaPersonal(documento, async () => ({
    ok: true,
    json: async () => ({ [`${prefijo}titulo`]: '<img src=x onerror="alert(1)">' }),
  }));
  const html = renderizarErrorCargaAreaPersonal(Object.assign(new Error("secreto"), { codigo: "sin_catalogar" }));
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/);
  assert.doesNotMatch(html, /<img|secreto/);
  assert.ok(html.includes(traducir(`${prefijo}carga.garantia`)));
});

test("la vista no conserva las frases de error embebidas ni consume almacenamiento", async () => {
  const aplicacion = await readFile(new URL("./aplicacion.js", import.meta.url), "utf8");
  const cuerpo = aplicacion.slice(aplicacion.indexOf("export function renderizarErrorCargaAreaPersonal"), aplicacion.indexOf("function datosMinimosMiBolsa"));
  assert.doesNotMatch(cuerpo, /error\.message|Identifíquese para consultar|No tiene acceso a esta área/);
  assert.doesNotMatch(cuerpo, /localStorage|sessionStorage|document\.cookie/);
});
