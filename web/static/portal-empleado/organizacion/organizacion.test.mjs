import assert from "node:assert/strict";
import test from "node:test";
import { renderizarCuadro } from "../modulos/contratacion-temporal/componentes-expedientes.js";
import { API_ORGANIZACION, ESQUEMA_ORGANIZACION, FUENTE_RPT, filtrarUnidades, validarOrganizacion } from "./organizacion.js";

const unidad = (extra = {}) => ({ clave:"u-1", etiqueta:"Centro <uno>", tipo:"centro", ...extra });
const base = () => ({ data:{ esquema:ESQUEMA_ORGANIZACION, catalogo_id:"rpt", catalogo_version:1, catalogo_huella_sha256:"a".repeat(64), estado:"borrador", fuente_ref:"fuente-servidor", descripcion:"Procedencia declarada por el servidor.", unidades:[unidad()] } });
test("la consulta de organización es accesible desde el cuadro real, sin adaptador DEMO", () => {
  const html = renderizarCuadro({
    cuadro: { demostracion: false, indicadores: [], expedientes: [] },
    filtros: { texto: "", estado: "", fase: "" }, carga: "listo",
  }, (clave) => clave);
  assert.match(html, /href="\/portal-empleado\/organizacion\/"/);
  assert.match(html, /organizacion_referencia/);
  assert.doesNotMatch(html, /ct-exp-operativo-titulo/);
});
test("valida envelope, tipos y límites de organización", () => { assert.equal(validarOrganizacion(base()).unidades[0].clave, "u-1"); assert.throws(() => validarOrganizacion({ data:{ ...base().data, esquema:"otro" } }), /no válida/); assert.throws(() => validarOrganizacion({ data:{ ...base().data, unidades:Array.from({length:1001}, (_,i)=>unidad({clave:`u-${i}`})) } }), /no válida/); });
test("filtra por texto, padre y acentos sin alterar la fuente", () => { const datos=[unidad({clave:"padre",etiqueta:"Área de Informática",tipo:"delegacion"}), unidad({clave:"u-2",etiqueta:"Puesto técnico",tipo:"puesto_responsabilidad",adscripcion_clave:"padre"})]; assert.equal(filtrarUnidades(datos,"informatica","").length,2); assert.equal(filtrarUnidades(datos,"","delegacion").length,1); assert.equal(datos.length,2); });
test("expone endpoint real y no añade almacenamiento ni datos de demostración", async () => { let llamada; const body = new TextEncoder().encode(JSON.stringify(base())); const cliente = (await import("./organizacion.js")).crearCliente(async (url, options) => { llamada={url,options}; return { ok:true, body:{ getReader(){ let done=false; return { read:async()=>{ if(done)return {done:true}; done=true; return {done:false,value:body}; }, cancel:async()=>{} }; } } }; }); await cliente.obtener(); assert.equal(llamada.url, API_ORGANIZACION); assert.equal(llamada.options.credentials,"same-origin"); assert.equal(llamada.options.redirect,"error"); assert.equal(llamada.options.cache,"no-store"); });
