import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { MENSAJES_BORRADORES_EN, MENSAJES_BORRADORES_ES } from "./portal-borradores-i18n.js";

test("borradores conserva claves y marcadores en ambos idiomas", () => {
  assert.deepEqual(Object.keys(MENSAJES_BORRADORES_EN), Object.keys(MENSAJES_BORRADORES_ES));
  const marcadores = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/g)].map((match) => match[1]).sort();
  for (const clave of Object.keys(MENSAJES_BORRADORES_ES)) {
    assert.deepEqual(marcadores(MENSAJES_BORRADORES_EN[clave]), marcadores(MENSAJES_BORRADORES_ES[clave]), clave);
  }
});

test("el traductor común usa inglés con lang=en aunque el navegador prefiera castellano", () => {
  const modulo = new URL("./portal-i18n.js", import.meta.url).href;
  const codigo = `globalThis.location = { href: "https://vec.example/portal?lang=en" }; globalThis.navigator = { languages: ["es-ES"] }; const { traducirPortal, formatearNumeroPortal } = await import(${JSON.stringify(modulo)}); console.log(JSON.stringify([traducirPortal("bl_titulo"), traducirPortal("borradores_registros_otros", { total: formatearNumeroPortal(1234) })]));`;
  const resultado = spawnSync(process.execPath, ["--input-type=module", "-e", codigo], { encoding: "utf8" });
  assert.equal(resultado.status, 0, resultado.stderr);
  assert.deepEqual(JSON.parse(resultado.stdout.trim()), ["Prepare an internal draft", "1,234 records in the authorised scope"]);
});
