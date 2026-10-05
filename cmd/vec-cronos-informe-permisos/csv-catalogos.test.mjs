import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const catalogos = await Promise.all(['es', 'en'].map(async idioma => JSON.parse(await readFile(new URL(`../../web/static/textos/${idioma}/cronos-informe-permisos-csv.json`, import.meta.url), 'utf8'))));

test('los catálogos CSV de permisos usan las mismas claves y expresan las horas en minutos', () => {
  const [es, en] = catalogos;
  for (const grupo of ['cabeceras', 'unidades', 'computos', 'estados']) {
    assert.deepEqual(Object.keys(es[grupo]).sort(), Object.keys(en[grupo]).sort(), grupo);
  }
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  for (const c of catalogos) {
    assert.equal(c.esquema, 'cronos-permisos-csv-v1');
    assert.equal(c.version, '1');
    assert.ok(c.unidades.hora === 'Minutos' || c.unidades.hora === 'Minutes');
    assert.ok(c.sintetico.length > 0);
  }
});
