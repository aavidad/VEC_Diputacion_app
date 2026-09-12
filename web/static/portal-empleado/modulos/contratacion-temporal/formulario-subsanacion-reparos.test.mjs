import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioSubsanacionReparos } from "./formulario-subsanacion-reparos.js";
import { MENSAJES_SUBSANACION_REPAROS_ES as textos } from "./i18n-subsanacion-reparos.js";
const contexto = Object.freeze({ expediente_ref: "expediente:subsanacion:001", version_esperada: 6 });
const recibo = Object.freeze({ esquema:"vec.contratacion-temporal.recibo-subsanacion-reparos.v1",operacion:"registrar_subsanacion",expediente_ref:"expediente:subsanacion:001",version_resultante:7,fase_resultante:"subsanacion_unidad",estado_resultante:"incidencia",recibo_ref:"recibo:subsanacion:001",auditoria_ref:"auditoria:subsanacion:001",evento_ref:"evento:subsanacion:001",actor_ref:"actor:subsanacion:001",registrada_en:"2026-09-13T08:00:00Z" });
test("subsanación confirma solo el recibo de servidor", async () => {
  const eventos=new Map(); const raiz={innerHTML:"",addEventListener:(n,f)=>eventos.set(n,f),removeEventListener(){},contains:()=>true,querySelector:()=>({}),replaceChildren(){this.innerHTML="";}};
  const confirmaciones=[];
  const desmontar=montarFormularioSubsanacionReparos({raiz,contexto,traducir:(k)=>textos[k]??k,generarClaveIdempotencia:()=>"123e4567-e89b-42d3-a456-426614174000",confirmarOperacion:()=>true,alConfirmar:(reciboConfirmado, contextoOriginal)=>confirmaciones.push([reciboConfirmado, contextoOriginal]),cliente:{registrarSubsanacionReparos:async()=>recibo}});
  await eventos.get("submit")({target:{closest:()=>({elements:{namedItem:()=>({value:"Corrección del reparo."})}})},preventDefault(){}}); assert.match(raiz.innerHTML,/Subsanación confirmada por el servidor/); assert.match(raiz.innerHTML,/Subsanación por la unidad/); desmontar();
  assert.deepEqual(confirmaciones, [[recibo, contexto]]);
});
test("remonta el recibo confirmado con su contexto original sin habilitar otro envío", () => {
  const eventos=new Map(); let llamadas=0; const raiz={innerHTML:"",addEventListener:(n,f)=>eventos.set(n,f),removeEventListener(){},contains:()=>true,querySelector:()=>({}),replaceChildren(){this.innerHTML="";}};
  const desmontar=montarFormularioSubsanacionReparos({raiz,contexto:{ expediente_ref: contexto.expediente_ref, version_esperada: 7 },reciboConfirmado:{ recibo, contexto },traducir:(k)=>textos[k]??k,cliente:{registrarSubsanacionReparos:()=>{llamadas+=1;}}});
  assert.match(raiz.innerHTML,/Recibo de subsanación/u); assert.doesNotMatch(raiz.innerHTML,/data-ct-subsanacion-form/u); assert.equal(eventos.has("submit"),true); assert.equal(llamadas,0); desmontar();
});
