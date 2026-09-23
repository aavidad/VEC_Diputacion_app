import assert from "node:assert/strict";
import test from "node:test";
import { MENSAJES_PERSONAL_ES, crearTraductorPersonal, formatearFechaEstructuraOrganizativa, formatearRecuentoCategorias, formatearRecuentoRPT } from "./i18n.js";

test("el catálogo de Personal advierte de la naturaleza DEMO y de sus límites", () => {
  const t = crearTraductorPersonal();
  assert.match(t("presentacion_demo"), /DEMO/);
  assert.match(t("aviso_demo"), /no consulta datos reales/i);
  assert.match(t("servicios_ayuda"), /no se calcula antigüedad, trienios/i);
  assert.match(t("nominas_ayuda"), /No se muestran importes/i);
  assert.match(t("dietas_ayuda"), /no acreditan liquidación/i);
  assert.match(t("ficha_accesos_ayuda"), /no envía identificadores ni acredita una relación de servicio/i);
  assert.match(t("ficha_estado_no_configurado"), /Fuente no conectada/i);
  assert.match(t("ficha_catalogos_completos"), /No acreditan ocupación actual/i);
  assert.match(t("ficha_abrir_ayuda"), /^\? /);
  assert.match(t("ficha_tiempo_ayuda"), /fichaje no acredita servicios reconocidos/i);
  assert.match(t("ficha_no_configurado"), /ausencia de datos no equivale a cero/i);
  assert.match(t("ficha_desplazar_tabla"), /horizontalmente/i);
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

test("la fecha de estructura se presenta en castellano y Europe/Madrid", () => {
  const fecha = formatearFechaEstructuraOrganizativa("2026-09-06T00:00:00Z");
  assert.match(fecha, /6 sept 2026|6\/9\/2026|06\/09\/2026/);
  assert.match(fecha, /Europe\/Madrid/);
  assert.doesNotMatch(fecha, /2026-09-06T00:00:00Z/);
  assert.throws(() => formatearFechaEstructuraOrganizativa("2026-09-06"), /no válida/);
});
