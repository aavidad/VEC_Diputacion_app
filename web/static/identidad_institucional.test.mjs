import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const directorio = dirname(fileURLToPath(import.meta.url));
const rutaLogo = "/portal-empleado/assets/logo-diputacion-granada.svg";

for (const pagina of ["index.html", "bolsa/index.html", "portal-empleado/index.html"]) {
  test(`${pagina} muestra la identidad institucional oficial`, () => {
    const html = readFileSync(join(directorio, pagina), "utf8");
    assert.match(html, new RegExp(`src=["']${rutaLogo.replaceAll("/", "\\/")}["']`));
    assert.match(html, /alt=["']Diputación de Granada["']/);
  });
}

test("el logotipo institucional se distribuye como activo local", () => {
  assert.equal(
    existsSync(join(directorio, "portal-empleado/assets/logo-diputacion-granada.svg")),
    true,
  );
});
