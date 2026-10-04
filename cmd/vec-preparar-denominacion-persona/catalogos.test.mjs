import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import assert from 'node:assert/strict';

test('catálogos de la CLI mantienen claves y mensajes completos',async()=>{
 const es=JSON.parse(await readFile(new URL('../../web/static/textos/es/persona-denominacion-preparar.json',import.meta.url),'utf8'));
 const en=JSON.parse(await readFile(new URL('../../web/static/textos/en/persona-denominacion-preparar.json',import.meta.url),'utf8'));
 assert.equal(es.idioma,'es');assert.equal(en.idioma,'en');
 assert.deepEqual(Object.keys(es.mensajes).sort(),Object.keys(en.mensajes).sort());
 for(const d of [es,en]){assert.equal(Object.keys(d.mensajes).length,8);for(const m of Object.values(d.mensajes)){assert.equal(typeof m,'string');assert.ok(m.trim());}}
});
