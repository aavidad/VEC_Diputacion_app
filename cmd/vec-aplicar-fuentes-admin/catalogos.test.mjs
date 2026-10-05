import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

test('Los catálogos castellano e inglés cubren los mismos diagnósticos', async () => {
  const leer = async idioma => JSON.parse(await readFile(
    new URL(`../../web/static/textos/${idioma}/admin-fuentes-aplicar.json`, import.meta.url), 'utf8'));
  const es = await leer('es');
  const en = await leer('en');
  assert.deepEqual(Object.keys(es.mensajes).sort(), Object.keys(en.mensajes).sort());
  for (const catalogo of [es, en]) {
    assert.equal(Object.keys(catalogo.mensajes).length, 11);
    assert.ok(Object.values(catalogo.mensajes).every(texto => typeof texto === 'string' && texto.trim()));
  }
});
