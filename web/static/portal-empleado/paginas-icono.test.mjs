import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import test from "node:test";

// Sin <link rel="icon"> el navegador pide /favicon.ico, que no existe: un 404
// en cada carga. Toda página del portal declara su icono.
async function paginas(directorio) {
  const entradas = await readdir(directorio, { withFileTypes: true });
  const anidadas = await Promise.all(entradas.map((entrada) => {
    const ruta = new URL(entrada.name + (entrada.isDirectory() ? "/" : ""), directorio);
    if (entrada.isDirectory()) return entrada.name === "testdata" ? [] : paginas(ruta);
    return entrada.name.endsWith(".html") ? [ruta] : [];
  }));
  return anidadas.flat();
}

test("cada página del portal declara un icono y no provoca el 404 de /favicon.ico", async () => {
  const lista = await paginas(new URL("./", import.meta.url));
  assert.ok(lista.length > 10);
  for (const ruta of lista) {
    const html = await readFile(ruta, "utf8");
    assert.match(html, /<link\s[^>]*rel="icon"/u, ruta.pathname);
  }
});
