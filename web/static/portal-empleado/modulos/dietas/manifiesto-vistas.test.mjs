import assert from "node:assert/strict";
import { readFile, readdir, stat } from "node:fs/promises";
import test from "node:test";

const raizWeb = new URL("../../../../", import.meta.url);
const prefijo = "static/portal-empleado/modulos/dietas/";

async function entradas(nombre) {
  return new Set((await readFile(new URL(nombre, raizWeb), "utf8"))
    .split(/\r?\n/u).map((linea) => linea.trim()).filter(Boolean));
}

for (const nombre of ["interno.manifest", "produccion.manifest"]) {
  test(`${nombre} incluye las vistas y los textos de Dietas que se sirven`, async () => {
    const manifiesto = await entradas(nombre);
    for (const carpeta of ["", "catalogo/", "informes/"]) {
      const archivos = await readdir(new URL(`${prefijo}${carpeta}`, raizWeb));
      for (const archivo of archivos.filter((valor) => /^vista(?:-[\w-]+)?\.js$/u.test(valor))) {
        const ruta = `${prefijo}${carpeta}${archivo}`;
        assert.ok(manifiesto.has(ruta), `${nombre}: falta ${ruta}`);
      }
    }
    for (const idioma of ["es", "en"]) {
      for (const catalogo of ["dietas-catalogo", "dietas-informes"]) {
        const ruta = `static/textos/${idioma}/${catalogo}.json`;
        assert.ok(manifiesto.has(ruta), `${nombre}: falta ${ruta}`);
      }
    }
    for (const carpeta of ["catalogo/", "informes/"]) {
      assert.ok(!manifiesto.has(`${prefijo}${carpeta}demo.js`));
      assert.ok(!manifiesto.has(`${prefijo}${carpeta}index.html`));
    }
  });
}

test("las páginas sin lector nominal no exponen datos de demostración", async () => {
  for (const carpeta of ["catalogo", "informes"]) {
    for (const archivo of ["index.html", "demo.js"]) {
      await assert.rejects(stat(new URL(`${prefijo}${carpeta}/${archivo}`, raizWeb)),
        { code: "ENOENT" });
    }
  }
});
