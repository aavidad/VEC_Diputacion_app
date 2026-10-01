import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { crearEditorBaremo, aMicropuntos, aDecimal, leerReglas } from './baremo-editor.js';
import { comprobarSimulacion, crearClienteBaremo } from './baremo-cliente.js';
import { renderizarBaremo } from './baremo-vista.js';
import { cargarTextos } from '../../../comun/textos.js';
const reglas = JSON.parse(await readFile(new URL('../../../../../internal/modules/bolsa/application/simulacionbaremo/testdata/reglas_a.json', import.meta.url), 'utf8'));
const ejemplo = { referencia: 'experiencia-a', modo: 'experiencia', reglas };
const resultado = (total = '101667') => ({ esquema: 'vec.bolsa.simulacion_experiencia.v1', alcance: 'simulacion', convocatoria_ref: reglas.identidad.convocatoria_ref,
 huella_resultado_sha256: 'a'.repeat(64), resultado: { estado: 'completado', total, secciones: [] } });
const pendiente = () => { let resolver; const promise = new Promise((r) => { resolver = r; }); return { promise, resolver }; };

test('edita sin mutar las reglas cargadas y envía ambas versiones al mismo motor', async () => {
 const solicitudes = [];
 const editor = crearEditorBaremo({ cliente: { simular: async (s) => { solicitudes.push(s); return resultado(); } } });
 editor.cargar(ejemplo); editor.editar(['reglas_experiencia', 0, 'puntos_por_unidad'], aMicropuntos('0,2'));
 assert.equal(ejemplo.reglas.reglas_experiencia[0].puntos_por_unidad, '100000');
 await editor.comparar();
 assert.equal(solicitudes.length, 2);
 assert.deepEqual(solicitudes.map((s) => s.ejemplo_ref), ['experiencia-a', 'experiencia-a']);
 assert.equal(solicitudes[0].reglas.reglas_experiencia[0].puntos_por_unidad, '100000');
 assert.equal(solicitudes[1].reglas.reglas_experiencia[0].puntos_por_unidad, '200000');
 assert.ok(editor.estado().comparacion);
 editor.editar(['fecha_corte_inclusiva'], '2026-09-01');
 assert.equal(editor.estado().comparacion, null);
});

test('descarta respuestas tardías y cancela al cambiar de caso', async () => {
 const retraso = pendiente(); let signal;
 const editor = crearEditorBaremo({ cliente: { simular: (_s, opciones) => { signal = opciones.signal; return retraso.promise; } } });
 editor.cargar(ejemplo); const trabajo = editor.comparar();
 editor.cargar({ ...ejemplo, referencia: 'experiencia-b' });
 assert.equal(signal.aborted, true);
 retraso.resolver(resultado()); await trabajo;
 assert.equal(editor.estado().comparacion, null);
 assert.equal(editor.estado().ejemplo.referencia, 'experiencia-b');
});

test('conserva el error del motor y nunca inventa un resultado', async () => {
 const editor = crearEditorBaremo({ cliente: { simular: async () => { throw Object.assign(new Error(), { codigo: 'reglas_invalidas' }); } } });
 editor.cargar(ejemplo); await editor.comparar();
 assert.equal(editor.estado().error, 'reglas_invalidas');
 assert.equal(editor.estado().comparacion, null);
 assert.equal(editor.estado().trabajando, false);
});

test('convierte representación decimal exacta y rechaza exponentes, negativos y precisión excesiva', () => {
 for (const v of ['1e3', '-1', 'NaN', '1.0000001', '01', '']) assert.throws(() => aMicropuntos(v));
 assert.equal(aMicropuntos('9999999999999,123456'), '9999999999999123456');
 assert.equal(aDecimal('9999999999999123456'), '9999999999999.123456');
 assert.equal(aMicropuntos('0'), '0');
});

test('importa solo configuración acotada y rechaza rutas que alteran prototipos', () => {
 const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
 assert.throws(() => leerReglas('{"foo":1}'));
 assert.throws(() => editor.editar(['__proto__', 'x'], 'y'));
 assert.equal(editor.exportar(), JSON.stringify(reglas));
});

