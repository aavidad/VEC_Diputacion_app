import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { runInNewContext } from 'node:vm';

// Ejecuta la función real de montaje, sin cargar transportes ni una página web.
const fuente = await readFile(new URL('./montaje.js', import.meta.url), 'utf8');
const pintar = fuente.slice(fuente.indexOf('function pintar() {'), fuente.indexOf('\nconst editor ='));
function repintar({ caja, tipo = 'text', externo = false, perdidaDuranteComparacion = false, recortado = false }) {
  const llamadas = [];
  const body = {}, fuera = {}, anterior = { dataset: tipo === 'text' ? { concursoRuta: 'coeficiente' } : {}, type: tipo };
  const documento = { body, activeElement: externo ? fuera : perdidaDuranteComparacion ? body : anterior };
  const siguiente = { dataset: { ...anterior.dataset }, type: tipo,
    getBoundingClientRect: () => caja,
    focus(opciones) { llamadas.push(opciones); documento.activeElement = this; },
    contains: () => false,
  };
  documento.elementFromPoint = () => recortado ? {} : siguiente;
  const raiz = {
    contains: (nodo) => nodo === anterior,
    set innerHTML(valor) { if (documento.activeElement === anterior) documento.activeElement = body; },
    querySelector: () => siguiente,
    querySelectorAll: (selector) => selector.includes(' input,') ? [siguiente] : [],
  };
  const estado = { trabajando: false, invalidos: {} };
  const contexto = { document: documento, window: { innerWidth: 720, innerHeight: 450 }, raiz,
    panel: 'concursos', focoComparacion: perdidaDuranteComparacion || externo,
    t: (clave) => clave, editor: { estado: () => estado },
    concursos: { estado: () => estado, opciones: () => ({}), alPintar() {} },
    renderizarConcursos: () => '', renderizarPanelesBaremo: () => '', filtrar() {},
    textos: {}, ejemplos: [], filtro: '', error: '', ayuda: false,
  };
  runInNewContext(`${pintar}\npintar();`, contexto);
  return { llamadas, documento, siguiente };
}
const visible = { left: 30, top: 200, right: 680, bottom: 240 };
const fuera = { left: 30, top: 743, right: 680, bottom: 783 };

test('un campo visible conserva el desplazamiento al restaurar el foco', () => {
  const r = repintar({ caja: visible });
  assert.equal(r.llamadas.length, 1); assert.equal(r.llamadas[0].preventScroll, true);
  assert.equal(r.documento.activeElement, r.siguiente);
});

test('la acción tras comparar recupera el foco una sola vez y permite desplazamiento nativo', () => {
  const r = repintar({ caja: fuera, tipo: 'submit', perdidaDuranteComparacion: true });
  assert.equal(r.llamadas.length, 1); assert.equal(r.llamadas[0].preventScroll, false);
});

test('un control desplazado por el resultado entra en la vista aunque conserve su identidad', () => {
  const r = repintar({ caja: fuera, tipo: 'submit' });
  assert.equal(r.llamadas.length, 1); assert.equal(r.llamadas[0].preventScroll, false);
});

test('el repintado no retoma el foco cuando la persona lo ha movido fuera del montaje', () => {
  const r = repintar({ caja: fuera, externo: true });
  assert.equal(r.llamadas.length, 0);
});

test('el recorte por un contenedor permite el desplazamiento aunque la caja quepa en el viewport', () => {
  const r = repintar({ caja: visible, recortado: true });
  assert.equal(r.llamadas.length, 1); assert.equal(r.llamadas[0].preventScroll, false);
});
