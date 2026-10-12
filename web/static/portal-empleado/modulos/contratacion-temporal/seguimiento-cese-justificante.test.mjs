import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// Las causas de cese indican su justificante con una clave interna del
// catálogo de reglas; la pantalla debe mostrar su nombre, nunca la clave.
test("cada justificante del catálogo de causas tiene nombre en español e inglés", async () => {
  const reglas = await readFile(new URL("../../../../../data/demo/reglas/ct_causas_cese.demo.json", import.meta.url), "utf8");
  const tipos = [...new Set([...reglas.matchAll(/"justificante":\s*"([a-z_]+)"/gu)].map((m) => m[1]))];
  assert.ok(tipos.length > 0);
  for (const idioma of ["es", "en"]) {
    const texto = await readFile(new URL(`../../../textos/${idioma}/contratacion-temporal-seguimiento-cese.json`, import.meta.url), "utf8");
    for (const tipo of tipos) assert.match(texto, new RegExp(`"cese_justificante_${tipo}":\\s*"[^"_]+"`, "u"), `${idioma}: ${tipo}`);
  }
});
