import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { crearEditorBaremo, aMicropuntos, aDecimal, leerReglas, normalizarFraccionJornada, comprobarCatalogoJornada } from './baremo-editor.js';
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

test('desglosa bruto, máximo y resultado del motor para topes activos, iguales y no alcanzados', async () => {
 for (const idioma of ['es', 'en']) {
  const textos = await cargarTextos('baremo-bolsa', { idioma });
  const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
  const antes = resultado('3000000'); antes.resultado.secciones = [
   { seccion: 'experiencia', antes_tope: '3000000/1', tope: { limite: '1000000/1', aplicado: true }, puntos_finales: '1000000' },
   { seccion: 'formacion', antes_tope: '1000000/1', tope: { limite: '1000000/1', aplicado: false }, puntos_finales: '1000000' },
   { seccion: 'otros', antes_tope: '500000/1', tope: { limite: '1000000/1', aplicado: false }, puntos_finales: '500000' },
  ];
  const despues = resultado('3000000'); despues.resultado.secciones = [
   { clave: 'experiencia', suma_reglas: '3000000', maximo_puntos: '1000000', puntos: '1000000' },
   { clave: 'formacion', suma_reglas: '1000000', maximo_puntos: '1000000', puntos: '1000000' },
   { clave: 'otros', suma_reglas: '500000', maximo_puntos: '1000000', puntos: '500000' },
  ];
  const comparacion = { antes, despues }, preimagen = structuredClone(comparacion);
  const html = renderizarBaremo({ ...editor.estado(), comparacion }, { textos, ejemplos: [ejemplo] });
  for (const clave of ['antes_tope', 'maximo_apartado', 'despues_tope']) assert.ok(html.includes(textos.traducir(`editor.${clave}`)));
  const filas = [...html.matchAll(/<tr><th scope="row">(?:Experiencia|Experience|Formación|Training|Otros méritos|Other merits)<\/th>(.*?)<\/tr>/gu)].map((m) => m[1]);
  assert.equal(filas.length, 6);
  for (const offset of [0, 3]) {
   assert.match(filas[offset], />3<\/td><td class="columna-numero">1<\/td><td class="columna-numero">1<\/td>/u);
   assert.match(filas[offset + 1], />1<\/td><td class="columna-numero">1<\/td><td class="columna-numero">1<\/td>/u);
   assert.ok(filas[offset + 2].includes(idioma === 'es' ? '>0,5</td>' : '>0.5</td>'));
  }
  assert.deepEqual(comparacion, preimagen);
 }
});

test('no aproxima racionales ni transforma datos ausentes o malformados en cero', async () => {
 for (const idioma of ['es', 'en']) {
  const textos = await cargarTextos('baremo-bolsa', { idioma });
  const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
  const antes = resultado(); antes.resultado.secciones = [
   { seccion: 'experiencia', antes_tope: '1000000/3', tope: { limite: '2000000/1' }, puntos_finales: '333333' },
   { seccion: 'otros', antes_tope: '0/1', tope: { limite: null }, puntos_finales: '0' },
  ];
  const despues = resultado(); despues.resultado.secciones = [
   { clave: 'formacion', suma_reglas: '<img src=x>', maximo_puntos: '1000000/0', puntos: '1' },
   { clave: 'otros', puntos: '0' },
  ];
  const html = renderizarBaremo({ ...editor.estado(), comparacion: { antes, despues } }, { textos, ejemplos: [ejemplo] });
  assert.match(html, />1 ÷ 3<\/td>/u);
  assert.doesNotMatch(html, /0[,.]333333333|<img/u);
  assert.ok(html.includes(textos.traducir('editor.sin_tope')));
  assert.equal(html.split(textos.traducir('editor.dato_no_disponible')).length - 1, 4);
  assert.match(html, />0<\/td><td class="columna-numero">(?:Sin tope|No limit)<\/td><td class="columna-numero">0<\/td>/u);
 }
});

