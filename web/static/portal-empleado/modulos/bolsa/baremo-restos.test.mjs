import assert from 'node:assert/strict';
import test from 'node:test';
import {readFile} from 'node:fs/promises';
import {crearEditorBaremo, comprobarCatalogoRestos, errorRestosRegla} from './baremo-editor.js';
import {renderizarBaremo} from './baremo-vista.js';
import {cargarTextos} from '../../../comun/textos.js';
const reglas=JSON.parse(await readFile(new URL('../../../../../internal/modules/bolsa/application/simulacionbaremo/testdata/reglas_a.json',import.meta.url),'utf8'));
const catalogo=JSON.parse(await readFile(new URL('../../../catalogos/baremo-restos-v1.json',import.meta.url),'utf8'));
const ejemplo={referencia:'experiencia_sintetica_v1',modo:'experiencia',reglas};
const resultado=()=>({resultado:{estado:'completado',total:'101667',secciones:[]}});
const ruta='["reglas_experiencia",0,"restos","modo"]';

test('el catálogo de restos reproduce los modos, fronteras y redondeos del compilador Go V1',async()=>{
 const tipos=await readFile(new URL('../../../../../internal/modules/bolsa/domain/reglasbaremo/tipos.go',import.meta.url),'utf8');
 const compilador=await readFile(new URL('../../../../../internal/modules/bolsa/domain/calculoexperiencia/compilacion.go',import.meta.url),'utf8');
 const modos=new Map([...tipos.matchAll(/(Restos\w+)\s+ModoRestos\s*=\s*"([^"]+)"/gu)].map(m=>[m[1],m[2]]));
 const semantica=compilador.slice(compilador.indexOf('func semanticaRestosCompilableV1('),compilador.indexOf('func validarGrupoCompilableV1('));
 const fronteras=new Map([...semantica.matchAll(/case reglasbaremo\.(Restos\w+):\s*return semanticaRestosV1\{(fronteraRestos\w+V1),/gu)].map(m=>[modos.get(m[1]),m[2]]));
 assert.equal(fronteras.size,4);
 assert.deepEqual(catalogo.opciones.map(o=>o.modo).sort(),[...modos.values()].sort());
 for(const opcion of comprobarCatalogoRestos(catalogo).opciones){
  assert.deepEqual(opcion.momentos_redondeo,fronteras.get(opcion.modo)==='fronteraRestosReglaV1'?['regla']:['periodo','regla']);
 }
 assert.match(compilador,/semantica\.frontera == fronteraRestosReglaV1 &&\s*momento == reglasbaremo\.RedondearPorPeriodo/u);
 assert.match(compilador,/regla\.MaximoUnidades\(\)\.EstaLimitado\(\) &&\s*momento == reglasbaremo\.RedondearPorPeriodo/u);
 assert.match(compilador,/regla\.UnidadTemporal\(\)\.UnidadBase\(\) != reglasbaremo\.UnidadTemporalDia/u);
});

test('rechaza catálogos con la misma versión pero matriz o identidades contradictorias',()=>{
 for(const alterar of [c=>c.version=2,c=>c.motor='otro',c=>c.contrato_reglas='otro',c=>c.unidad_base='mes',c=>c.tope_unidades_por_periodo=true,c=>c.opciones.pop(),c=>c.opciones[0].modo='futuro',c=>c.opciones[1].modo=c.opciones[0].modo,c=>c.opciones[0].etiqueta='jornada_integra']){
  const copia=structuredClone(catalogo);alterar(copia);assert.throws(()=>comprobarCatalogoRestos(copia),/catalogo_restos_no_disponible/u);
 }
 for(let i=0;i<4;i++)for(const momentos of [[],['total'],['regla','regla'],catalogo.opciones[i].momentos_redondeo.length===1?['periodo','regla']:['regla']]){
  const copia=structuredClone(catalogo);copia.opciones[i].momentos_redondeo=momentos;assert.throws(()=>comprobarCatalogoRestos(copia),/catalogo_restos_no_disponible/u);
 }
});

test('editar restos conserva redondeo y demás miembros, invalida y compara ambas versiones con el mismo consumidor',async()=>{
 const llamadas=[];const copia=structuredClone(catalogo);
 const editor=crearEditorBaremo({catalogoRestos:copia,cliente:{simular:async s=>{llamadas.push(s);return resultado();}}});
 copia.opciones.length=0;editor.cargar(ejemplo);await editor.comparar();editor.editarRestos(0,'descartar_por_regla');
 assert.equal(editor.estado().comparacion,null);
 const esperado=structuredClone(reglas);esperado.reglas_experiencia[0].restos.modo='descartar_por_regla';
 assert.deepEqual(editor.estado().borrador,esperado);assert.deepEqual(ejemplo.reglas,reglas);
 await editor.comparar();assert.deepEqual(llamadas.at(-2),{modo:'experiencia',ejemplo_ref:ejemplo.referencia,reglas});
 assert.deepEqual(llamadas.at(-1),{modo:'experiencia',ejemplo_ref:ejemplo.referencia,reglas:esperado});
 const exportado=JSON.parse(editor.exportar());const otro=crearEditorBaremo({catalogoRestos:catalogo,cliente:{}});otro.cargar(ejemplo,exportado);assert.deepEqual(JSON.parse(otro.exportar()),esperado);
 const estado=editor.estado();estado.catalogoRestos.opciones.length=0;assert.equal(editor.estado().catalogoRestos.opciones.length,4);
});

test('una elección incompatible se conserva como inválida, bloquea envío y exportación y no cambia redondeo',async()=>{
 let llamadas=0;const editor=crearEditorBaremo({catalogoRestos:catalogo,cliente:{simular:async()=>{llamadas++;return resultado();}}});
 const importadas=structuredClone(reglas);importadas.reglas_experiencia[0].redondeo.momento='periodo';importadas.reglas_experiencia[0].maximo_unidades={modo:'sin_limite'};
 editor.cargar(ejemplo,importadas);
 for(const modo of ['acumular_por_regla','descartar_por_regla','futuro']){
  assert.throws(()=>editor.editarRestos(0,modo),/restos_(redondeo_incompatible|no_disponibles)/u);
  assert.equal(editor.estado().invalidos[ruta],modo);
  assert.deepEqual(editor.estado().borrador,importadas);
  editor.editar(['reglas_experiencia',0,'puntos_por_unidad'],'200000');
  assert.equal(editor.estado().invalidos[ruta],modo);
  assert.throws(()=>editor.exportar(),/campo_invalido/u);await editor.comparar();assert.equal(llamadas,0);
  editor.editar(['reglas_experiencia',0,'puntos_por_unidad'],reglas.reglas_experiencia[0].puntos_por_unidad);
 }
 editor.editarRestos(0,'descartar_por_periodo');assert.deepEqual(editor.estado().invalidos,{});
 await editor.comparar();assert.equal(llamadas,2);assert.equal(editor.estado().borrador.reglas_experiencia[0].redondeo.momento,'periodo');
});

test('las incompatibilidades importadas permanecen intactas y bloquean comparación/exportación con un catálogo positivo',async()=>{
 for(const cambiar of [r=>r.restos.modo='futuro',r=>{r.restos.modo='acumular_por_regla';r.redondeo.momento='periodo';},r=>r.redondeo.momento='periodo',r=>r.unidad_temporal.unidad_base='mes',r=>r.redondeo.momento='total']){
  let llamadas=0;const editor=crearEditorBaremo({catalogoRestos:catalogo,cliente:{simular:async()=>{llamadas++;return resultado();}}});
  const importadas=structuredClone(reglas);cambiar(importadas.reglas_experiencia[0]);editor.cargar(ejemplo,importadas);await editor.comparar();
  assert.equal(llamadas,0);assert.ok(editor.estado().error.startsWith('restos_'));assert.throws(()=>editor.exportar(),/restos_/u);assert.deepEqual(editor.estado().borrador,importadas);
 }
 const regla=structuredClone(reglas.reglas_experiencia[0]);regla.redondeo.momento='periodo';
 assert.equal(errorRestosRegla(regla,catalogo),'restos_tope_incompatible');
});

test('sin catálogo válido se cierra sólo la nueva edición y se conserva el consumidor anterior',async()=>{
 for(const cat of [null,{...catalogo,version:2},{...catalogo,tope_unidades_por_periodo:true}]){
  let llamadas=0;const editor=crearEditorBaremo({catalogoRestos:cat,cliente:{simular:async()=>{llamadas++;return resultado();}}});editor.cargar(ejemplo);
  assert.throws(()=>editor.editarRestos(0,'descartar_por_regla'),/catalogo_restos_no_disponible/u);
  assert.deepEqual(JSON.parse(editor.exportar()),reglas);await editor.comparar();assert.equal(llamadas,2);
 }
 const meritos=JSON.parse(await readFile(new URL('../../../../../internal/modules/bolsa/application/simulacionbaremo/testdata/meritos_reglas_a.json',import.meta.url),'utf8'));
 const editor=crearEditorBaremo({catalogoRestos:catalogo,cliente:{}});editor.cargar({referencia:'meritos',modo:'meritos',reglas:meritos});
 assert.throws(()=>editor.editarRestos(0,'descartar_por_regla'),/campo_invalido/u);assert.deepEqual(JSON.parse(editor.exportar()),meritos);
});

test('un cambio de restos cancela la respuesta tardía, incluida una selección inválida',async()=>{
 for(const modo of ['descartar_por_regla','futuro']){
  let resolver,signal;const editor=crearEditorBaremo({catalogoRestos:catalogo,cliente:{simular:(_s,o)=>{signal=o.signal;return new Promise(r=>resolver=r);}}});editor.cargar(ejemplo);const vuelo=editor.comparar();
  if(modo==='futuro')assert.throws(()=>editor.editarRestos(0,modo));else editor.editarRestos(0,modo);
  assert.equal(signal.aborted,true);resolver(resultado());await vuelo;assert.equal(editor.estado().comparacion,null);assert.equal(editor.estado().trabajando,false);
 }
});

for(const idioma of ['es','en'])test(`vista traducida de restos conserva selección/pista/error y cierre sin catálogo (${idioma})`,async()=>{
 const textos=await cargarTextos('baremo-bolsa',{idioma});
 const editor=crearEditorBaremo({catalogoRestos:catalogo,cliente:{}});editor.cargar(ejemplo);
 const html=renderizarBaremo(editor.estado(),{textos,ejemplos:[ejemplo]});
 assert.match(html,/data-restos-modo data-indice="0"/u);assert.equal((html.match(/data-restos-modo/gu)||[]).length,1);
 assert.ok(html.includes(textos.traducir('editor.restos')));assert.ok(html.includes(textos.traducir('editor.restos_conservar_exactos_explicacion')));
 assert.throws(()=>editor.editarRestos(0,'futuro'));
 const invalido=renderizarBaremo({...editor.estado(),ayuda:true},{textos,ejemplos:[ejemplo]});
 assert.match(invalido,/value="futuro" selected disabled/u);assert.match(invalido,/data-restos-error="restos_no_disponibles"/u);assert.match(invalido,/data-accion="exportar" disabled/u);
 const cerrado=crearEditorBaremo({cliente:{}});cerrado.cargar(ejemplo);assert.match(renderizarBaremo(cerrado.estado(),{textos}),/data-restos-modo[^>]* disabled/u);
 assert.deepEqual(textos.faltantes,[]);assert.match(html,/disabled aria-describedby="baremo-activacion"/u);
});
