import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { MENSAJES_PORTAL } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

// Toda clave data-i18n-portal del HTML del portal debe existir en el catálogo:
// una clave ausente hace que el portal deje de arrancar.
test("index.html solo usa claves del catálogo del portal", () => {
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const claves = [...html.matchAll(/data-i18n-portal(?:-[a-z-]+)?="([^"]+)"/g)].map((m) => m[1]);
  assert.ok(claves.length > 0);
  const ausentes = [...new Set(claves)].filter((c) => !(c in MENSAJES_PORTAL));
  assert.deepEqual(ausentes, []);
});