test('explica el tope global de méritos con la suma y el total canónicos sin consultar el borrador', async () => {
 for (const idioma of ['es', 'en']) {
  const textos = await cargarTextos('baremo-bolsa', { idioma });
  const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
  const estado = editor.estado(); estado.borrador.maximo_total = '999000000';
  for (const [suma, limite, total] of [['4100000','4000000','4000000'],['4000000','4000000','4000000'],['3500000','4000000','3500000'],['0','0','0']]) {
   const antes = { ...resultado(total), esquema: 'vec.bolsa.simulacion_meritos.v1' };
   Object.assign(antes.resultado, { suma_secciones: suma, maximo_total: limite });
   const despues = { ...resultado('4100000'), esquema: 'vec.bolsa.simulacion_meritos.v1' };
   Object.assign(despues.resultado, { suma_secciones: '4100000', maximo_total: '5000000' });
   const comparacion = { antes, despues }, preimagen = structuredClone(comparacion);
   const html = renderizarBaremo({ ...estado, comparacion }, { textos, ejemplos: [ejemplo] });
   const resumenes = [...html.matchAll(/<dl>(.*?)<\/dl>/gu)].map((m) => m[1]);
   const puntos = (valor) => aDecimal(valor).replace('.', idioma === 'es' ? ',' : '.');
   assert.equal(resumenes.length, 2);
   assert.equal(resumenes[0], `<div><dt>${textos.traducir('editor.suma_apartados')}</dt><dd>${puntos(suma)}</dd></div><div><dt>${textos.traducir('editor.tope_global')}</dt><dd>${puntos(limite)}</dd></div><div><dt>${textos.traducir('editor.total')}</dt><dd>${puntos(total)}</dd></div>`);
   assert.ok(resumenes[1].includes(`<dd>${puntos('4100000')}</dd>`));
   assert.ok(resumenes[1].includes('<dd>5</dd>'));
   assert.doesNotMatch(resumenes.join(''), />999</u);
   assert.deepEqual(comparacion, preimagen);
  }
 }
});

test('experiencia sin campos globales conserva solo su total y un dato ausente no se deduce de las secciones', async () => {
 const textos = await cargarTextos('baremo-bolsa');
 const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
 const antes = resultado('101667'); antes.resultado.secciones = [{ seccion: 'experiencia', puntos_finales: '101667' }];
 const despues = { ...resultado('4000000'), esquema: 'vec.bolsa.simulacion_meritos.v1' };
 despues.resultado.suma_secciones = '4100000';
 const estado = editor.estado(); estado.borrador.maximo_total = '999000000';
 const html = renderizarBaremo({ ...estado, comparacion: { antes, despues } }, { textos, ejemplos: [ejemplo] });
 const resumenes = [...html.matchAll(/<dl>(.*?)<\/dl>/gu)].map((m) => m[1]);
 assert.equal(resumenes[0], '<div><dt>Total</dt><dd>0,101667</dd></div>');
 assert.equal(resumenes[1], '<div><dt>Suma de apartados</dt><dd>4,1</dd></div><div><dt>Tope global</dt><dd>Dato no disponible</dd></div><div><dt>Total</dt><dd>4</dd></div>');
});

test('ambos motores bloqueados no muestran total, suma ni máximos parciales', async () => {
 const textos = await cargarTextos('baremo-bolsa');
 const editor = crearEditorBaremo({ cliente: {} }); editor.cargar(ejemplo);
 const antes = { ...resultado(), resultado: { estado: 'bloqueado', bloqueos: [{codigo:'reglas_en_grupos_distintos'}] } };
 const despues = { ...resultado(), esquema: 'vec.bolsa.simulacion_meritos.v1', resultado: { estado: 'bloqueado', incidencias: [{codigo:'merito_duplicado'}], maximo_total: '4000000' } };
 const html = renderizarBaremo({ ...editor.estado(), comparacion: { antes, despues } }, { textos, ejemplos: [ejemplo] });
 const resultados = html.slice(html.indexOf('class="cuerpo-panel baremo-resultados"'), html.indexOf('<footer'));
 assert.doesNotMatch(resultados, /<dl>|<dd>|<table>/u);
 assert.ok(resultados.includes(textos.traducir('editor.bloqueo_merito_duplicado')));
 assert.ok(resultados.includes(textos.traducir('editor.bloqueo_reglas_en_grupos_distintos')));
});


