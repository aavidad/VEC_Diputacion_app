import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { catalogoTarifasDietasValido, importeACentimos } from "./vista.js";

const leer = async (ruta) => JSON.parse(await readFile(new URL(ruta, import.meta.url), "utf8"));

test("el ejemplo refleja las reglas existentes y nunca se clasifica como tarifa confirmada", async () => {
  const catalogo = await leer("../../../../../../data/demo/dietas/catalogo-rrhh.json");
  const original = await leer("../../../../../../data/catalogos/dietas/liquidacion-ejemplo-v1.json");
  assert.equal(catalogoTarifasDietasValido(catalogo), true);
  assert.equal(catalogo.estado, "ejemplo");
  assert.equal(catalogo.version, original.version);
  assert.equal(catalogo.version_tarifa_ref, original.version_tarifa_ref);
  assert.deepEqual(catalogo.fuentes, original.fuentes);
  assert.deepEqual(catalogo.tarifas.map((tarifa) => [tarifa.id, tarifa.importe_centimos]),
    original.reglas.map((regla) => [regla.referencia, regla.tope_centimos || regla.centimos_por_km]));
  assert.equal(catalogo.historia.every((entrada) => entrada.estado === "ejemplo"), true);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, estado: "aprobado" }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, fuentes: ["javascript:alert(1)"] }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, fuentes: ["https://user:secret@example.org/"] }), false);
  assert.equal(catalogoTarifasDietasValido({ ...catalogo, fuentes: ["https://example.org/norma"] }), true);
});

test("el importe decimal se convierte sin redondeo ni separador de miles", () => {
  assert.equal(importeACentimos("37,40", "es-ES"), 3740);
  assert.equal(importeACentimos("0,26", "es-ES"), 26);
  assert.equal(importeACentimos("37.40", "en-GB"), 3740);
  assert.equal(importeACentimos("1,234.50", "en-GB"), null);
  assert.equal(importeACentimos("1,234", "es-ES"), null);
  assert.equal(importeACentimos("1,005", "es-ES"), null);
  assert.equal(importeACentimos("-1", "es-ES"), null);
  assert.equal(importeACentimos("999999999,99", "es-ES"), 99999999999);
});

test("las dos traducciones cubren las mismas claves de la vista", async () => {
  const es = (await leer("../../../../textos/es/dietas-catalogo.json")).catalogo;
  const en = (await leer("../../../../textos/en/dietas-catalogo.json")).catalogo;
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  assert.equal(Object.values(es).every((valor) => typeof valor === "string" && valor.length > 0), true);
  assert.equal(Object.values(en).every((valor) => typeof valor === "string" && valor.length > 0), true);
});
