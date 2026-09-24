import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { renderizarEstadoEntrega } from "./estado-entrega.js";

const directorio = new URL("./", import.meta.url);
const [pagina, tema, estilos] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("estado-entrega.css", directorio), "utf8"),
]);

test("el estado bloqueado usa el peligro del tema común en el portal que lo carga", () => {
  const html = renderizarEstadoEntrega({
    estado: "bloqueado_dependencia",
    resumen: "Conexión pendiente.",
    pendientes: ["Activar el servicio."],
  });

  assert.match(pagina, /<link rel="stylesheet" href="\/portal-empleado\/portal\.css\?[^"\s]+">/u);
  assert.match(pagina, /<link rel="stylesheet" href="\/portal-empleado\/estado-entrega\.css\?[^"\s]+">/u);
  assert.match(html, /class="estado-entrega estado-entrega--bloqueado"/u);
  assert.match(tema, /--portal-peligro:\s*#[0-9a-f]{6}\s*;/iu);
  assert.match(estilos, /\.estado-entrega--bloqueado\s*\{\s*--estado-entrega-acento:\s*var\(--portal-peligro(?:,\s*#[0-9a-f]{6})?\);\s*\}/iu);
  assert.doesNotMatch(estilos, /var\(--portal-error\b/u);
});