test('normaliza el umbral exacto sin redondear y respeta los límites del racional V1', () => {
 assert.equal(normalizarFraccionJornada('2/4'), '1/2');
 assert.equal(normalizarFraccionJornada('1/3'), '1/3');
 assert.equal(normalizarFraccionJornada('2000000000/4000000000'), '1/2');
 assert.equal(normalizarFraccionJornada('999999999/1000000000'), '999999999/1000000000');
 for (const valor of ['', null, '0/1', '1/0', '-1/2', '3/2', '0.5', '1e3/2', '01/2', '1/1000000001', '<img>']) {
  assert.throws(() => normalizarFraccionJornada(valor), /umbral_invalido/u);
 }
});

const catalogoJornada = JSON.parse(await readFile(new URL('../../../catalogos/baremo-jornada-v1.json', import.meta.url), 'utf8'));

test('catálogo de jornadas coincide con variantes del modelo y disponibilidad del compilador Go V1', async () => {
 const modelo = await readFile(new URL('../../../../../internal/modules/bolsa/domain/reglasbaremo/tipos.go', import.meta.url), 'utf8');
 const compilador = await readFile(new URL('../../../../../internal/modules/bolsa/domain/calculoexperiencia/compilacion.go', import.meta.url), 'utf8');
 const variantes = new Map([...modelo.matchAll(/(Jornada\w+)\s+ModoJornada\s*=\s*"([^"]+)"/gu)].map((m) => [m[1], m[2]]));
 const casosAdmitidos = /switch regla\.Jornada\(\)\.Modo\(\) \{\s*case ([\s\S]+?):\s*case/u.exec(compilador)[1];
 const admitidas = [...casosAdmitidos.matchAll(/reglasbaremo\.(Jornada\w+)/gu)].map((m) => variantes.get(m[1]));
 const comprobado = comprobarCatalogoJornada(catalogoJornada);
 assert.deepEqual(comprobado.opciones.map((o) => o.modo).sort(), [...variantes.values()].sort());
 assert.deepEqual(comprobado.opciones.filter((o) => o.disponible).map((o) => o.modo).sort(), admitidas.sort());
 for (const idioma of ['es','en']) {
  const textos = await cargarTextos('baremo-bolsa', { idioma });
  for (const opcion of comprobado.opciones) { assert.ok(textos.traducir(`editor.${opcion.etiqueta}`)); if (!opcion.disponible) assert.ok(textos.traducir(`editor.${opcion.motivo}`)); }
 }
});

test('cambia jornada sin umbral predeterminado, conserva inválidos y elimina umbral al salir', async () => {
 const solicitudes = []; const editor = crearEditorBaremo({ catalogoJornada, cliente: { simular: async(s) => { solicitudes.push(s); return resultado(); } } });
 editor.cargar(ejemplo); await editor.comparar(); assert.ok(editor.estado().comparacion);
 editor.editarPoliticaJornada(0, 'integra_desde_umbral');
 assert.deepEqual(editor.estado().borrador.reglas_experiencia[0].jornada, { modo:'integra_desde_umbral', umbral:'' });
 assert.equal(editor.estado().comparacion, null); assert.throws(()=>editor.exportar(), /campo_invalido/u);
 await editor.comparar(); assert.equal(solicitudes.length, 2);
 assert.throws(()=>editor.editarUmbralJornada(0, '1/0'), /umbral_invalido/u);
 assert.equal(editor.estado().invalidos['["reglas_experiencia",0,"jornada","umbral"]'], '1/0');
 const textos = await cargarTextos('baremo-bolsa');
 assert.match(renderizarBaremo({...editor.estado(),ayuda:true},{textos,ejemplos:[ejemplo]}), /value="1\/0" aria-invalid="true"/u);
 editor.editarUmbralJornada(0, '2/4'); await editor.comparar();
 assert.deepEqual(solicitudes.at(-1).reglas.reglas_experiencia[0].jornada, {modo:'integra_desde_umbral',umbral:'1/2'});
 const exportado = JSON.parse(editor.exportar());
 assert.deepEqual(exportado.reglas_experiencia[0].jornada,{modo:'integra_desde_umbral',umbral:'1/2'});
 assert.deepEqual(ejemplo.reglas.reglas_experiencia[0].jornada,{modo:'proporcional'});
 editor.editarPoliticaJornada(0,'protegida_integra');
 assert.deepEqual(editor.estado().borrador.reglas_experiencia[0].jornada,{modo:'protegida_integra'});
 assert.deepEqual(editor.estado().invalidos,{}); assert.equal(editor.estado().comparacion,null);
 editor.editarPoliticaJornada(0,'integra_desde_umbral'); assert.equal(editor.estado().borrador.reglas_experiencia[0].jornada.umbral,'');
 assert.throws(()=>editor.editarUmbralJornada(0,'1/0'),/umbral_invalido/u);
 editor.editarPoliticaJornada(0,'integra');assert.deepEqual(editor.estado().invalidos,{});assert.deepEqual(JSON.parse(editor.exportar()).reglas_experiencia[0].jornada,{modo:'integra'});
 const importador=crearEditorBaremo({catalogoJornada,cliente:{}});importador.cargar(ejemplo,leerReglas(JSON.stringify(exportado)));assert.deepEqual(JSON.parse(importador.exportar()),exportado);
});

test('una política importada no soportada se conserva y bloquea comparación sin sustituirla', async () => {
 let llamadas = 0; const editor = crearEditorBaremo({ catalogoJornada, cliente: { simular: async()=>{ llamadas++; return resultado(); } } });
 for (const modo of ['por_horas','politica_futura']) {
  const importadas = structuredClone(reglas);importadas.reglas_experiencia[0].jornada={modo};
  editor.cargar(ejemplo,importadas);await editor.comparar();
  assert.equal(llamadas,0);assert.equal(editor.estado().error,'jornada_no_disponible');
  assert.deepEqual(JSON.parse(editor.exportar()).reglas_experiencia[0].jornada,{modo});
 }
 const preimagen=editor.estado();assert.throws(()=>editor.editarPoliticaJornada(0,'por_horas'),/jornada_no_disponible/u);assert.deepEqual(editor.estado(),preimagen);
 editor.editarPoliticaJornada(0,'integra');await editor.comparar();assert.equal(llamadas,2);assert.ok(editor.estado().comparacion);
});

test('sin catálogo compatible la edición se cierra y la política cargada permanece intacta', async () => {
 for (const catalogo of [null,{...catalogoJornada,version:2},{...catalogoJornada,motor:'otro'}, {...catalogoJornada,opciones:[catalogoJornada.opciones[0]]}]) {
  const editor=crearEditorBaremo({catalogoJornada:catalogo,cliente:{}});editor.cargar(ejemplo);
  assert.throws(()=>editor.editarPoliticaJornada(0,'integra'),/jornada_no_disponible/u);
  assert.deepEqual(JSON.parse(editor.exportar()),reglas);
  const textos=await cargarTextos('baremo-bolsa');const html=renderizarBaremo(editor.estado(),{textos,ejemplos:[ejemplo]});
  assert.match(html,/data-jornada-modo[^>]* disabled/u);assert.ok(html.includes(textos.traducir('editor.catalogo_jornada_no_disponible')));
 }
});

test('cambiar política cancela cálculo tardío y una importación de umbral inválida conserva el borrador', async () => {
 const espera=pendiente();let signal;const editor=crearEditorBaremo({catalogoJornada,cliente:{simular:(_s,o)=>{signal=o.signal;return espera.promise;}}});
 editor.cargar(ejemplo);const vuelo=editor.comparar();editor.editarPoliticaJornada(0,'integra');assert.equal(signal.aborted,true);
 espera.resolver(resultado());await vuelo;assert.equal(editor.estado().comparacion,null);
 const preimagen=editor.estado();
 for (const umbral of ['2/4','1/0',undefined]) {
  const importadas=structuredClone(reglas);importadas.reglas_experiencia[0].jornada={modo:'integra_desde_umbral',...(umbral===undefined?{}:{umbral})};
  assert.throws(()=>editor.cargar(ejemplo,importadas),/archivo_invalido/u);assert.deepEqual(editor.estado(),preimagen);
 }
});
