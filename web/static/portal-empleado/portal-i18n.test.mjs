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

test("Bolsa interna usa catálogo común y formatos es-ES para textos y valores", async () => {
  const {
    crearTraductorBolsaInterna,
    MENSAJES_BOLSA_INTERNA_ES,
    formatearFechaPortal,
    formatearNumeroPortal,
  } = await import("./portal-i18n.js");
  const traducir = crearTraductorBolsaInterna();
  for (const clave of Object.keys(MENSAJES_BOLSA_INTERNA_ES)) {
    assert.notEqual(traducir(clave), "");
  }
  assert.equal(traducir("numero_convocatorias", { numero: "3" }), "3 convocatorias encontradas.");
  assert.equal(formatearNumeroPortal(12345), "12.345");
  assert.equal(formatearFechaPortal("2026-08-01T09:00"), "1/8/26, 9:00");
  assert.throws(() => traducir("bolsa_texto_improvisado"), /desconocida/);
});
