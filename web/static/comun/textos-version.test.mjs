import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import test from "node:test";
import { VERSION_TEXTOS } from "./textos.js";

// Los catálogos se leen con `?huella=VERSION_TEXTOS` y el navegador los guarda un
// año: cualquier cambio en `textos/<idioma>/*.json` exige una huella nueva.
test("VERSION_TEXTOS es la huella de todos los catálogos de textos", async () => {
  const raiz = new URL("../textos/", import.meta.url);
  const huella = createHash("sha256");
  const idiomas = (await readdir(raiz, { withFileTypes: true })).filter((e) => e.isDirectory()).map((e) => e.name).sort();
  for (const idioma of idiomas) {
    for (const nombre of (await readdir(new URL(`${idioma}/`, raiz))).filter((n) => n.endsWith(".json")).sort()) {
      huella.update(`${idioma}/${nombre}\0`).update(await readFile(new URL(`${idioma}/${nombre}`, raiz))).update("\0");
    }
  }
  const esperada = huella.digest("hex").slice(0, 16);
  assert.equal(VERSION_TEXTOS, esperada, `cambie VERSION_TEXTOS en web/static/comun/textos.js a "${esperada}"`);
});
