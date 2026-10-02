import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { crearEditorBaremo, normalizarMinimoFormacion, leerReglas } from './baremo-editor.js';
import { renderizarBaremo } from './baremo-vista.js';
import { cargarTextos } from '../../../comun/textos.js';
const reglas = JSON.parse(await readFile(new URL('../../../../../internal/modules/bolsa/application/simulacionbaremo/testdata/meritos_reglas_a.json', import.meta.url), 'utf8'));
const ejemplo = { referencia: 'meritos_sinteticos_v1', modo: 'meritos', reglas };
const resultado = () => ({ resultado: { estado: 'completado', total: '4000000', secciones: [] } });
const ruta = '["reglas",0,"minimo_unidades"]';

test('la duración admite cero y fracciones de horas mayores que uno sin aproximación', () => {
 for (const [valor, esperado] of [['0/7','0/1'],['62/2','31/1'],['3/2','3/2'],['1/3','1/3'],['2000000000/4000000000','1/2'],['1000000000/1','1000000000/1'],['1/1000000000','1/1000000000']]) {
  assert.equal(normalizarMinimoFormacion(valor), esperado);
 }
 for (const valor of [null,0,'','31','1.5','1,5','1e3/1','-1/2','1/-2','1/0','01/2','1/02','1000000001/1','1/1000000001','10000000000000000000/1','<img>']) {
  assert.throws(() => normalizarMinimoFormacion(valor), /minimo_formacion_invalido/u);
 }
});

test('editar el mínimo conserva los demás miembros y compara por el consumidor de méritos existente', async () => {
 const solicitudes = [];
 const editor = crearEditorBaremo({ cliente: { simular: async (s) => { solicitudes.push(s); return resultado(); } } });
 editor.cargar(ejemplo); await editor.comparar();
 editor.editarMinimoFormacion(0, '62/2');
 assert.equal(editor.estado().comparacion, null);
 const esperado = structuredClone(reglas); esperado.reglas[0].minimo_unidades = '31/1';
 assert.deepEqual(editor.estado().borrador, esperado);
 assert.deepEqual(ejemplo.reglas, reglas);
 await editor.comparar();
 assert.deepEqual(solicitudes.at(-2), { modo:'meritos', ejemplo_ref:ejemplo.referencia, reglas });
 assert.deepEqual(solicitudes.at(-1), { modo:'meritos', ejemplo_ref:ejemplo.referencia, reglas:esperado });
 const importado = leerReglas(editor.exportar());
 const otro = crearEditorBaremo({ cliente:{} }); otro.cargar(ejemplo, importado);
 assert.deepEqual(JSON.parse(otro.exportar()), esperado);
 editor.editarMinimoFormacion(0,'0/1'); assert.equal(editor.estado().borrador.reglas[0].minimo_unidades,'0/1');
 for (const indice of [1,2,-1,0.5,'0',99]) assert.throws(() => editor.editarMinimoFormacion(indice,'1/1'), /campo_invalido/u);
});

test('conserva el texto inválido, bloquea envío y exportación y permite corregirlo', async () => {
 let llamadas = 0;
 const editor = crearEditorBaremo({ cliente:{simular:async()=>{llamadas++;return resultado();}} });
 editor.cargar(ejemplo);
 for (const valor of ['-1/2','1/0','1/1000000001','1000000001/1','<img src=x>']) {
  assert.throws(()=>editor.editarMinimoFormacion(0,valor), /minimo_formacion_invalido/u);
  assert.equal(editor.estado().invalidos[ruta],valor);
  assert.equal(editor.estado().borrador.reglas[0].minimo_unidades,'20/1');
  assert.throws(()=>editor.exportar(), /campo_invalido/u);
  await editor.comparar(); assert.equal(llamadas,0);
 }
 editor.editarMinimoFormacion(0,'31/1'); assert.deepEqual(editor.estado().invalidos,{});
 await editor.comparar(); assert.equal(llamadas,2); assert.ok(editor.estado().comparacion);
});

test('editar el mínimo cancela el cálculo en curso e impide presentar su respuesta tardía', async () => {
 for (const valor of ['31/1','1/0']) {
  let resolver, signal;
  const editor = crearEditorBaremo({cliente:{simular:(_s,o)=>{signal=o.signal;return new Promise(r=>{resolver=r;});}}});
  editor.cargar(ejemplo); const vuelo=editor.comparar();
  if(valor==='1/0') assert.throws(()=>editor.editarMinimoFormacion(0,valor)); else editor.editarMinimoFormacion(0,valor);
  assert.equal(signal.aborted,true);resolver(resultado());await vuelo;
  assert.equal(editor.estado().comparacion,null);assert.equal(editor.estado().trabajando,false);
 }
});

test('una importación de mínimo ausente, no canónico o fuera de límites conserva el estado anterior', async () => {
 const editor=crearEditorBaremo({cliente:{simular:async()=>resultado()}});editor.cargar(ejemplo);await editor.comparar();
 const previo=editor.estado();
 for(const valor of [undefined,null,'62/2','0/7','-1/1','1/0','1000000001/1']) {
  const importadas=structuredClone(reglas);importadas.reglas[0].minimo_unidades=valor;
  assert.throws(()=>editor.cargar(ejemplo,importadas),/archivo_invalido/u);assert.deepEqual(editor.estado(),previo);
 }
});

for(const idioma of ['es','en']) test(`el campo sólo aparece en formación por horas y conserva error traducido (${idioma})`, async()=>{
 const textos=await cargarTextos('baremo-bolsa',{idioma});
 const editor=crearEditorBaremo({cliente:{}});editor.cargar(ejemplo);
 const inicial=renderizarBaremo(editor.estado(),{textos,ejemplos:[ejemplo]});
 assert.equal((inicial.match(/data-minimo-formacion/gu)??[]).length,1);
 assert.ok(inicial.includes(textos.traducir('editor.minimo_formacion')));
 assert.match(inicial,/data-minimo-formacion data-indice="0"/u);
 assert.match(inicial,/value="20\/1"/u);
 assert.throws(()=>editor.editarMinimoFormacion(0,'<img src=x>'));
 const html=renderizarBaremo({...editor.estado(),ayuda:true},{textos,ejemplos:[ejemplo]});
 assert.doesNotMatch(html,/<img/u);assert.match(html,/value="&lt;img src=x&gt;" aria-invalid="true"/u);
 assert.ok(html.includes(textos.traducir('editor.minimo_formacion_invalido')));
 assert.match(html,/disabled aria-describedby="baremo-activacion"/u);
 assert.deepEqual(textos.faltantes,[]);
 const ajeno=structuredClone(reglas);ajeno.reglas[0].familia='otros';
 editor.cargar({...ejemplo,reglas:ajeno});assert.doesNotMatch(renderizarBaremo(editor.estado(),{textos}),/data-minimo-formacion/u);
});
