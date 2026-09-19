import assert from "node:assert/strict";
import test from "node:test";
import { crearEjecutorHTTPMarcajesCronos, RUTA_MARCAJE_PROPIO_CRONOS } from "./cliente-http-marcajes.js";

const comando = {tipo:"registrar_fichaje",movimiento:"entrada",clave_operacion:"op-cronos-0001"};
const recibo = {referencia:"recibo:cronos:00000000-0000-4000-8000-000000000001",instante_utc:"2026-09-19T12:00:00.123456Z",marcaje_original_ref:"marcaje:cronos:op-cronos-0001",replay:false};
test("marcaje sólo envía movimiento y clave al endpoint propio", async () => {
 let llamada;
 const ejecutar=crearEjecutorHTTPMarcajesCronos({fetchImpl:async (...args)=>{llamada=args;return {ok:true,json:async()=>({recibo})}}});
 const resultado=await ejecutar({...comando,empleado_ref:"no_enviar",instante_utc:"no_enviar",canal:"no_enviar"});
 assert.equal(llamada[0],RUTA_MARCAJE_PROPIO_CRONOS);
 assert.equal(llamada[1].credentials,"omit");
 assert.equal(llamada[1].mode,"same-origin");
 assert.equal(llamada[1].redirect,"error");
 assert.equal(llamada[1].cache,"no-store");
 assert.deepEqual(JSON.parse(llamada[1].body),{movimiento:"entrada",clave_operacion:"op-cronos-0001"});
 assert.deepEqual(resultado,{recibo});
});
test("reintento tras respuesta ambigua conserva la misma clave y recibo", async () => {
 const cuerpos=[];
 const ejecutar=crearEjecutorHTTPMarcajesCronos({fetchImpl:async (_ruta,opciones)=>{
  cuerpos.push(opciones.body);
  if(cuerpos.length===1) throw new Error("network");
  return {ok:true,json:async()=>({recibo:{...recibo,replay:true}})};
 }});
 await assert.rejects(ejecutar(comando));
 const recuperado=await ejecutar(comando);
 assert.equal(cuerpos[0],cuerpos[1]);
 assert.equal(recuperado.recibo.referencia,recibo.referencia);
 assert.equal(recuperado.recibo.instante_utc,recibo.instante_utc);
});
test("rechaza destino ajeno y comandos sin clave sin hacer HTTP", async () => {
 let llamadas=0;
 const fetchImpl=async()=>{llamadas++};
 assert.throws(()=>crearEjecutorHTTPMarcajesCronos({ruta:"https://otro.invalid/",fetchImpl}));
 const ejecutar=crearEjecutorHTTPMarcajesCronos({fetchImpl});
 await assert.rejects(ejecutar({...comando,clave_operacion:""}));
 await assert.rejects(ejecutar({...comando,movimiento:"inventado"}));
 assert.equal(llamadas,0);
});
test("rechaza recibo de otra operación y nunca propaga error del servidor", async () => {
 const ajeno=crearEjecutorHTTPMarcajesCronos({fetchImpl:async()=>({ok:true,json:async()=>({recibo:{...recibo,marcaje_original_ref:"marcaje:cronos:otra-operacion"}})})});
 await assert.rejects(ajeno(comando),{code:"cronos_recibo_no_confiable"});
 const fallo=crearEjecutorHTTPMarcajesCronos({fetchImpl:async()=>({ok:false,status:409,json:async()=>({error:"detalle_privado"})})});
 await assert.rejects(fallo(comando),{code:"cronos_clave_operacion_conflicto",message:"cronos_clave_operacion_conflicto"});
});
