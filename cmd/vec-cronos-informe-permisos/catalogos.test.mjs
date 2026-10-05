import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const catalogos = await Promise.all(['es', 'en'].map(async idioma => JSON.parse(await readFile(new URL(`../../web/static/textos/${idioma}/cronos-informe-permisos.json`, import.meta.url), 'utf8'))));
function hojas(objeto, prefijo = '') {
  return Object.entries(objeto).flatMap(([clave, valor]) => typeof valor === 'object' && valor !== null ? hojas(valor, `${prefijo}${clave}.`) : [[`${prefijo}${clave}`, valor]]);
}
test('los dos catálogos tienen el mismo esquema y todas las hojas son textos', () => {
  const listas = catalogos.map(c => hojas(c));
  assert.deepEqual(listas[0].map(([clave]) => clave).sort(), listas[1].map(([clave]) => clave).sort());
  for (const lista of listas) for (const [clave, valor] of lista) {
    assert.equal(typeof valor, 'string', clave);
    assert.ok(valor.trim().length > 0, clave);
  }
  for (const catalogo of catalogos) assert.equal(catalogo.version, '2');
});
test('las traducciones conservan los marcadores y no enumeran exclusiones', () => {
  const mapas = catalogos.map(c => new Map(hojas(c)));
  for (const [clave, valor] of mapas[0]) {
    const marcadores = texto => [...texto.matchAll(/\{\{([^}]+)\}\}/g)].map(m => m[1]).sort();
    assert.deepEqual(marcadores(valor), marcadores(mapas[1].get(clave)), clave);
  }
  for (const c of catalogos) {
    assert.equal(c.fila.includes('{{campos}}'), true);
    for (const plantilla of Object.values(c.campos)) assert.equal(plantilla.includes('{{valor}}'), true);
    assert.equal(/excluid|excluded|subtotal|total:/i.test(c.alcance + c.fila + c.limite), false);
  }
});