test('rechaza criterios malformados antes de publicar la carga y conserva el borrador y su comparación', async () => {
 const textos = await cargarTextos('baremo-bolsa');
 let editor; let publicaciones = 0;
 editor = crearEditorBaremo({ cliente: { simular: async () => resultado() }, alCambiar: () => {
  publicaciones++;
  renderizarBaremo(editor.estado(), { textos, ejemplos: [ejemplo] });
 } });
 editor.cargar(ejemplo);
 editor.editar(['reglas_experiencia', 0, 'puntos_por_unidad'], '200000');
 await editor.comparar();
 const preimagen = editor.estado(); const exportado = editor.exportar(); const avisos = publicaciones;
 for (const criterios of [{}, 'ambito', null, [null], [{ valores: {} }], [{ valores: [null] }]]) {
  const malformadas = structuredClone(reglas); malformadas.reglas_experiencia[0].criterios = criterios;
  assert.throws(() => leerReglas(JSON.stringify(malformadas)), /archivo_invalido/u);
  assert.throws(() => editor.cargar(ejemplo, malformadas), /archivo_invalido/u);
  assert.deepEqual(editor.estado(), preimagen);
  assert.equal(editor.exportar(), exportado);
  assert.equal(publicaciones, avisos);
 }
});

test('una carga inválida no cancela ni borra la comparación que sigue en curso', async () => {
 const retraso = pendiente(); let signal; let llamadas = 0;
 const editor = crearEditorBaremo({ cliente: { simular: (_s, opciones) => {
  signal = opciones.signal;
  return llamadas++ === 0 ? retraso.promise : Promise.resolve(resultado());
 } } });
 editor.cargar(ejemplo); const trabajo = editor.comparar(); const preimagen = editor.estado();
 assert.throws(() => editor.cargar(ejemplo, { foo: 1 }), /archivo_invalido/u);
 assert.deepEqual(editor.estado(), preimagen);
 assert.equal(signal.aborted, false);
 retraso.resolver(resultado()); await trabajo;
 assert.ok(editor.estado().comparacion);
 assert.equal(editor.estado().trabajando, false);
});

test('transporta sin credenciales ni redirecciones y acepta un bloqueo sin total', async () => {
 const llamadas = [];
 const bloqueado = { ...resultado(), resultado: { estado: 'bloqueado', bloqueos: [] } };
 const cliente = crearClienteBaremo({ fetchImpl: async (url, opciones) => { llamadas.push({ url, opciones }); return { ok: true, text: async () => JSON.stringify(bloqueado) }; } });
 assert.equal((await cliente.simular({ modo: 'experiencia', ejemplo_ref: 'experiencia-a', reglas })).resultado.estado, 'bloqueado');
 assert.equal(llamadas[0].opciones.credentials, 'omit');
 assert.equal(llamadas[0].opciones.redirect, 'error');
 assert.equal(comprobarSimulacion({ ...bloqueado, esquema: 'vec.bolsa.simulacion_meritos.v1', resultado: { estado: 'bloqueado', incidencias: [] } }).resultado.estado, 'bloqueado');
 assert.throws(() => comprobarSimulacion({ ...bloqueado, resultado: { estado: 'bloqueado', bloqueos: [], total: '0' } }));
 assert.throws(() => comprobarSimulacion({ ...resultado(), alcance: 'oficial' }));
 assert.throws(() => comprobarSimulacion({ ...resultado(), resultado: { estado: 'completado' } }));
});

test('vista usa catálogo real en ambos idiomas, escapa datos y mantiene activación deshabilitada', async () => {
 for (const idioma of ['es', 'en']) {
  const textos = await cargarTextos('baremo-bolsa', { idioma });
  assert.deepEqual(textos.faltantes, []);
  const editor = crearEditorBaremo({ cliente: {} }); editor.cargar({ ...ejemplo, referencia: '<img src=x onerror=alert(1)>' });
  const html = renderizarBaremo(editor.estado(), { textos, ejemplos: [editor.estado().ejemplo] });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /disabled aria-describedby="baremo-activacion"/u);
  assert.match(html, /type="date"[^>]*required/u);
  assert.match(html, /data-puntos/u);
 }
});


