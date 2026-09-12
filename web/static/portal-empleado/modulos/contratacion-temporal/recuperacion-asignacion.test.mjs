import assert from "node:assert/strict";
import test from "node:test";

import { renderizarModuloContratacionTemporal } from "./vista-expedientes.js";

function estado(version = 7) {
  return {
    vista: "expediente", carga: "listo", tipo_mensaje: "info", mensaje_clave: "estado_expediente_listo",
    expediente: { expediente_ref: "expediente:ct:asignacion", version, cabecera: [], tareas: [], fases: [] },
    cuadro: { expedientes: [{ expediente_ref: "expediente:ct:asignacion", fase_clave: "asignacion_unidad", estado_clave: "en_curso", version }] },
  };
}

test("recupera el contenedor de asignación sólo para la misma fase y versión", () => {
  const opciones = { asignacionDisponible: true };
  const primero = renderizarModuloContratacionTemporal(estado(), opciones);
  const segundo = renderizarModuloContratacionTemporal(estado(), opciones);
  assert.match(primero, /data-ct-exp-asignacion/);
  assert.equal((segundo.match(/data-ct-exp-asignacion/g) ?? []).length, 1);
  const divergente = estado(8);
  divergente.cuadro.expedientes[0].version = 7;
  assert.doesNotMatch(renderizarModuloContratacionTemporal(divergente, opciones), /data-ct-exp-asignacion/);
});


test("no ofrece asignación durante carga, fuera de fase o con operación cerrada", () => {
 for (const cambiar of [
   e => { e.carga = "cargando"; },
   e => { e.cuadro.expedientes[0].fase_clave = "analisis"; },
   e => { e.cuadro.expedientes[0].estado_clave = "cerrado"; },
 ]) {
  const e = estado(); cambiar(e);
  assert.doesNotMatch(renderizarModuloContratacionTemporal(e, {asignacionDisponible:true}), /data-ct-exp-asignacion/);
 }
 assert.doesNotMatch(renderizarModuloContratacionTemporal(estado(),{asignacionDisponible:false}), /data-ct-exp-asignacion/);
});
