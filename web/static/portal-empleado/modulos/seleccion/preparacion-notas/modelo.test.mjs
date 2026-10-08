import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { leerSalida, leerArchivo, MAXIMO_BYTES } from './modelo.js';
import { pintarSalida, textosRevision } from './vista.js';
import { cargarTextos } from '../../../../comun/textos.js';

const bytes = new Uint8Array(await readFile(new URL('./testdata/preparacion.json', import.meta.url)));
const base = () => JSON.parse(new TextDecoder().decode(bytes));
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto));

test('salida real conserva antecedente, nota propuesta pendiente y original para descargar', async () => {
  const dto = leerSalida(bytes);
  assert.equal(dto.antecedente.solicitudes[0].fases[0].puntos_micropuntos, 9000000);
  assert.equal(dto.propuesta.solicitudes[0].fases[0].puntos_micropuntos, 9000000);
  assert.equal(dto.propuesta.solicitudes[1].fases[0].puntos_micropuntos, null);
  assert.equal(dto.propuesta.solicitudes[1].orden, null);
  assert.deepEqual(dto.cambios.map(c => [c.solicitud_ref, c.anterior_micropuntos, c.propuesta_micropuntos]), [['solicitud_2', 7000000, null]]);
  const archivo = await leerArchivo({ size: bytes.length, arrayBuffer: async () => bytes.buffer });
  assert.deepEqual(archivo.bytes, bytes);
});
test('rechaza estado oficial, configuración incompatible y metadatos ajenos', () => {
  for (const mutar of [
    d => { d.alcance = 'oficial'; }, d => { d.aprobado = true; },
    d => { d.configuracion.version = 0; }, d => { d.configuracion.nominal = true; },
    d => { d.propuesta.bases_version += 1; }, d => { d.propuesta.solicitudes[0].nombre = 'Otra solicitud'; },
    d => { d.propuesta.solicitudes[0].referencia = 'otra'; }, d => { d.pendientes.pop(); },
    d => { d.huella_antecedente_sha256 = 'no_es_una_huella'; },
  ]) { const d = base(); mutar(d); assert.throws(() => leerSalida(codificar(d)), /formato/); }
});
test('notas y cambios están limitados a solicitudes y pruebas del mismo cálculo', () => {
  for (const mutar of [
    d => { d.notas_propuestas[0].solicitud_ref = 'otra'; }, d => { d.notas_propuestas[0].fase_ref = 'meritos'; },
    d => { d.notas_propuestas[0].puntos_micropuntos = 10000001; }, d => { d.notas_propuestas.push(d.notas_propuestas[0]); },
    d => { d.cambios[0].propuesta_micropuntos = 1000000; }, d => { d.cambios.push(d.cambios[0]); },
    d => { d.cambios[0].anterior_micropuntos = null; }, d => { d.cambios = []; },
    d => { d.cambios[0].anterior_micropuntos = 6000000; },
    d => { d.propuesta.solicitudes[0].acceso_detalle[0].estado = 'no_cumple'; },
    d => { d.propuesta.solicitudes[0].fases[0].puntos_micropuntos = 5000000; },
    d => { d.propuesta.solicitudes[1].total_micropuntos = 7000000; },
  ]) { const d = base(); mutar(d); assert.throws(() => leerSalida(codificar(d)), /formato/); }
});
test('rechaza duplicados, prototipos, UTF-8 inválido y tamaño antes de mostrar el archivo', async () => {
  for (const s of ['{"alcance":1,"alcance":2}', '{"__proto__":{}}', '{"constructor":{}}']) assert.throws(() => leerSalida(new TextEncoder().encode(s)), /formato/);
  assert.throws(() => leerSalida(new Uint8Array([255])), /formato/);
  for (const size of [0, MAXIMO_BYTES + 1]) await assert.rejects(leerArchivo({ size, arrayBuffer() { throw Error('no_leer'); } }), /tamano/);
});
test('catálogos ES/EN tienen paridad y traducen estados y actuaciones pendientes', async () => {
  const a = JSON.parse(await readFile(new URL('../../../../textos/es/seleccion.json', import.meta.url)));
  const b = JSON.parse(await readFile(new URL('../../../../textos/en/seleccion.json', import.meta.url)));
  const claves = v => Object.entries(v).flatMap(([k,x]) => typeof x === 'object' ? claves(x).map(c => `${k}.${c}`) : [k]).sort();
  assert.deepEqual(claves(a), claves(b));
  for (const idioma of ['es','en']) {
    const textos = textosRevision(await cargarTextos('seleccion', { idioma }));
    for (const p of leerSalida(bytes).pendientes) assert.ok(textos.traducir(`actuacion.${p}`));
    assert.deepEqual(textos.faltantes, []);
  }
});
class Nodo {
  constructor(d, tag) { this.ownerDocument = d; this.tagName = tag; this.children = []; this.dataset = {}; this._texto = ''; }
  append(...xs) { this.children.push(...xs); }
  replaceChildren(...xs) { this.children = xs; }
  set textContent(v) { this._texto = String(v); this.children = []; }
  get textContent() { return this._texto + this.children.map(c => c.textContent).join(' '); }
  setAttribute(k,v) { this[k] = v; }
}
const elementos = e => [e, ...e.children.flatMap(elementos)];
test('vista escapada explica antes/después, mantiene pendientes y no muestra una aprobación', async () => {
  for (const idioma of ['es','en']) {
    const d = { createElement: tag => new Nodo(d,tag) }, raiz = d.createElement('div');
    const dto = base(); dto.antecedente.solicitudes[0].nombre = '<img src=x onerror=alert(1)>';
    dto.propuesta.solicitudes[0].nombre = dto.antecedente.solicitudes[0].nombre;
    const textos = textosRevision(await cargarTextos('seleccion', { idioma }));
    pintarSalida({ raiz, dto: leerSalida(codificar(dto)), textos });
    assert.ok(raiz.textContent.includes('<img src=x onerror=alert(1)>'));
    assert.ok(!elementos(raiz).some(n => n.tagName === 'img'));
    assert.ok(raiz.textContent.includes(textos.traducir('pendiente')));
    assert.ok(raiz.textContent.includes(textos.traducir('referencias_limite')));
    assert.ok(elementos(raiz).filter(n => n.tagName === 'td').every(n => n.dataset.etiqueta));
    assert.ok(elementos(raiz).filter(n => n.tagName === 'th').every(n => ['row','col'].includes(n.scope)));
    assert.deepEqual(textos.faltantes, []);
  }
});