test('conserva un campo inválido al repintar ayuda y bloquea la exportación', async () => {
 const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
 editor.registrarInvalido(['reglas_experiencia', 0, 'puntos_por_unidad'], '1e3');
 assert.equal(editor.estado().cambiado, true);
 assert.throws(() => editor.exportar(), /campo_invalido/u);
 const textos = await cargarTextos('baremo-bolsa');
 const html = renderizarBaremo({ ...editor.estado(), ayuda: true }, { textos, ejemplos: [ejemplo] });
 assert.match(html, /value="1e3" aria-invalid="true"/u);
 assert.match(html, /data-accion="exportar" disabled/u);
 editor.editar(['reglas_experiencia', 0, 'puntos_por_unidad'], '200000');
 assert.equal(JSON.parse(editor.exportar()).reglas_experiencia[0].puntos_por_unidad, '200000');
});

test('presenta los desgloses canónicos de experiencia y méritos sin guiones ni apartados genéricos', async () => {
 const textos = await cargarTextos('baremo-bolsa');
 const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
 const antes = resultado(); antes.resultado.secciones = [{ seccion: 'experiencia', antes_tope: '101667/1', tope: { limite: '1000000/1' }, puntos_finales: '101667' }];
 const despues = resultado('4000000'); despues.resultado.secciones = [{ clave: 'formacion', suma_reglas: '1500000', maximo_puntos: '1200000', puntos: '1200000' }];
 const html = renderizarBaremo({ ...editor.estado(), comparacion: { antes, despues } }, { textos, ejemplos: [ejemplo] });
 assert.match(html, /Experiencia/u); assert.match(html, /0,101667/u); assert.match(html, /Formación/u); assert.match(html, />1,2</u);
 assert.doesNotMatch(html, />—</u);
});

test('cambiar de panel cancela la simulación y conserva borrador, inválidos y error previo', async () => {
 const retraso = pendiente(); let signal; let llamadas = 0;
 const editor = crearEditorBaremo({ cliente: { simular: (_s, opciones) => { llamadas++; signal = opciones.signal; return retraso.promise; } } });
 editor.cargar(ejemplo); editor.editar(['fecha_corte_inclusiva'], '2026-09-01');
 const trabajo = editor.comparar(); editor.cancelarSimulacion();
 assert.equal(signal.aborted, true);
 retraso.resolver(resultado()); await trabajo;
 assert.equal(llamadas, 1);
 assert.equal(editor.estado().borrador.fecha_corte_inclusiva, '2026-09-01');
 assert.equal(editor.estado().trabajando, false); assert.equal(editor.estado().comparacion, null);
 editor.registrarInvalido(['reglas_experiencia', 0, 'puntos_por_unidad'], '1e3');
 const guardado = editor.estado(); editor.cancelarSimulacion();
 assert.deepEqual(editor.estado(), guardado);
 const conError = crearEditorBaremo({ cliente: { simular: async () => { throw Object.assign(new Error(), { codigo: 'reglas_invalidas' }); } } });
 conError.cargar(ejemplo); await conError.comparar(); conError.cancelarSimulacion();
 assert.equal(conError.estado().error, 'reglas_invalidas');
});

test('Concursos tiene panel vacío propio, navegación nativa traducida y Bolsa oculta', async () => {
 const { renderizarPanelesBaremo } = await import('./baremo-vista.js');
 for (const idioma of ['es','en']) {
  const textos = await cargarTextos('baremo-bolsa', { idioma });
  const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
  const html = renderizarPanelesBaremo(editor.estado(), { textos, ejemplos: [ejemplo], panel: 'concursos' });
  assert.match(html, /id="baremo-panel-bolsa" hidden/u);
  assert.match(html, /data-panel="concursos"[^>]*aria-current="page"/u);
  const concursos = html.slice(html.indexOf('<section id="baremo-panel-concursos"'));
  assert.doesNotMatch(concursos, /<form|<input|data-accion="comparar"|<button/u);
  assert.ok(concursos.includes(textos.traducir('editor.concursos_vacio')));
  assert.ok(concursos.includes(textos.traducir('editor.concursos_limite')));
  const bolsa = renderizarPanelesBaremo(editor.estado(), { textos, ejemplos: [ejemplo], panel: 'bolsa' });
  assert.match(bolsa, /id="baremo-panel-concursos"[^>]* hidden/u);
 }
});
