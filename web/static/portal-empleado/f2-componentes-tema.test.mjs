import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const leer = (ruta) => readFile(new URL(ruta, import.meta.url), "utf8");
const [componentes, tema, inicio, portal, organizacion, peticiones] = await Promise.all([
  leer("./portal-componentes.css"),
  leer("./portal.css"),
  leer("./portal-inicio.js"),
  leer("./index.html"),
  leer("./organizacion/index.html"),
  leer("./peticiones-centro/index.html"),
]);

test("las etiquetas y el vacío del resumen usan el texto secundario del tema común", () => {
  assert.match(inicio, /class="metrica-etiqueta"/u);
  assert.match(inicio, /class="portal-rrhh-resumen-vacio"/u);
  for (const selector of [
    ".tarjeta-metrica-rrhh .metrica-etiqueta",
    ".portal-rrhh-resumen-vacio",
  ]) {
    const regla = componentes.match(new RegExp(`${selector.replaceAll(".", "\\.")}\\s*\\{([^}]*)\\}`, "u"))?.[1];
    assert.ok(regla, `falta el selector ${selector}`);
    assert.match(regla, /color:\s*var\(--portal-muted\)\s*;/u, selector);
  }
  assert.doesNotMatch(componentes, /--portal-tinta-secundaria|--color-texto-secundario/u);
});

test("el token responde al alto contraste y los consumidores cargan el CSS común", () => {
  const normal = tema.match(/:root\s*\{[^}]*--portal-muted:\s*([^;]+);/u)?.[1];
  const altoContraste = tema.match(/body\.portal-empleado-app\[data-contraste="true"\]\s*\{[^}]*--portal-muted:\s*([^;]+);/u)?.[1];
  assert.ok(normal, "falta el token de texto secundario");
  assert.ok(altoContraste, "falta su variante de alto contraste");
  assert.notEqual(normal, altoContraste);
  for (const [nombre, html] of [["portal", portal], ["organización", organizacion], ["peticiones", peticiones]]) {
    assert.match(html, /href="\/portal-empleado\/portal-componentes\.css(?:\?[^" ]+)?"/u, nombre);
  }
});
