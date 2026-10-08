import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosPersonal } from "./i18n.js?v=20261008-alta-rpt-circular-v4";

test.before(async () => { await prepararTextosPersonal(); });
import { montarRegistroB2 } from "./registro-b2.js?v=20261008-alta-rpt-circular-v4";
import { ErrorRegistroB2 } from "./registro-b2-cliente.js";

function raizFalsa() {
  class Nodo {
    constructor(d, tagName) { Object.assign(this, { ownerDocument:d,tagName,children:[],dataset:{},listeners:new Map(),attrs:new Map(),textContent:"",value:"" }); }
    append(...hijos) { for (const hijo of hijos) { hijo.parent=this; this.children.push(hijo); } }
    replaceChildren(...hijos) { this.children=[]; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children=this.parent.children.filter((n)=>n!==this); }
    setAttribute(k,v) { this.attrs.set(k,String(v)); }
    addEventListener(k,fn) { this.listeners.set(k,fn); }
    focus() {}
  }
  const d={createElement:(tag)=>new Nodo(d,tag)};
  return new Nodo(d,"root");
}
const nodos=(n)=>[n,...n.children.flatMap(nodos)];
const texto=(n)=>nodos(n).map((x)=>x.textContent).join(" ");
const completar=()=>new Promise((r)=>setImmediate(r));
const empleado=`emp_${"a".repeat(24)}`;
const relacion=`rel_${"r".repeat(24)}`;
const otraRelacion=`rel_${"s".repeat(24)}`;
function respuesta(q,incluir=true) {
  const corte={vigente_en:q.vigenteEn,conocido_en:q.conocidoEn};
  const traza={desde:"2024-01-01",version:1,registrada_en:"2024-01-01T10:00:00Z",acto_ref:"acto:uno",fuente_ref:"fuente:uno",fuente_version:1};
  const ficha={empleado_ref:q.empleadoRef,version:3,corte,eficacia_administrativa:false,firma_oficial:false,relaciones:[relacion,otraRelacion].map((ref)=>({relacion_ref:ref,estado:"vigente",traza})),ocupaciones:[],situaciones:[],servicios:[]};
  const resultado={ficha,preparacion_servicios:{empleado_ref:q.empleadoRef,version:3,corte,estado:"preparacion_sintetica",cobertura:"no_acreditada",eficacia_administrativa:false,firma_oficial:false,servicios:[]}};
  if (incluir) resultado.preparacion_rpt={empleado_ref:q.empleadoRef,version_ficha:3,corte:{...corte},esquema:"vec.personal.preparacion-relacion-rpt.v1",uso:"preparacion",cobertura:"no_acreditada",estado_rpt:"pendiente_fuente_rpt",relaciones:[
    {relacion_ref:relacion,estado:"finalizada",en_intervalo:false,traza:{...traza,version:2,hasta:"2026-09-30"}},
    {relacion_ref:otraRelacion,estado:"suspendida",en_intervalo:true,traza},
  ]};
  return resultado;
}
function montar(consultarFicha) {
  const raiz=raizFalsa();
  const montaje=montarRegistroB2({raiz,empleadoRef:empleado,cliente:{consultarFicha,listarVacantes:()=>{throw Error("consulta inesperada");}},reloj:()=>new Date("2026-10-01T10:00:00Z")});
  return {raiz,montaje};
}
const selector=(raiz)=>nodos(raiz).find((n)=>n.dataset.registroB2Relacion!==undefined);
const seleccionar=(raiz,ref)=>{const s=selector(raiz);s.value=ref;s.listeners.get("change")();};
const panel=(raiz)=>nodos(raiz).find((n)=>n.className==="panel"&&n.children[0]?.children.some((x)=>x.tagName==="h3"&&x.textContent==="Relación para Organización"));

test("RPT consume el DTO backend y muestra solo la relación seleccionada",async()=>{
  let llamadas=0;
  const {raiz}=montar((q)=>{llamadas++;return respuesta(q);});
  await completar();
  assert.match(texto(panel(raiz)),/Seleccione una relación/);
  seleccionar(raiz,relacion);
  assert.match(texto(panel(raiz)),/Finalizada/);
  assert.doesNotMatch(texto(panel(raiz)),/Suspendida|Vigente/);
  seleccionar(raiz,otraRelacion);
  assert.match(texto(panel(raiz)),/Suspendida/);
  assert.doesNotMatch(texto(panel(raiz)),/Finalizada/);
  assert.equal(llamadas,1);
});

test("ausencia RPT conserva CER y la ficha de un backend compatible",async()=>{
  const {raiz}=montar((q)=>respuesta(q,false));await completar();
  assert.equal(panel(raiz),undefined);
  assert.match(texto(raiz),/Servicios para revisión|Relaciones de servicio/);
});

test("RPT ajena o inválida retira ficha y ambas preparaciones",async()=>{
  for (const cambiar of [
    (m)=>({...m,empleado_ref:`emp_${"b".repeat(24)}`}),
    (m)=>({...m,version_ficha:4}),
    (m)=>({...m,corte:{...m.corte,vigente_en:"2026-09-30"}}),
    (m)=>({...m,corte:{...m.corte,conocido_en:"2026-10-01T00:00:00Z"}}),
    (m)=>({...m,cobertura:"completa"}),
    (m)=>({...m,estado_rpt:"ocupada"}),
    ()=>null,
  ]) {
    const {raiz}=montar((q)=>{const r=respuesta(q);r.preparacion_rpt=cambiar(r.preparacion_rpt);return r;});await completar();
    assert.doesNotMatch(texto(raiz),/Servicios para revisión|Relaciones de servicio|Relación para Organización/);
    assert.ok(nodos(raiz).some((n)=>n.attrs.get("role")==="alert"));
  }
});

test("revocación retira CER y RPT y bloquea un selector antiguo",async()=>{
  let revocado=false;
  const {raiz}=montar((q)=>{if(revocado)throw new ErrorRegistroB2("acceso_denegado",403);return respuesta(q);});
  await completar();seleccionar(raiz,relacion);const antiguo=selector(raiz);revocado=true;
  nodos(raiz).find((n)=>n.tagName==="form").listeners.get("submit")({preventDefault(){}});
  assert.equal(panel(raiz),undefined);await completar();antiguo.listeners.get("change")();
  assert.doesNotMatch(texto(raiz),/Servicios para revisión|Relaciones de servicio|Relación para Organización/);
});

test("cambiar empleado descarta la preparación tardía de otro sujeto",async()=>{
  let resolver;
  const {raiz,montaje}=montar((q)=>q.empleadoRef===empleado?new Promise((r)=>{resolver=()=>r(respuesta(q));}):respuesta(q,false));
  montaje.cambiarEmpleado(`emp_${"b".repeat(24)}`);await completar();resolver();await completar();
  assert.equal(panel(raiz),undefined);assert.match(texto(raiz),/Servicios para revisión/);
  montaje.desmontar();assert.equal(raiz.children.length,0);
});
