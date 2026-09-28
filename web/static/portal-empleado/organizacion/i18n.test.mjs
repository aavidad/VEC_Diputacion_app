import assert from "node:assert/strict";
import test from "node:test";
import {
  MENSAJES_ORGANIZACION_EN,
  MENSAJES_ORGANIZACION_ES,
  crearTraductorOrganizacion,
} from "./i18n.js";

const variables = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/gu)].map(([, clave]) => clave).sort();

test("el catálogo inglés conserva todas las claves y marcadores del catálogo castellano", () => {
  assert.deepEqual(Object.keys(MENSAJES_ORGANIZACION_EN).sort(), Object.keys(MENSAJES_ORGANIZACION_ES).sort());
  for (const clave of Object.keys(MENSAJES_ORGANIZACION_ES)) {
    assert.deepEqual(variables(MENSAJES_ORGANIZACION_EN[clave]), variables(MENSAJES_ORGANIZACION_ES[clave]), clave);
  }
});

test("el traductor inglés conserva las advertencias de preparación y el plural", () => {
  const t = crearTraductorOrganizacion("en");
  assert.match(t("importPublishBlocked"), /disabled/i);
  assert.match(t("notice"), /does not prove/i);
  assert.equal(t("importFactsOne", { total: "1" }), "1 typed fact");
  assert.equal(t("importFactsMany", { total: "2" }), "2 typed facts");
  assert.equal(t("count", { visible: "1", total: "2" }), "1 of 2 units");
});
