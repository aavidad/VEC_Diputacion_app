import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { crearEditorConcursos, leerConfiguracion } from './concursos-editor.js';
import { comprobarResultadoConcursos, crearClienteConcursos } from './concursos-cliente.js';
import { renderizarConcursos } from './concursos-vista.js';
import { cargarTextos } from '../../../comun/textos.js';

const ejemplo = JSON.parse(await readFile(new URL('../../../../../internal/modules/provision/adapters/simulacion/ejemplos/concursos-v1.sintetico.json', import.meta.url), 'utf8'));
const pendiente = () => ({ schema_version: 'provision.simulacion.v1', alcance: 'simulacion', resultado: {
  estado: 'simulacion_local_sin_efectos', version_motor: 'provision.v1', version_reglas: ejemplo.configuracion.version,
  convocatoria_ref: ejemplo.configuracion.convocatoria_ref, completo: false, total: null,
  incidencias: ['dato_no_disponible:grado'], desglose: [], huella_reglas: 'a'.repeat(64), huella_entrada: 'b'.repeat(64), huella_resultado: 'c'.repeat(64),
} });

test('Concursos compara versiones con el motor, conserva su entrada y cancela resultados tardíos', async () => {
  const peticiones = [];
  const editor = crearEditorConcursos({ cliente: { simular: async (s) => { peticiones.push(s); return pendiente(); } } });
  editor.cargar(ejemplo); editor.editar(['maximo_total'], '1000000'); await editor.comparar();
  assert.equal(peticiones[0].configuracion.maximo_total, ejemplo.configuracion.maximo_total);
  assert.equal(peticiones[1].configuracion.maximo_total, '1000000');
  assert.equal(peticiones[1].ejemplo_ref, ejemplo.referencia); assert.ok(!Object.hasOwn(peticiones[1], 'entrada'));
  let resolver;
  const lento = crearEditorConcursos({ cliente: { simular: () => new Promise((r) => { resolver = r; }) } });
  lento.cargar(ejemplo); const trabajo = lento.comparar(); lento.editar(['maximo_total'], '0'); resolver(pendiente()); await trabajo;
  assert.equal(lento.estado().comparacion, null); assert.equal(lento.estado().trabajando, false);
});

test('importación rechazada por Go conserva borrador, comparación e inválidos', async () => {
  let rechazar = false;
  const editor = crearEditorConcursos({ cliente: { simular: async () => { if (rechazar) throw Object.assign(new Error(), { codigo: 'reglas_invalidas' }); return pendiente(); } } });
  editor.cargar(ejemplo); editor.editar(['maximo_total'], '1000000'); await editor.comparar();
  const anterior = editor.estado(); rechazar = true;
  const configuracion = structuredClone(ejemplo.configuracion); configuracion.no_admitido = true;
  await assert.rejects(editor.importar(JSON.stringify(configuracion)), { codigo: 'archivo_invalido' });
  assert.deepEqual(editor.estado().borrador, anterior.borrador); assert.deepEqual(editor.estado().comparacion, anterior.comparacion);
  assert.throws(() => leerConfiguracion('{'), { codigo: 'archivo_invalido' });
  assert.throws(() => editor.editar(['__proto__', 'x'], '1'), { codigo: 'campo_invalido' });
});

test('editar mientras se importa impide sustituir el borrador con una respuesta tardía', async () => {
  let resolver;
  const editor = crearEditorConcursos({ cliente: { simular: () => new Promise((r) => { resolver = r; }) } });
  editor.cargar(ejemplo); const pendienteImportacion = editor.importar(JSON.stringify(ejemplo.configuracion));
  editor.editar(['maximo_total'], '1000000'); resolver(pendiente());
  assert.equal(await pendienteImportacion, false); assert.equal(editor.estado().borrador.maximo_total, '1000000');
});

