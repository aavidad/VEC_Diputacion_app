import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { leerArchivo, leerSalida, leerTextosMotivos, rutaTextosMotivos, MAXIMO_BYTES } from './modelo.js';
import { pintarLista } from './vista.js';
import { cargarTextos } from '../../../../comun/textos.js';

// Contrato conservado por el CLI: el visor consume exactamente esa salida.
const bytes = new Uint8Array(await readFile(new URL('../../../../../../cmd/vec-selectivos-preparar-admision/testdata/lista-resultado.json', import.meta.url)));
const base = () => JSON.parse(new TextDecoder().decode(bytes));
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto));
const motivosDe = async idioma => leerTextosMotivos(JSON.parse(await readFile(new URL(rutaTextosMotivos(idioma, 'seleccion-admision-ejemplo'), import.meta.url))));

test('la salida del CLI se abre como borrador sin aprobar ni publicar', async () => {
  const dto = leerSalida(bytes);
  assert.equal(dto.aprobada, false); assert.equal(dto.publicada, false); assert.equal(dto.estado, 'borrador_pendiente_aprobacion');
  const r = await leerArchivo({ size: bytes.length, arrayBuffer: async () => bytes.buffer });
  assert.deepEqual(r.bytes, bytes);
});
test('incoherencias, actos falsos y datos de más se rechazan', () => {
  for (const mutar of [
    d => { d.aprobada = true; }, d => { d.publicada = true; }, d => { d.estado = 'aprobada'; },
    d => { d.excluidas[0].nombre = 'Antonio Reyes Álvarez'; }, d => { d.resumen.admitidas += 1; },
    d => { d.excluidas[0].subsanable = !d.excluidas[0].subsanable; }, d => { d.excluidas[0].motivos = []; },
    d => { d.excluidas[1].motivos.push(d.excluidas[1].motivos[0]); }, d => { d.admitidas.push(d.excluidas[0]); },
    d => { d.catalogo.paquete_ejemplo = false; }, d => { d.pendientes = d.pendientes.slice(1); },
    d => { d.plazo_subsanacion.unidad = 'horas'; }, d => { d.vencimiento_subsanacion = '2026-11-02'; },
    d => { d.esquema = 'vec.seleccion.lista-admision-definitiva.v1'; },
  ]) { const d = base(); mutar(d); assert.throws(() => leerSalida(codificar(d)), /formato/); }
  for (const s of ['{"revision":1,"revision":2}', '{"__proto__":{}}']) assert.throws(() => leerSalida(new TextEncoder().encode(s)), /formato/);
});
test('tamaño comprobado antes de leer el archivo', async () => {
  for (const size of [0, MAXIMO_BYTES + 1]) await assert.rejects(leerArchivo({ size, arrayBuffer() { throw Error('no_leer'); } }), /tamano/);
});
test('textos de motivos: ruta cerrada y archivo válido o nada', async () => {
  assert.equal(rutaTextosMotivos('es', '../x'), null); assert.equal(rutaTextosMotivos('ES', 'a'), null);
  assert.equal(leerTextosMotivos({ a: 1 }).size, 0); assert.equal(leerTextosMotivos([]).size, 0);
  for (const idioma of ['es', 'en']) {
    const motivos = await motivosDe(idioma);
    for (const e of base().excluidas) for (const m of e.motivos) assert.ok(motivos.get(m.codigo), `${idioma}: ${m.codigo}`);
  }
});
class Nodo {
  constructor(d, tag) { this.ownerDocument = d; this.tagName = tag; this.children = []; this.dataset = {}; this._texto = ''; }
  append(...xs) { this.children.push(...xs); }
  replaceChildren(...xs) { this.children = xs; }
  set textContent(v) { this._texto = String(v); this.children = []; }
  get textContent() { return this._texto + this.children.map(c => (typeof c === 'string' ? c : c.textContent)).join(' '); }
  setAttribute(k, v) { this[k] = v; }
}
const elementos = e => [e, ...e.children.flatMap(elementos)];
test('vista: motivos traducidos, subsanación en texto, tablas accesibles y sin acto de aprobación', async () => {
  const claves = v => Object.entries(v).flatMap(([k, x]) => typeof x === 'object' ? claves(x).map(c => `${k}.${c}`) : [k]).sort();
  const es = JSON.parse(await readFile(new URL('../../../../textos/es/selectivos-lista-admision-visor.json', import.meta.url)));
  const en = JSON.parse(await readFile(new URL('../../../../textos/en/selectivos-lista-admision-visor.json', import.meta.url)));
  assert.deepEqual(claves(es), claves(en));
  for (const idioma of ['es', 'en']) {
    const d = { createElement: tag => new Nodo(d, tag), createDocumentFragment: () => new Nodo(d, '#fragmento') };
    const raiz = d.createElement('div'); const textos = await cargarTextos('selectivos-lista-admision-visor', { idioma });
    const motivos = await motivosDe(idioma);
    pintarLista({ raiz, dto: leerSalida(bytes), textos, motivos: new Map([...motivos, ['solicitud_fuera_de_plazo', '<img src=x onerror=alert(1)>']]) });
    const texto = raiz.textContent;
    assert.ok(texto.includes(motivos.get('tasa_no_justificada')) && texto.includes('<img src=x onerror=alert(1)>'));
    assert.ok(texto.includes(textos.traducir('no_puede_subsanar')) && texto.includes(textos.traducir('puede_subsanar')));
    assert.ok(texto.includes(textos.traducir('estado_borrador')));
    assert.ok(texto.includes(textos.plural('unidades.dias_habiles', 10)));
    assert.ok(!elementos(raiz).some(n => n.tagName === 'img' || n.tagName === 'button'));
    assert.ok(elementos(raiz).filter(n => n.tagName === 'td').every(n => n.dataset.etiqueta));
    assert.ok(elementos(raiz).filter(n => n.tagName === 'th').every(n => ['row', 'col'].includes(n.scope)));
    for (const p of base().pendientes) assert.ok(textos.traducir(`pendientes.${p.slice(p.lastIndexOf('.') + 1)}`));
    assert.deepEqual(textos.faltantes, []);
    const sinTextos = d.createElement('div'); pintarLista({ raiz: sinTextos, dto: base(), textos });
    assert.ok(sinTextos.textContent.includes('tasa_no_justificada'));
  }
});
