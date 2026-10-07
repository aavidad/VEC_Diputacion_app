import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosPersonal } from "./i18n.js?v=20261007-pantallas-textos-final-v1";

test.before(async () => { await prepararTextosPersonal(); });
import { montarRegistroB2 } from "./registro-b2.js";
import { ErrorRegistroB2 } from "./registro-b2-cliente.js";

function raizFalsa() {
  class Nodo {
    constructor(d,tagName) { Object.assign(this,{ownerDocument:d,tagName,children:[],dataset:{},listeners:new Map(),attrs:new Map(),textContent:"",value:""}); }
    append(...hijos) { for(const hijo of hijos){hijo.parent=this;this.children.push(hijo);} }
    replaceChildren(...hijos) { this.children=[];this.append(...hijos); }
    remove() { if(this.parent)this.parent.children=this.parent.children.filter((n)=>n!==this); }
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
  const resultado={ficha,
    preparacion_servicios:{empleado_ref:q.empleadoRef,version:3,corte,estado:"preparacion_sintetica",cobertura:"no_acreditada",eficacia_administrativa:false,firma_oficial:false,servicios:[]},
    preparacion_rpt:{empleado_ref:q.empleadoRef,version_ficha:3,corte,esquema:"vec.personal.preparacion-relacion-rpt.v1",uso:"preparacion",cobertura:"no_acreditada",estado_rpt:"pendiente_fuente_rpt",relaciones:[relacion,otraRelacion].map((ref)=>({relacion_ref:ref,estado:"vigente",en_intervalo:true,traza}))},
  };
  if(incluir)resultado.preparacion_carrera={empleado_ref:q.empleadoRef,version:3,corte:{...corte},esquema:"vec.personal.preparacion-antecedentes-carrera.v1",alcance:"preparacion",pendientes:["fuente_institucional","cobertura_antecedentes","politica_carrera"],relaciones:[relacion,otraRelacion].map((ref,i)=>({relacion_ref:ref,estado:i===0?"finalizada":"suspendida",regimen_ref:"regimen:uno",regimen:`Régimen sintético ${i}`,traza,historia:[],pendientes:["grupo_subgrupo","puesto_nivel_m","grado_personal_h"],servicios:[],situaciones:[]}))};
  return resultado;
}
function montar(consultarFicha) {
  const raiz=raizFalsa();
  const montaje=montarRegistroB2({raiz,empleadoRef:empleado,cliente:{consultarFicha,listarVacantes:()=>{throw Error("consulta inesperada");}},reloj:()=>new Date("2026-10-01T10:00:00Z")});
  return {raiz,montaje};
}
const selector=(raiz)=>nodos(raiz).find((n)=>n.dataset.registroB2Relacion!==undefined);
const seleccionar=(raiz,ref)=>{const s=selector(raiz);s.value=ref;s.listeners.get("change")();};
const panel=(raiz)=>nodos(raiz).find((n)=>n.className==="panel"&&n.children[0]?.children.some((x)=>x.tagName==="h3"&&x.textContent==="Antecedentes para Carrera"));

test("Carrera conserva pendientes globales y filtra la relación sin recalcular antecedentes",async()=>{
  let llamadas=0;let original;
  const {raiz}=montar((q)=>{llamadas++;original=respuesta(q);return original;});await completar();
  assert.match(texto(panel(raiz)),/Falta confirmar la procedencia de estos antecedentes/);
  assert.match(texto(raiz),/Seleccione la relación de servicio/);
  assert.doesNotMatch(texto(panel(raiz)),/Régimen sintético/);
  seleccionar(raiz,relacion);
  assert.match(texto(panel(raiz)),/Régimen sintético 0 · Finalizada/);
  assert.doesNotMatch(texto(panel(raiz)),/Régimen sintético 1|Vigente/);
  seleccionar(raiz,otraRelacion);
  assert.match(texto(panel(raiz)),/Régimen sintético 1 · Suspendida/);
  assert.doesNotMatch(texto(panel(raiz)),/Régimen sintético 0/);
  assert.equal(llamadas,1);assert.equal(original.preparacion_carrera.relaciones.length,2);
});

test("ausencia de Carrera conserva CER, RPT y la ficha compatible",async()=>{
  const {raiz}=montar((q)=>respuesta(q,false));await completar();
  assert.equal(panel(raiz),undefined);
  assert.match(texto(raiz),/Servicios para revisión/);
  assert.match(texto(raiz),/Relación para Organización/);
});

test("modelo ajeno o inválido en una relación no seleccionada retira toda la ficha",async()=>{
  for(const cambiar of [
    (m)=>({...m,empleado_ref:`emp_${"b".repeat(24)}`}),
    (m)=>({...m,version:4}),
    (m)=>({...m,corte:{...m.corte,vigente_en:"2026-09-30"}}),
    (m)=>({...m,corte:{...m.corte,conocido_en:"2026-10-01T00:00:00Z"}}),
    (m)=>({...m,alcance:"acreditado"}),
    (m)=>({...m,firma_oficial:true}),
    (m)=>({...m,relaciones:[m.relaciones[0],{...m.relaciones[1],servicios:null}]}),
    ()=>null,
  ]) {
    const {raiz}=montar((q)=>{const r=respuesta(q);r.preparacion_carrera=cambiar(r.preparacion_carrera);return r;});await completar();
    assert.doesNotMatch(texto(raiz),/Servicios para revisión|Relaciones de servicio|Relación para Organización|Antecedentes para Carrera/);
    assert.ok(nodos(raiz).some((n)=>n.attrs.get("role")==="alert"));
  }
});

test("revocación elimina las tres preparaciones y bloquea el selector anterior",async()=>{
  let revocado=false;
  const {raiz}=montar((q)=>{if(revocado)throw new ErrorRegistroB2("acceso_denegado",403);return respuesta(q);});
  await completar();seleccionar(raiz,relacion);const antiguo=selector(raiz);revocado=true;
  nodos(raiz).find((n)=>n.tagName==="form").listeners.get("submit")({preventDefault(){}});
  assert.equal(panel(raiz),undefined);await completar();antiguo.listeners.get("change")();
  assert.doesNotMatch(texto(raiz),/Servicios para revisión|Relaciones de servicio|Relación para Organización|Antecedentes para Carrera/);
});

test("cambiar empleado descarta los antecedentes tardíos de otro sujeto",async()=>{
  let resolver;
  const {raiz,montaje}=montar((q)=>q.empleadoRef===empleado?new Promise((r)=>{resolver=()=>r(respuesta(q));}):respuesta(q,false));
  montaje.cambiarEmpleado(`emp_${"b".repeat(24)}`);await completar();resolver();await completar();
  assert.equal(panel(raiz),undefined);assert.match(texto(raiz),/Servicios para revisión/);
  montaje.desmontar();assert.equal(raiz.children.length,0);
});
