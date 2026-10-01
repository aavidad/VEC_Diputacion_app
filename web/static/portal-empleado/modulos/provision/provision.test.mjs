import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { crearEstado, cambiarPreferencias, peticionSimulacion, validarResultado, formatearPuntos, actualizarConfiguracion } from './modelo.js';
import { crearClienteProvisionLocal } from './cliente-local.js';
import { montarModuloProvision } from './montaje.js';
import { crearTextos } from '../../../comun/textos.js';
const es = JSON.parse(await readFile(new URL('../../../textos/es/provision.json', import.meta.url), 'utf8'));
const en = JSON.parse(await readFile(new URL('../../../textos/en/provision.json', import.meta.url), 'utf8'));
const textos = crearTextos({ modulo: 'provision', idioma: 'es', localizacion: 'es-ES', respaldo: es });
const preparacion = () => ({ ejemplo_ref: 'ejemplo:1', proceso: { referencia: 'proceso:1', version: '1', configuracion: { version: 'v1', fecha_corte: '2025-01-01', ventana_desde: '2020-01-01', reglas: [] }, puestos: [{ referencia: 'puesto:1', rpt_ref: 'rpt:1', rpt_version: 'v1', nivel: 22 }, { referencia: 'puesto:2', rpt_ref: 'rpt:2', rpt_version: 'v1', nivel: 24 }] }, preferencias: [{ orden: 1, puesto_ref: 'puesto:1' }, { orden: 2, puesto_ref: 'puesto:2' }] });
const resultado = estado => ({ alcance: 'simulacion', estado: 'borrador', proceso: { referencia: 'proceso:1', version: '1' }, solicitud: { version_reglas: 'v1' }, valoraciones: estado.preferencias.map((puesto_ref, i) => ({ puesto_ref, orden: i + 1, requisitos_estado: 'pendiente', requisitos: [], resultado: { puesto_ref, completo: false, total: null, desglose: [{ familia: 'grado', estado: 'pendiente_dato', resultado: '0' }] } })) });
function dom() {
  class N {
    constructor(tag) { this.tagName = tag; this.ownerDocument = d; this.children = []; this.dataset = {}; this.atributos = {}; this.listeners = {}; this.nodeType = 1; }
    append(...ns) { this.children.push(...ns); }
    replaceChildren(...ns) { this.children = ns; }
    get firstChild() { return this.children[0]; }
    setAttribute(k, v) { this.atributos[k] = v; }
    addEventListener(k, f) { this.listeners[k] = f; }
    querySelectorAll() { return this.children.flatMap(n => [n, ...n.querySelectorAll()]).filter(n => n.dataset.foco); }
    focus() { d.activeElement = this; }
  }
  const d = { createElement: tag => new N(tag) }; return new N('main');
}
const nodos = n => [n, ...n.children.flatMap(nodos)];
const click = (raiz, clave) => nodos(raiz).find(n => n.dataset.foco === clave).listeners.click();
test('preferencias se reordenan sin duplicar ni cambiar RPT; invalida valoración', () => {
  const entrada = preparacion(); let estado = crearEstado(entrada); estado.resultado = resultado(estado);
  estado = cambiarPreferencias(estado, 'puesto:1', 'abajo'); assert.deepEqual(estado.preferencias, ['puesto:2', 'puesto:1']); assert.equal(estado.resultado, null);
  assert.equal(cambiarPreferencias(estado, 'puesto:2', 'seleccionar'), estado);
  assert.deepEqual(entrada.preferencias.map(p => p.puesto_ref), ['puesto:1', 'puesto:2']);
  assert.deepEqual(Object.keys(peticionSimulacion(estado)), ['ejemplo_ref', 'configuracion', 'preferencias']);
});
test('total pendiente conserva null y rechaza cero fabricado y respuesta ajena', () => {
  const estado = crearEstado(preparacion()); const r = resultado(estado); assert.equal(validarResultado(r, estado).valoraciones[0].resultado.total, null);
  r.valoraciones[0].resultado.total = '0'; assert.throws(() => validarResultado(r, estado)); r.valoraciones[0].resultado.total = null; r.proceso.referencia = 'otro'; assert.throws(() => validarResultado(r, estado));
});
test('puntos exactos, traducciones completas y cambios configuración sin puntuar', () => {
  assert.equal(formatearPuntos('9007199254740993123456', 'es-ES'), '9.007.199.254.740.993,123456');
  const estado = crearEstado(preparacion()); estado.proceso.configuracion.reglas = [{ coeficiente: '0', maximo: '1000000' }];
  const cambio = actualizarConfiguracion(estado, { regla: 0, campo: 'coeficiente', valor: '1,000001' }); assert.equal(cambio.proceso.configuracion.reglas[0].coeficiente, '1000001'); assert.throws(() => actualizarConfiguracion(estado, { regla: 0, campo: 'coeficiente', valor: '1e9' }));
  const traducidos = crearTextos({ modulo: 'provision', idioma: 'en', localizacion: 'en-GB', respaldo: es, propio: en }); assert.deepEqual(traducidos.faltantes, []);
});
test('cliente transmite solo configuración y preferencias; denegación comprensible', async () => {
  const llamadas = []; const cliente = crearClienteProvisionLocal({ fetchImpl: async (url, opciones) => { llamadas.push({ url, opciones }); return { ok: true, text: async () => '{}' }; } });
  await cliente.simular({ ...peticionSimulacion(crearEstado(preparacion())), empleado_ref: 'NO', entrada: { acreditado: true } });
  const body = JSON.parse(llamadas[0].opciones.body); assert.equal(body.empleado_ref, undefined); assert.equal(body.entrada, undefined); assert.equal(llamadas[0].opciones.credentials, 'omit');
  const denegado = crearClienteProvisionLocal({ fetchImpl: async () => ({ ok: false, status: 403 }) }); await assert.rejects(denegado.listar(), e => e.codigo === 'denegado');
});
test('vista mantiene controles jurídicos bloqueados; cancela respuesta tardía y recupera foco', async () => {
  const raiz = dom(); let resolver; let signal; const cliente = { simular: (_p, opts) => { signal = opts.signal; return new Promise(r => resolver = r); } };
  const m = await montarModuloProvision({ raiz, cliente, preparacion: preparacion(), textos });
  click(raiz, 'vista-personal'); click(raiz, 'abajo-puesto:1'); assert.equal(raiz.ownerDocument.activeElement.dataset.foco, 'arriba-puesto:1');
  assert.equal(nodos(raiz).find(n => n.dataset.foco === 'presentar').disabled, true);
  click(raiz, 'vista-valoracion'); const promesa = click(raiz, 'simular'); click(raiz, 'vista-puestos'); assert.equal(signal.aborted, true);
  resolver(resultado(m.obtenerEstado())); await promesa; assert.equal(m.obtenerEstado().resultado, null);
  m.desmontar(); assert.equal(raiz.children.length, 0);
});
test('render ES/EN sin HTML inyectado y estados pendientes distintos de puntuación cero', async () => {
  for (const [idioma, propio, localizacion] of [['es', es, 'es-ES'], ['en', en, 'en-GB']]) {
    const raiz = dom(); const tx = crearTextos({ modulo: 'provision', idioma, localizacion, respaldo: es, propio });
    const m = await montarModuloProvision({ raiz, textos: tx, preparacion: preparacion(), proyeccion: { puestos: { 'puesto:1': { denominacion: '<img onerror=alert(1)>' } } }, cliente: { simular: async () => resultado(m.obtenerEstado()) } });
    click(raiz, 'vista-valoracion'); await click(raiz, 'simular'); assert.equal(nodos(raiz).some(n => n.tagName === 'img'), false); assert.equal(nodos(raiz).some(n => n.textContent === tx.traducir('estados.sin_dato')), true);
  }
});
