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

test("conserva el recibo de asignación y abre el informe jurídico tras recuperar la unidad", () => {
  const confirmado = estado(8);
  confirmado.expediente.cabecera = [{ clave: "unidad", valor: "unidad:desarrollo:rrhh" }];
  const html = renderizarModuloContratacionTemporal(confirmado, {
    asignacionDisponible: true,
    informeJuridicoDisponible: true,
    reciboAsignacionConfirmado: {
      expediente_ref: "expediente:ct:asignacion",
      version_resultante: 8,
      recibo_ref: "recibo:ct:asignacion:001",
      confirmada_en: "2026-09-13T10:00:00Z",
    },
  });
  assert.match(html, /data-ct-asignacion-confirmada/u);
  assert.match(html, /recibo:ct:asignacion:001/u);
  assert.match(html, /data-ct-exp-informe-juridico/u);
  assert.doesNotMatch(html, /data-ct-exp-asignacion(?:[\s>])/u);
});

test("conserva el recibo y no vuelve a ofrecer asignación si falla el GET del detalle", () => {
  const pendiente = estado(7);
  const html = renderizarModuloContratacionTemporal(pendiente, {
    asignacionDisponible: true,
    informeJuridicoDisponible: true,
    reciboAsignacionConfirmado: {
      expediente_ref: "expediente:ct:asignacion",
      version_resultante: 8,
      recibo_ref: "recibo:ct:asignacion:pendiente-001",
      confirmada_en: "2026-09-13T10:00:00Z",
    },
  });
  assert.match(html, /data-ct-asignacion-confirmada/u);
  assert.match(html, /recibo:ct:asignacion:pendiente-001/u);
  assert.doesNotMatch(html, /data-ct-exp-asignacion(?:[\s>])/u);
  assert.doesNotMatch(html, /data-ct-exp-informe-juridico/u);
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
