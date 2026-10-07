import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { leerSalida, leerArchivo, MAXIMO_BYTES } from './modelo.js';
import { crearTextos } from '../../../../comun/textos.js';

const completa = await readFile(new URL('./testdata/preparacion-completa.json', import.meta.url));
const incompleta = await readFile(new URL('./testdata/preparacion-incompleta.json', import.meta.url));
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto));

test('abre salidas reales del CLI, incluidas ausencias que siguen pendientes, y conserva bytes', async () => {
  for (const bytes of [completa, incompleta]) {
    const dto = leerSalida(bytes); assert.equal(dto.preparacion.estado, 'pendiente');
    const file = new File([bytes], 'salida.json'); const carga = await leerArchivo(file);
    assert.deepEqual(carga.bytes, new Uint8Array(bytes)); assert.deepEqual(carga.dto, dto);
    assert.ok(dto.preparacion.pendientes.some(p => p.campo === 'firma_y_custodia'));
  }
  assert.equal(leerSalida(incompleta).preparacion.material_propuesto.contenido.plazos, null);
});

test('rechaza aprobación simulada, pendientes suprimidos, mensajes cruzados y campos ajenos', () => {
  for (const modificar of [
    dto => { dto.preparacion.estado = 'aprobado'; },
    dto => { dto.preparacion.pendientes = []; dto.mensajes = []; },
    dto => { dto.preparacion.pendientes = dto.preparacion.pendientes.filter(p => p.campo !== 'firma_y_custodia'); dto.mensajes = dto.mensajes.filter(p => p.campo !== 'firma_y_custodia'); },
    dto => { dto.mensajes[0].codigo = 'circuito_pendiente'; },
    dto => { dto.preparacion.material_propuesto.contenido.aprobado = true; },
  ]) { const dto = JSON.parse(completa); modificar(dto); assert.throws(() => leerSalida(codificar(dto)), /formato/u); }
});

test('limita archivos, rechaza UTF-8 malformado, claves repetidas y contaminación de prototipos', async () => {
  for (const bytes of [new Uint8Array(), new Uint8Array(MAXIMO_BYTES + 1)]) assert.throws(() => leerSalida(bytes), /tamano/u);
  for (const texto of ['{"preparacion":{},"preparacion":{}}', '{"__proto__":{"contaminado":true}}', '[', completa.toString().replace('"estado": "pendiente"', '"estado": "pendiente", "est\\u0061do": "aprobado"')]) assert.throws(() => leerSalida(new TextEncoder().encode(texto)), /formato/u);
  assert.throws(() => leerSalida(new Uint8Array([0xc3, 0x28])), /formato/u);
  let leido = false; await assert.rejects(leerArchivo({ size: MAXIMO_BYTES + 1, arrayBuffer() { leido = true; } }), /tamano/u); assert.equal(leido, false);
  assert.equal(Object.prototype.contaminado, undefined);
});

test('las dos traducciones comunes cubren el contrato y no cambian el material del archivo', async () => {
  const es = JSON.parse(await readFile(new URL('../../../../textos/es/seleccion-bases-preparacion.json', import.meta.url)));
  const en = JSON.parse(await readFile(new URL('../../../../textos/en/seleccion-bases-preparacion.json', import.meta.url)));
  const claves = o => Object.entries(o).flatMap(([k, v]) => typeof v === 'object' ? claves(v).map(x => `${k}.${x}`) : [k]);
  assert.deepEqual(claves(en).sort(), claves(es).sort());
  for (const [idioma, propio, localizacion] of [['es', es, 'es-ES'], ['en', en, 'en-GB']]) {
    const textos = crearTextos({ modulo: 'seleccion-bases-preparacion', idioma, localizacion, respaldo: es, propio });
    assert.deepEqual(textos.faltantes, []);
    for (const bytes of [completa, incompleta]) for (const p of leerSalida(bytes).preparacion.pendientes) {
      assert.ok(textos.traducir(`campos.${p.campo}`)); assert.ok(textos.traducir(`motivos.${p.codigo}`));
    }
    assert.ok(textos.plural('recuento_pendientes', 14));
  }
});
