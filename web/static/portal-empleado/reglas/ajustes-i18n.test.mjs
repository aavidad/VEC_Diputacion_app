import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { cargarTextosAjustes, existeClaveAjustes, reintentarTextosAjustes, traducirAjustes } from "./ajustes-i18n.js";

test("el catálogo de plazos carga solo los textos de esta pantalla y se puede recuperar", async () => {
  const castellano = JSON.parse(readFileSync(new URL("../../textos/es/reglas-plazos.json", import.meta.url), "utf8"));
  const ingles = JSON.parse(readFileSync(new URL("../../textos/en/reglas-plazos.json", import.meta.url), "utf8"));
  assert.deepEqual(Object.keys(castellano.general).sort(), Object.keys(ingles.general).sort());
  await cargarTextosAjustes();
  assert.equal(existeClaveAjustes("ajustesGuardadoSinLectura"), true);
  assert.ok(traducirAjustes("ajustesTitulo"));
  await reintentarTextosAjustes();
  assert.ok(traducirAjustes("ajustesReintentarIdioma"));
});
