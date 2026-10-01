import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { montarPreparacion } from './arranque.js';

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
