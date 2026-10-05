import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("CSV del saldo: cabeceras y estados ES/EN conservan las mismas claves y unidades en minutos", async () => {
  const leer = async (idioma) => JSON.parse(await readFile(new URL(`../../web/static/textos/${idioma}/cronos-informe-saldo-csv.json`, import.meta.url), "utf8"));
  const [es, en] = await Promise.all([leer("es"), leer("en")]);
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  assert.deepEqual(Object.keys(es.cabeceras).sort(), Object.keys(en.cabeceras).sort());
  assert.deepEqual(Object.keys(es.estados).sort(), Object.keys(en.estados).sort());
  for (const [catalogo, unidad] of [[es, "minutos"], [en, "minutes"]]) {
    for (const clave of ["previstos_minutos", "trabajados_minutos", "saldo_minutos"]) assert.ok(catalogo.cabeceras[clave].includes(unidad));
    assert.equal(catalogo.esquema, "cronos-saldo-csv-v1");
    assert.ok(catalogo.sintetico.length > 0);
  }
});
