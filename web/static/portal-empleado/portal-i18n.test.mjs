import assert from "node:assert/strict";
import test from "node:test";
import {
  crearTraductorPortal,
  MENSAJES_PORTAL_ES,
} from "./portal-i18n.js";

test("el catálogo i18n cubre los estados nuevos de acceso, navegación y reintento", () => {
  const traducir = crearTraductorPortal();
  for (const clave of Object.keys(MENSAJES_PORTAL_ES)) {
    assert.equal(typeof traducir(clave), "string");
    assert.notEqual(traducir(clave), "");
  }
  assert.match(traducir("acceso_borradores_denegado"), /permiso/);
  assert.match(traducir("accion_reintentar"), /Reintentar/);
});

test("un catálogo incompleto o una clave no gobernada fallan cerrados", () => {
  assert.throws(() => crearTraductorPortal({}), /incompleto/);
  const traducir = crearTraductorPortal();
  assert.throws(() => traducir("texto_improvisado"), /desconocida/);
});
