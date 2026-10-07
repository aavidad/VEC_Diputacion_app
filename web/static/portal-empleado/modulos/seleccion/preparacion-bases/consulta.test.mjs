import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { crearLectorBasesHTTP, RUTA_CONSULTA_BASES } from './cliente-http.js';
import { leerConsulta, validarSelector } from './contrato-http.js';

const cli = JSON.parse(await readFile(new URL('./testdata/preparacion-completa.json', import.meta.url)));
const h = 'a'.repeat(64), selector = { modo: 'actual', preparacion_ref: 'preparacion:sintetica', revision: 0, huella_material_sha256: '' };
function respuesta() {
  return { estado: 'obtenida', preparacion: { ambito: { organizacion_ref: 'org_' + 'a'.repeat(16) },
    estado: { preparacion_ref: selector.preparacion_ref, revision: 1, huella_material_sha256: h },
    material: { contenido: cli.preparacion.material_propuesto.contenido,
      referencias: Object.entries(cli.preparacion.material_propuesto.referencias).map(([campo, referencia]) => ({ campo, referencia })) } },
  pendientes: cli.preparacion.pendientes,
  recibo: { recibo_ref: 'recibo:original', historia_ref: 'historia:sintetica', auditoria_ref: 'auditoria:original', evento_ref: 'evento:sintetico', huella_intencion_sha256: h, confirmada_en: '2026-10-01T12:00:00Z' },
  acceso: { decision_ref: 'decision:sintetica', consumo_huella_sha256: h, auditoria_ref: 'auditoria:consulta', recibo_ref: 'recibo:consulta', correlacion_ref: 'correlacion:sintetica', accedida_en: '2026-10-03T12:00:00Z' } };
}
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto, null, 2) + '\n');
test('ruta nominal fija, selectores mínimos, recibos separados y bytes exactos', async () => {
  const dto = respuesta(), bytes = codificar(dto); let peticion;
  const lector = crearLectorBasesHTTP({ fetchImpl: async (ruta, opciones) => {
    peticion = opciones; assert.equal(ruta, RUTA_CONSULTA_BASES);
    return new Response(bytes, { headers: { 'Content-Type': 'application/json; charset=utf-8' } });
  } });
  const r = await lector.consultar(selector); assert.deepEqual(r.bytes, bytes); assert.deepEqual(r.dto, dto);
  assert.equal(r.dto.recibo.recibo_ref, 'recibo:original'); assert.equal(r.dto.acceso.recibo_ref, 'recibo:consulta');
  for (const [k,v] of Object.entries({ credentials:'same-origin',mode:'same-origin',cache:'no-store',redirect:'error',referrerPolicy:'no-referrer' })) assert.equal(peticion[k],v);
  assert.deepEqual(JSON.parse(peticion.body), selector);
  assert.deepEqual(leerConsulta(bytes, { ...selector, modo:'exacta',revision:1,huella_material_sha256:h }), dto);
});
test('rechaza estado falso, selector cruzado, circuitos suprimidos y evidencias incompletas', () => {
  for (const cambiar of [d => d.estado='guardada', d => d.preparacion.estado.preparacion_ref='preparacion:ajena',
    d => d.pendientes=d.pendientes.filter(x=>x.campo!=='reglas_baremacion'), d => d.pendientes=d.pendientes.filter(x=>x.campo!=='acto_aprobacion'),
    d => d.acceso.recibo_ref='', d => d.recibo.confirmada_en='0001-01-01T00:00:00Z', d => d.preparacion.material.aprobado=true]) {
    const d=respuesta(); cambiar(d); assert.throws(()=>leerConsulta(codificar(d),selector), /respuesta_incompatible/u);
  }
  assert.throws(()=>leerConsulta(codificar(respuesta()),{...selector,modo:'exacta',revision:2,huella_material_sha256:h}),/respuesta_incompatible/u);
  assert.throws(()=>validarSelector({...selector,actor:'inventado'}),/selector_invalido/u);
  assert.throws(()=>leerConsulta(new TextEncoder().encode('{"estado":"obtenida","estado":"aprobada"}'),selector),/respuesta_incompatible/u);
});
test('denegación, ausencia, conflicto, indisponibilidad y formato no muestran datos', async () => {
  for (const [estado,codigo] of [[401,'acceso_denegado'],[403,'acceso_denegado'],[404,'no_encontrada'],[409,'version_en_conflicto'],[503,'servicio_no_disponible']]) {
    let leido=false; const lector=crearLectorBasesHTTP({fetchImpl:async()=>({status:estado,ok:false,get body(){leido=true;return null;}})});
    await assert.rejects(lector.consultar(selector),e=>e.codigo===codigo); assert.equal(leido,false);
  }
  for (const headers of [{'Content-Type':'text/html'},{'Content-Type':'application/json','Content-Length':'5000000'}]) {
    const lector=crearLectorBasesHTTP({fetchImpl:async()=>new Response(codificar(respuesta()),{headers})});
    await assert.rejects(lector.consultar(selector),e=>e.codigo==='respuesta_incompatible');
  }
});
test('cancelación y plazo rechazan incluso fetch que entregue tarde tras abortar', async () => {
  const controlador=new AbortController(); let devolver;
  const lector=crearLectorBasesHTTP({fetchImpl:()=>new Promise(r=>{devolver=r;})});
  const consulta=lector.consultar(selector,{signal:controlador.signal}); controlador.abort();
  devolver(new Response(codificar(respuesta()),{headers:{'Content-Type':'application/json'}}));
  await assert.rejects(consulta,e=>e.codigo==='operacion_abortada');
  const lento=crearLectorBasesHTTP({plazoMs:1,fetchImpl:async()=>{await new Promise(r=>setTimeout(r,10));return new Response(codificar(respuesta()),{headers:{'Content-Type':'application/json'}});}});
  await assert.rejects(lento.consultar(selector),e=>e.codigo==='servicio_no_disponible');
});
