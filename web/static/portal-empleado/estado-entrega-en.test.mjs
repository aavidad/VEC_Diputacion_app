import assert from "node:assert/strict";
import test from "node:test";

globalThis.location = { href: "https://portal.example.test/?lang=en" };
const { renderizarEstadoEntrega } = await import("./estado-entrega.js");

test("lang=en traduce el estado de entrega y conserva el texto aportado por el módulo", () => {
  const html = renderizarEstadoEntrega({
    estado: "visual_pendiente_backend",
    resumen: "Source data remains unchanged",
    pendientes: ["Pending source connection"],
  });
  assert.match(html, /Delivery status: Connection pending/u);
  assert.match(html, /What remains to be done/u);
  assert.match(html, /Source data remains unchanged/u);
  assert.doesNotMatch(html, /Qué falta para terminarlo/u);
});
