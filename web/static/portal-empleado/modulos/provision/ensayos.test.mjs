import test from 'node:test';
import assert from 'node:assert/strict';
import { cargarTextos } from '../../../comun/textos.js';
import { prepararAdjudicacion, cambiarAdjudicacion, validarAdjudicacion, crearControladorAdjudicacion } from './ensayos-modelo.js';
import { crearClienteEnsayosLocal } from './ensayos-cliente.js';
import { pintarAdjudicacion } from './ensayos-vista.js';
const ejemplo = () => ({ ejemplo_ref: 'ensayo:1', configuracion: { schema_version: 'provision.adjudicacion.v1', proceso_ref: 'proceso:1', version: 'v1', politica_ref: 'politica:1', politica_version: 'p1', bases_ref: 'bases:1', metodo: 'aceptacion_diferida_personas_proponen_ensayo_v1', prioridad: 'total_descendente', incompatibilidad: 'un_puesto_por_persona', desempates: [{ regla_id: 'regla:cursos', sentido: 'mayor' }, { regla_id: 'regla:grado', sentido: 'menor' }] }, resumen: { vacantes: [{ vacante_ref: 'vacante:a', puesto_ref: 'puesto:a' }], solicitudes: [{ persona_ref: 'persona:a', preferencias: [{ vacante_ref: 'vacante:a', orden: 1, total: '20000000' }] }] } });
const respuesta = (estado, parcial = {}) => ({ schema_version: 'provision.adjudicacion.v1', alcance: 'simulacion_sintetica_sin_efectos', estado: 'propuesta_simulada', proceso_ref: estado.configuracion.proceso_ref, version: estado.configuracion.version, politica_ref: estado.configuracion.politica_ref, politica_version: estado.configuracion.politica_version, metodo: estado.configuracion.metodo, asignaciones: [{ persona_ref: 'persona:a', vacante_ref: 'vacante:a', puesto_ref: 'puesto:a', preferencia: 1 }], incidencias: [], huella_resultado: 'abc', ...parcial });
function dom() {
  const d = { createElement: tag => new N(tag) };
  class N { constructor(tag) { this.tagName = tag; this.ownerDocument = d; this.nodeType = 1; this.children = []; this.dataset = {}; this.atributos = {}; this.listeners = {}; }
    append(...hijos) { this.children.push(...hijos); } replaceChildren(...hijos) { this.children = hijos; }
    setAttribute(k, v) { this.atributos[k] = v; } addEventListener(k, fn) { this.listeners[k] = fn; }
    querySelectorAll() { return nodos(this).filter(n => n.dataset.foco); }
  } return new N('main');
}
const nodos = n => [n, ...n.children.flatMap(nodos)];
test('la cadena del ensayo conserva orden explícito; cambios invalidan el resultado', () => {
  const original = prepararAdjudicacion(ejemplo()); original.resultado = respuesta(original);
  const siguiente = cambiarAdjudicacion(original, { indice: 0, accion: 'abajo' });
  assert.equal(siguiente.configuracion.desempates[0].regla_id, 'regla:grado'); assert.equal(siguiente.resultado, null);
  assert.equal(original.configuracion.desempates[0].regla_id, 'regla:cursos'); assert.throws(() => cambiarAdjudicacion(original, { campo: 'persona_ref', valor: 'ajena' }));
});
test('rechaza asignaciones duplicadas, ajenas o parciales en un ensayo pendiente', () => {
  const estado = prepararAdjudicacion(ejemplo()); const r = respuesta(estado); assert.equal(validarAdjudicacion(r, estado).estado, 'propuesta_simulada');
  assert.throws(() => validarAdjudicacion({ ...r, asignaciones: [...r.asignaciones, ...r.asignaciones] }, estado));
  assert.throws(() => validarAdjudicacion({ ...r, estado: 'pendiente' }, estado));
  assert.throws(() => validarAdjudicacion({ ...r, alcance: 'oficial' }, estado));
  const pendiente = validarAdjudicacion({ ...r, estado: 'pendiente', asignaciones: [], incidencias: [{ codigo: 'empate_residual' }] }, estado); assert.equal(pendiente.incidencias[0].codigo, 'empate_residual');
});
test('transporte sólo envía referencia y configuración; no acepta puntuaciones cliente', async () => {
  let body; const cliente = crearClienteEnsayosLocal({ fetchImpl: async (_url, opts) => { body = JSON.parse(opts.body); assert.equal(opts.credentials, 'omit'); return { ok: true, text: async () => '{}' }; } });
  await cliente.simularAdjudicacion({ ejemplo_ref: 'ensayo:1', configuracion: ejemplo().configuracion, solicitudes: [{ total: '99999999' }] });
  assert.deepEqual(Object.keys(body), ['ejemplo_ref', 'configuracion']);
});
test('controlador cancela al cambiar caso y no aplica una respuesta tardía', async () => {
  let resolver; let signal; const controlador = crearControladorAdjudicacion({ cliente: { listarAdjudicaciones: async () => ({ ejemplos: [ejemplo()] }), simularAdjudicacion: (_p, opts) => { signal = opts.signal; return new Promise(r => resolver = r); } }, notificar: () => {} });
  await controlador.cargar(); const inicial = controlador.obtenerEstado(); const promesa = controlador.simular(); controlador.elegir(0); assert.equal(signal.aborted, true); resolver(respuesta(inicial)); await promesa; assert.equal(controlador.obtenerEstado().resultado, null); controlador.desmontar();
});
test('estado vacío y error de validación no presentan asignaciones', async () => {
  let ultimo; const c = crearControladorAdjudicacion({ cliente: { listarAdjudicaciones: async () => ({ ejemplos: [] }) }, notificar: e => ultimo = e }); await c.cargar(); assert.equal(ultimo.estado, 'vacio'); c.desmontar();
  const c2 = crearControladorAdjudicacion({ cliente: { listarAdjudicaciones: async () => ({ ejemplos: [ejemplo()] }) }, notificar: e => ultimo = e }); await c2.cargar(); const error = c2.cambiar({ campo: 'politica_ref', valor: '' }); assert.equal(error.estado.estado, 'validacion'); assert.equal(error.estado.invalidos.politica_ref, ''); c2.desmontar();
});
test('vista ES/EN presenta nombres de catálogo, empate real y resolución deshabilitada', async () => {
  for (const idioma of ['es', 'en']) {
    const textos = await cargarTextos('provision', { idioma }); assert.deepEqual(textos.faltantes, []);
    const raiz = dom(); const estado = prepararAdjudicacion(ejemplo()); estado.resultado = respuesta(estado, { estado: 'pendiente', asignaciones: [], incidencias: [{ codigo: 'empate_residual' }] }); estado.estado = 'resultado';
    pintarAdjudicacion({ raiz, estado, textos, acciones: {} });
    assert.equal(nodos(raiz).find(n => n.dataset.foco === 'adjudicacion-resolver').disabled, true);
    assert.equal(nodos(raiz).some(n => n.textContent === textos.traducir('ensayos.incidencias.empate_residual')), true);
    assert.equal(nodos(raiz).some(n => n.textContent === textos.traducir('ensayos.personas.persona:a')), true);
  }
});
