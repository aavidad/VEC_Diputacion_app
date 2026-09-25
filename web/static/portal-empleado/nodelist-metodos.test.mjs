import assert from "node:assert/strict";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import test from "node:test";

// En el navegador querySelectorAll devuelve un NodeList, que no tiene find,
// filter, map ni concat. El DOM falso de las pruebas devuelve arrays y lo
// oculta, así que se vigila el patrón directamente en el código servido.
const RAIZ = new URL("..", import.meta.url).pathname;
const PATRON = /querySelectorAll\([^)]*\)\s*\.\s*(find|filter|map|concat|some|every|reduce|includes|indexOf|flatMap)\s*\(/u;

function ficheros(dir) {
  return readdirSync(dir).flatMap((nombre) => {
    const ruta = join(dir, nombre);
    if (statSync(ruta).isDirectory()) return nombre === "vendor" ? [] : ficheros(ruta);
    return /\.(m?js)$/u.test(nombre) && !/\.test\.m?js$|test-helper/u.test(nombre) ? [ruta] : [];
  });
}

test("ningún fichero servido usa métodos de array sobre un NodeList", () => {
  const infractores = ficheros(RAIZ).flatMap((ruta) =>
    readFileSync(ruta, "utf8").split("\n")
      .map((linea, i) => (PATRON.test(linea) ? `${ruta.slice(RAIZ.length)}:${i + 1}` : null))
      .filter(Boolean));
  assert.deepEqual(infractores, []);
});
