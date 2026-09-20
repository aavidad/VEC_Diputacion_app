import assert from "node:assert/strict";
import test from "node:test";
import { MENSAJES_PERSONAL_ES, crearTraductorPersonal, formatearRecuentoCategorias, formatearRecuentoRPT } from "./i18n.js";

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

test("el catálogo conectado conserva textos y plural localizados", () => {
  const t = crearTraductorPersonal();
  assert.match(t("catalogo_error"), /No se muestran datos anteriores/);
  assert.match(t("catalogo_demo"), /demostracion:true/);
  assert.equal(formatearRecuentoCategorias(1), "1 categoría");
  assert.equal(formatearRecuentoCategorias(1_000), "1000 categorías");
  assert.throws(() => formatearRecuentoCategorias(-1), /no válido/);
});

test("la RPT pública tiene catálogo completo y recuento localizado", () => {
  const t = crearTraductorPersonal();
  ["rpt_titulo", "rpt_ayuda", "rpt_error", "rpt_fuente", "rpt_huella", "rpt_vacio", "rpt_tabla", "rpt_paginacion", "rpt_cabecera_dotacion"].forEach((clave) => assert.notEqual(t(clave), ""));
  assert.match(t("rpt_fuente", { documento: "RPT", aviso: "sin ocupantes" }), /RPT/);
  assert.equal(formatearRecuentoRPT(1), "1 categoría RPT");
  assert.equal(formatearRecuentoRPT(1_000), "1000 categorías RPT");
  assert.throws(() => formatearRecuentoRPT(-1), /no válido/);
});
