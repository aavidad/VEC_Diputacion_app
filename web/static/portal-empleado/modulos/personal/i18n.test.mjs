import assert from "node:assert/strict";
import test from "node:test";
import { MENSAJES_PERSONAL_ES, crearTraductorPersonal } from "./i18n.js";

test("el catálogo de Personal advierte de la naturaleza DEMO y de sus límites", () => {
  const t = crearTraductorPersonal();
  assert.match(t("presentacion_demo"), /DEMO/);
  assert.match(t("aviso_demo"), /no consulta datos reales/i);
  assert.match(t("servicios_ayuda"), /no se calcula antigüedad, trienios/i);
  assert.match(t("nominas_ayuda"), /No se muestran importes/i);
  assert.match(t("dietas_ayuda"), /no acreditan liquidación/i);
  assert.equal(t("actualizado", { fecha: "20/09/2026" }), "Actualizado: 20/09/2026");
  assert.ok(Object.isFrozen(MENSAJES_PERSONAL_ES));
});

test("el traductor rechaza catálogos y claves incompletos", () => {
  assert.throws(() => crearTraductorPersonal({ titulo: "incompleto" }), /incompleto/);
  assert.throws(() => crearTraductorPersonal()("desconocida"), /desconocida/);
});
