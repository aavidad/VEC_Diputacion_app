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
    querySelector(selector) { return nodos(this).find(n => selector.startsWith('#') ? n.id === selector.slice(1) : selector === '[data-provision-aviso]' && n.dataset.provisionAviso !== undefined) ?? null; }
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
test('calendario civil rechaza fechas inexistentes e intervalo vacío o invertido', () => {
  const estado = crearEstado(preparacion());
  for (const valor of ['2025-02-29', '2025-02-30', '2025-13-01', '']) assert.throws(() => actualizarConfiguracion(estado, { campo: 'fecha_corte', valor }));
  assert.throws(() => actualizarConfiguracion(estado, { campo: 'ventana_desde', valor: '2025-01-01' }));
  assert.throws(() => actualizarConfiguracion(estado, { campo: 'ventana_desde', valor: '2026-01-01' }));
  assert.equal(actualizarConfiguracion(estado, { campo: 'fecha_corte', valor: '2024-02-29' }).proceso.configuracion.fecha_corte, '2024-02-29');
});
test('errores de configuración conservan campo, descriptor y foco sin sustituir controles al salir', async () => {
  const raiz = dom(); const p = preparacion(); p.proceso.configuracion.reglas = [{ familia: 'grado', coeficiente: '0', maximo: '1000000' }];
  const m = await montarModuloProvision({ raiz, textos, preparacion: p, cliente: { simular: async () => ({}) } });
  const maximo = nodos(raiz).find(n => n.dataset.foco === 'maximo-0'); maximo.focus(); maximo.value = 'abc'; maximo.listeners.change();
  assert.equal(maximo.atributos['aria-invalid'], 'true'); assert.equal(maximo.atributos['aria-describedby'], 'provision-error-maximo-0');
  assert.equal(nodos(raiz).find(n => n.dataset.foco === 'maximo-0'), maximo); assert.equal(raiz.ownerDocument.activeElement, maximo);
  const fecha = nodos(raiz).find(n => n.dataset.foco === 'fecha_corte'); fecha.value = '2024-12-01'; fecha.listeners.change();
  assert.equal(m.obtenerEstado().mensaje, textos.traducir('configuracion.errores_pendientes'));
  click(raiz, 'vista-valoracion'); assert.equal(nodos(raiz).find(n => n.dataset.foco === 'simular').disabled, true);
  click(raiz, 'corregir-configuracion'); assert.equal(raiz.ownerDocument.activeElement.dataset.foco, 'maximo-0');
  const corregir = nodos(raiz).find(n => n.dataset.foco === 'maximo-0'); assert.equal(corregir.value, 'abc'); corregir.value = '0'; corregir.listeners.change();
  assert.equal(corregir.atributos['aria-invalid'], 'false'); assert.equal(Object.keys(m.obtenerEstado().invalidos).length, 0);
});
test('rechazo 400/422 distingue validación de servicio indisponible', async () => {
  for (const status of [400, 422]) {
    const cliente = crearClienteProvisionLocal({ fetchImpl: async () => ({ ok: false, status }) });
    await assert.rejects(cliente.simular({}), e => e.codigo === 'validacion');
  }
});

test('corregir la segunda fecha revalida el intervalo visible y conserva errores numéricos ajenos', async () => {
  const raiz = dom(); const p = preparacion(); p.proceso.configuracion.reglas = [{ familia: 'grado', coeficiente: '0', maximo: '1000000' }];
  let enviada; const cliente = { simular: async peticion => { enviada = peticion; return resultado(m.obtenerEstado()); } };
  const m = await montarModuloProvision({ raiz, textos, preparacion: p, cliente });
  const maximo = nodos(raiz).find(n => n.dataset.foco === 'maximo-0'); maximo.value = 'abc'; maximo.listeners.change();
  const inicio = nodos(raiz).find(n => n.dataset.foco === 'ventana_desde'); inicio.value = '2030-01-01'; inicio.listeners.change();
  assert.equal(inicio.atributos['aria-invalid'], 'true');
  const corte = nodos(raiz).find(n => n.dataset.foco === 'fecha_corte'); corte.value = '2035-01-01'; corte.listeners.change();
  assert.equal(inicio.atributos['aria-invalid'], 'false'); assert.equal(raiz.querySelector('#provision-error-ventana_desde').hidden, true);
  assert.deepEqual(Object.keys(m.obtenerEstado().invalidos), ['maximo-0']);
  assert.equal(m.obtenerEstado().proceso.configuracion.ventana_desde, '2030-01-01'); assert.equal(m.obtenerEstado().proceso.configuracion.fecha_corte, '2035-01-01');
  maximo.value = '1'; maximo.listeners.change(); click(raiz, 'vista-valoracion'); await click(raiz, 'simular');
  assert.equal(enviada.configuracion.ventana_desde, '2030-01-01'); assert.equal(enviada.configuracion.fecha_corte, '2035-01-01');
});