test('un campo inválido retira comparación y bloquea exportación sin perder el texto', async () => {
  const editor = crearEditorConcursos({ cliente: { simular: async () => pendiente() } });
  editor.cargar(ejemplo); await editor.comparar(); editor.editar(['fecha_corte'], '', { invalido: true });
  assert.equal(editor.estado().comparacion, null); assert.equal(editor.estado().invalidos['["fecha_corte"]'], '');
  assert.throws(() => editor.exportar(), { codigo: 'campo_invalido' }); editor.cancelar();
  assert.equal(editor.estado().cambiado, true); assert.equal(editor.estado().invalidos['["fecha_corte"]'], '');
});

test('cliente rechaza totales inventados, origen de proceso y huellas inválidas', async () => {
  const s = { configuracion: ejemplo.configuracion };
  assert.equal(comprobarResultadoConcursos(pendiente(), s).resultado.total, null);
  for (const alterar of [(r) => r.total = '0', (r) => r.convocatoria_ref = 'otra', (r) => r.huella_reglas = 'x']) {
    const datos = pendiente(); alterar(datos.resultado); assert.throws(() => comprobarResultadoConcursos(datos, s), { codigo: 'respuesta_invalida' });
  }
  const peticiones = [];
  const cliente = crearClienteConcursos({ fetchImpl: async (ruta, opciones) => { peticiones.push({ ruta, opciones }); return { ok: true, text: async () => JSON.stringify(pendiente()) }; } });
  await cliente.simular(s);
  assert.equal(peticiones[0].opciones.credentials, 'omit'); assert.equal(peticiones[0].opciones.redirect, 'error');
});

test('catálogos reales es/en muestran tablas, fecha exclusiva, incidencias y activación cerrada', async () => {
  for (const idioma of ['es', 'en']) {
    const textos = await cargarTextos('baremo-concursos', { idioma });
    const editor = crearEditorConcursos({ cliente: { simular: async () => pendiente() } }); editor.cargar(ejemplo); await editor.comparar();
    const html = renderizarConcursos(editor.estado(), { textos, ejemplos: [ejemplo] });
    assert.ok(html.includes(textos.traducir('concursos.fecha_corte')));
    assert.ok(html.includes(textos.traducir('concursos.incidencia_dato_no_disponible', { familia: textos.traducir('concursos.familia_grado') })));
    assert.match(html, /data-concurso-ruta="\[&quot;reglas&quot;,0,&quot;tramos&quot;,0,&quot;coeficiente&quot;\]"/u);
    assert.match(html, /disabled aria-describedby="concursos-activacion"/u); assert.doesNotMatch(html, /data-concurso-total/u);
    assert.deepEqual(textos.faltantes, []);
  }
});

test('explica exclusiones y agregaciones sin inventar una ecuación en detalles de entrada', async () => {
  const motivos = ['computado', 'posterior_corte', 'no_acreditado', 'usado_requisito', 'tipo_no_admitido', 'no_relacionado', 'caducado', 'horas_insuficientes', 'limite_elementos', 'fuera_ventana', 'suma_horas', 'conversion_y_coeficiente', 'tope_tramo'];
  for (const idioma of ['es', 'en']) {
    const textos = await cargarTextos('baremo-concursos', { idioma });
    const respuesta = pendiente();
    respuesta.resultado.desglose = [{ familia: 'cursos', estado: 'calculado', bruto: '480000', maximo: '9000000', resultado: '480000', detalles: motivos.map((motivo) => ({ motivo, unidades: '40/1', coeficiente: '12000', bruto: '0', maximo: '0', resultado: '0' })) }];
    const editor = crearEditorConcursos({ cliente: { simular: async () => respuesta } }); editor.cargar(ejemplo); await editor.comparar();
    const html = renderizarConcursos(editor.estado(), { textos, ejemplos: [ejemplo] });
    assert.doesNotMatch(html, /×|= 0|40\s*\/\s*1.*×.*0[.,]012.*=/u);
    for (const motivo of motivos.filter((v) => !['computado', 'suma_horas', 'conversion_y_coeficiente', 'tope_tramo'].includes(v))) {
      assert.ok(html.includes(textos.traducir(`concursos.motivo_${motivo}`)), `${idioma}: ${motivo}`);
    }
    assert.ok(html.includes(textos.traducir('concursos.motivo_computado', { unidades: '40/1' })));
  }
});
