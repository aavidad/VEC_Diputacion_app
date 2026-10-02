import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { montarPreparacion, textoResumen } from './arranque.js';
import { crearResumen, validarLimites } from './modelo.js';
import { cargarTextos } from '../../comun/textos.js';

const html = await readFile(new URL('./index.html', import.meta.url), 'utf8');
const { detalle } = JSON.parse(await readFile(new URL('../../../../scripts/recorridos/convoca_fixture.json', import.meta.url), 'utf8'));

// Doble DOM para el ciclo de eventos; el navegador comprueba la semántica real.
function documentoPrueba() {
  const documento = { documentElement: {}, activeElement: null };
  function nodo(atributos = '') {
    const attrs = new Map([...atributos.matchAll(/([\w-]+)="([^"]*)"/gu)].map((m) => [m[1], m[2]]));
    const eventos = new Map();
    return {
      dataset: {}, checked: false, hidden: false, value: '', textContent: '',
      setAttribute: (k, v) => attrs.set(k, v), getAttribute: (k) => attrs.get(k) ?? null,
      removeAttribute: (k) => attrs.delete(k),
      addEventListener: (k, fn) => eventos.set(k, fn),
      emitir(k) { return eventos.get(k)?.({ target: this }); },
      focus() { documento.activeElement = this; },
      click() {},
      append() {}, replaceChildren() { this.textContent = ''; }, querySelectorAll: () => [],
    };
  }
  const nodos = new Map([...html.matchAll(/<[^>]+\bid="([^"]+)"[^>]*>/gu)].map((m) => [m[1], nodo(m[0])]));
  documento.getElementById = (id) => nodos.get(id);
  documento.querySelectorAll = () => [];
  documento.createElement = () => nodo();
  return documento;
}

test('el rechazo se asocia a la casilla, conserva el foco y desaparece al corregir', async (t) => {
  const fetchOriginal = globalThis.fetch;
  t.after(() => { globalThis.fetch = fetchOriginal; });
  globalThis.fetch = async () => new Response(JSON.stringify(detalle), { headers: { 'content-type': 'application/json' } });
  const documento = documentoPrueba();
  const porId = (id) => documento.getElementById(id);
  const montaje = await montarPreparacion(documento, { href: 'https://fixture.invalid/bolsa/preparacion/?convocatoria=sintetica-2026' });
  const casilla = porId('lectura-confirmada');
  assert.match(html, /id="lectura-confirmada" required/u);
  assert.equal(casilla.getAttribute('aria-describedby'), 'error-lectura');
  assert.equal(casilla.getAttribute('aria-invalid'), 'false');
  porId('siguiente').emitir('click');
  assert.equal(casilla.getAttribute('aria-invalid'), 'true');
  assert.equal(documento.activeElement, casilla);
  assert.ok(porId('error-lectura').textContent);
  assert.equal(porId('paso-revision').hidden, false);
  casilla.checked = true; casilla.emitir('change');
  assert.equal(casilla.getAttribute('aria-invalid'), 'false');
  assert.equal(porId('error-lectura').textContent, '');
  porId('siguiente').emitir('click');
  assert.equal(porId('paso-archivos').hidden, false);
  porId('anterior').emitir('click');
  casilla.checked = false; casilla.emitir('change'); porId('siguiente').emitir('click');
  assert.equal(casilla.getAttribute('aria-invalid'), 'true');
  montaje.desmontar();
  assert.equal(casilla.getAttribute('aria-invalid'), 'false');
  assert.equal(casilla.checked, false);
  assert.equal(porId('error-lectura').textContent, '');
});


test('TXT y JSON descargan el mismo resumen mostrado: bytes, fecha y metadatos sin archivos', async (t) => {
  const fetchOriginal = globalThis.fetch;
  const crearURL = URL.createObjectURL;
  const revocarURL = URL.revokeObjectURL;
  t.after(() => { globalThis.fetch = fetchOriginal; URL.createObjectURL = crearURL; URL.revokeObjectURL = revocarURL; });
  const blobs = [];
  const revocadas = [];
  const descargas = [];
  globalThis.fetch = async () => new Response(JSON.stringify(detalle), { headers: { 'content-type': 'application/json' } });
  URL.createObjectURL = (blob) => { blobs.push(blob); return `blob:prueba-${blobs.length}`; };
  URL.revokeObjectURL = (url) => revocadas.push(url);
  const documento = documentoPrueba();
  const crearElemento = documento.createElement;
  documento.createElement = (tag) => {
    const elemento = crearElemento(tag);
    if (tag === 'a') elemento.click = () => descargas.push(elemento.download);
    return elemento;
  };
  const porId = (id) => documento.getElementById(id);
  const montaje = await montarPreparacion(documento, { href: 'https://fixture.invalid/bolsa/preparacion/?convocatoria=sintetica-2026' });
  assert.equal(porId('descargar-json').hidden, true);
  porId('descargar-json').emitir('click');
  assert.equal(blobs.length, 0);
  porId('lectura-confirmada').checked = true;
  porId('siguiente').emitir('click');
  porId('archivos-locales').files = [{ name: 'titulación.pdf', size: 23, text() { throw new Error('bytes privados'); } }];
  porId('archivos-locales').emitir('change');
  porId('siguiente').emitir('click');
  assert.equal(porId('descargar-json').hidden, false);
  porId('descargar-json').emitir('click');
  porId('descargar').emitir('click');
  porId('descargar-json').emitir('click');
  assert.equal(blobs[0].type, 'application/json;charset=utf-8');
  assert.equal(blobs[1].type, 'text/plain;charset=utf-8');
  const contenido = await blobs[0].text();
  const resumen = JSON.parse(contenido);
  const textos = await cargarTextos('convoca-preparacion', { idioma: 'es' });
  const esperado = crearResumen(detalle, { limites: validarLimites(textos.seccion('limites')), lectura: true,
    archivos: [{ nombre: 'titulación.pdf', tamano: 23 }], fecha: new Date(resumen.generada_en) });
  assert.deepEqual(resumen, esperado);
  assert.equal(resumen.esquema, 'vec.bolsa.preparacion-local.v1');
  assert.equal(resumen.estado, 'sin_presentar');
  assert.equal(contenido, await blobs[2].text());
  assert.equal(await blobs[1].text(), textoResumen(esperado, textos));
  assert.equal(blobs[0].size, Buffer.byteLength(contenido, 'utf8'));
  assert.deepEqual(resumen.archivos_locales, [{ nombre: 'titulación.pdf', tamano: 23 }]);
  assert.deepEqual(descargas, ['borrador-solicitud-sintetica-2026.json', 'borrador-solicitud-sintetica-2026.txt', 'borrador-solicitud-sintetica-2026.json']);
  porId('anterior').emitir('click');
  porId('descargar-json').emitir('click');
  assert.equal(blobs.length, 3);
  montaje.desmontar();
  assert.deepEqual(revocadas, ['blob:prueba-1', 'blob:prueba-2', 'blob:prueba-3']);
});
