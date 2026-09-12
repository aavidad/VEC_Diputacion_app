import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioSubsanacionReparos } from "./formulario-subsanacion-reparos.js";
import { MENSAJES_SUBSANACION_REPAROS_ES as textos } from "./i18n-subsanacion-reparos.js";
test("subsanación confirma solo el recibo de servidor", async () => {
  const eventos=new Map(); const raiz={innerHTML:"",addEventListener:(n,f)=>eventos.set(n,f),removeEventListener(){},contains:()=>true,querySelector:()=>({}),replaceChildren(){this.innerHTML="";}};
  const desmontar=montarFormularioSubsanacionReparos({raiz,contexto:{expediente_ref:"expediente:subsanacion:001",version_esperada:6},traducir:(k)=>textos[k]??k,generarClaveIdempotencia:()=>"123e4567-e89b-42d3-a456-426614174000",confirmarOperacion:()=>true,cliente:{registrarSubsanacionReparos:async()=>({esquema:"vec.contratacion-temporal.recibo-subsanacion-reparos.v1",operacion:"registrar_subsanacion",expediente_ref:"expediente:subsanacion:001",version_resultante:7,fase_resultante:"subsanacion_unidad",estado_resultante:"incidencia",recibo_ref:"recibo:subsanacion:001",auditoria_ref:"auditoria:subsanacion:001",evento_ref:"evento:subsanacion:001",actor_ref:"actor:subsanacion:001",registrada_en:"2026-09-13T08:00:00Z"})}});
  await eventos.get("submit")({target:{closest:()=>({elements:{namedItem:()=>({value:"Corrección del reparo."})}})},preventDefault(){}}); assert.match(raiz.innerHTML,/Subsanación confirmada por el servidor/); assert.match(raiz.innerHTML,/Subsanación por la unidad/); desmontar();
});
