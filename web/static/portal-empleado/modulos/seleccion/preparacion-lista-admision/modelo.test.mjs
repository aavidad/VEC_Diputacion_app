import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { leerArchivo, leerSalida, leerTextosMotivos, moduloTextosMotivos, MAXIMO_BYTES } from './modelo.js';
import { pintarLista } from './vista.js';
import { cargarTextos } from '../../../../comun/textos.js';

// Contrato conservado por el CLI: el visor consume exactamente esa salida.
const bytes = new Uint8Array(await readFile(new URL('../../../../../../cmd/vec-selectivos-preparar-admision/testdata/lista-resultado.json', import.meta.url)));
const base = () => JSON.parse(new TextDecoder().decode(bytes));
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto));
const motivosDe = async idioma => leerTextosMotivos((await cargarTextos(moduloTextosMotivos('seleccion-admision-ejemplo'), { idioma })).mensajes);

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
    d => { d.plazo_subsanacion = { unidad: 'meses', cantidad: 61 }; }, d => { delete d.catalogo.duda_ref; },
  ]) { const d = base(); mutar(d); assert.throws(() => leerSalida(codificar(d)), /formato/); }
  for (const s of ['{"revision":1,"revision":2}', '{"__proto__":{}}']) assert.throws(() => leerSalida(new TextEncoder().encode(s)), /formato/);
});
test('tamaño comprobado antes de leer el archivo', async () => {
  for (const size of [0, MAXIMO_BYTES + 1]) await assert.rejects(leerArchivo({ size, arrayBuffer() { throw Error('no_leer'); } }), /tamano/);
});
test('textos de motivos: catálogo cerrado y archivo válido o nada', async () => {
  assert.equal(moduloTextosMotivos('../x'), null); assert.equal(moduloTextosMotivos('A'), null);
  assert.equal(moduloTextosMotivos('seleccion-admision-ejemplo'), 'motivos-seleccion-admision-ejemplo');
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
    assert.ok(texto.includes(textos.traducir('motivo_no_subsanable', { motivo: '<img src=x onerror=alert(1)>' })));
    const filas = elementos(raiz).filter(n => n.tagName === 'th' && n.scope === 'row').map(n => n.textContent);
    assert.ok(filas.includes('aux-adm-2026-0012') && filas.every(f => !f.startsWith('preparacion:')));
    assert.ok(elementos(raiz).some(n => n.tagName === 'caption'));
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

// Lista definitiva: el mismo visor la reconoce por su esquema.
const bytesDef = new Uint8Array(await readFile(new URL('../../../../../../cmd/vec-selectivos-preparar-admision/testdata/lista-definitiva-resultado.json', import.meta.url)));
const baseDef = () => JSON.parse(new TextDecoder().decode(bytesDef));

test('la definitiva del CLI se abre como borrador que parte de una provisional', () => {
  const dto = leerSalida(bytesDef);
  assert.equal(dto.esquema, 'vec.seleccion.lista-admision-definitiva.v1'); assert.equal(dto.aprobada, false);
  assert.equal(dto.provisional.esquema, 'seleccion.admision.lista-provisional.v1');
  for (const mutar of [
    d => { d.publicada = true; }, d => { d.resumen.admitidas_tras_escrito = 0; }, d => { d.resumen.excluidas_sin_escrito = 1; },
    d => { d.admitidas[0].origen = 'recurso'; }, d => { d.excluidas[0].resolucion = 'estimada'; },
    d => { delete d.excluidas[0].via; }, d => { d.excluidas[0].via = 'recurso'; },
    d => { d.excluidas[0].resolucion = 'no_presentada'; d.resumen.excluidas_sin_escrito = 1; },
    d => { d.provisional.lista_ref = d.lista_ref; }, d => { d.pendientes = d.pendientes.filter(p => !p.endsWith('pie_recursos')); },
    d => { d.pendientes.push('seleccion.lista_admision.pendiente.vencimiento_al_publicar'); },
    d => { d.plazo_subsanacion = { unidad: 'dias_habiles', cantidad: 10 }; }, d => { d.excluidas[0].subsanable = true; },
    d => { d.admitidas.push({ antecedente: d.excluidas[0].antecedente, origen: 'provisional' }); },
  ]) { const d = baseDef(); mutar(d); assert.throws(() => leerSalida(codificar(d)), /formato/); }
  // Una provisional no se acepta con campos de la definitiva, ni al revés.
  const mezcla = base(); mezcla.provisional = baseDef().provisional;
  assert.throws(() => leerSalida(codificar(mezcla)), /formato/);
});
test('vista de la definitiva: origen de admitidas, resolución de excluidas y sus pendientes', async () => {
  for (const idioma of ['es', 'en']) {
    const d = { createElement: tag => new Nodo(d, tag), createDocumentFragment: () => new Nodo(d, '#fragmento') };
    const raiz = d.createElement('div'); const textos = await cargarTextos('selectivos-lista-admision-visor', { idioma });
    pintarLista({ raiz, dto: leerSalida(bytesDef), textos, motivos: await motivosDe(idioma) });
    const texto = raiz.textContent;
    for (const clave of ['tipo_definitiva', 'origenes.reclamacion', 'origenes.provisional', 'resoluciones.desestimada_subsanacion', 'kpi.tras_escrito'])
      assert.ok(texto.includes(textos.traducir(clave)), `${idioma}: ${clave}`);
    assert.ok(!texto.includes(textos.traducir('plazo')) && !texto.includes(textos.traducir('puede_subsanar')));
    for (const p of baseDef().pendientes) {
      const final = p.slice(p.lastIndexOf('.') + 1);
      assert.ok(texto.includes(textos.traducir(`pendientes_definitiva.${final}`)));
    }
    assert.ok(!texto.includes(textos.traducir('pendientes.catalogo_ejemplo')));
    // El resto de resoluciones y orígenes, con un borrador coherente mutado.
    const otra = baseDef();
    otra.excluidas[0].resolucion = 'no_presentada'; delete otra.excluidas[0].via; otra.resumen.excluidas_sin_escrito = 1;
    otra.admitidas[2].origen = 'subsanacion';
    const raiz2 = d.createElement('div'); pintarLista({ raiz: raiz2, dto: leerSalida(codificar(otra)), textos });
    for (const clave of ['resoluciones.no_presentada', 'origenes.subsanacion']) assert.ok(raiz2.textContent.includes(textos.traducir(clave)));
    const otra2 = baseDef(); otra2.excluidas[0].via = 'reclamacion';
    const raiz3 = d.createElement('div'); pintarLista({ raiz: raiz3, dto: leerSalida(codificar(otra2)), textos });
    assert.ok(raiz3.textContent.includes(textos.traducir('resoluciones.desestimada_reclamacion')));
    assert.ok(!raiz3.textContent.includes(textos.traducir('motivo_no_subsanable', { motivo: '' }).trim()));
    assert.ok(elementos(raiz).filter(n => n.tagName === 'td').every(n => n.dataset.etiqueta));
    assert.deepEqual(textos.faltantes, []);
  }
});
