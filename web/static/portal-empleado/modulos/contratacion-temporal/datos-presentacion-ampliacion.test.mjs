import assert from "node:assert/strict";
import test from "node:test";

import { enriquecerTareasPresentacion } from "./datos-presentacion-ampliacion.js";

function tarea(tarea_ref) {
  return { tarea_ref, paneles: [] };
}

function panel(tareas, tareaRef, panelRef) {
  return tareas.find((item) => item.tarea_ref === tareaRef).paneles
    .find((item) => item.panel_ref === panelRef);
}

test("la ampliación presenta aviso, respuesta y resolución sintéticos sin efectos jurídicos", () => {
  const tareas = enriquecerTareasPresentacion([
    tarea("tarea-iniciar-llamamiento"),
    tarea("tarea-resultado-llamamiento"),
    tarea("tarea-formalizacion"),
    tarea("tarea-incorporacion"),
    tarea("tarea-ginpix"),
    tarea("tarea-seguimiento"),
  ]);
  const texto = JSON.stringify(tareas);

  assert.match(texto, /Aviso local/u);
  assert.match(texto, /Preparado; sin entrega acreditada/u);
  assert.match(texto, /Respuesta declarada/u);
  assert.match(texto, /Aceptación declarada sintética/u);
  assert.match(texto, /Resolución manual de respuesta/u);
  assert.match(texto, /sin firma ni eficacia/u);
  assert.match(texto, /Ficha manual/u);
  assert.match(texto, /sin incorporación confirmada/u);
  assert.match(texto, /no acredita cese ni cierre/u);
  assert.doesNotMatch(texto, /\bEntregado\b/u);
  assert.doesNotMatch(texto, /Aceptación en plazo/u);
  assert.doesNotMatch(texto, /Cese programable/u);
  const historial = panel(tareas, "tarea-resultado-llamamiento", "panel-historial-candidatura");
  assert.deepEqual(historial.filas[2].celdas.slice(1, 3), [
    "Resolución manual de respuesta",
    "Registrada sintéticamente; sin firma ni eficacia",
  ]);
  const formalizacion = panel(tareas, "tarea-formalizacion", "panel-subpasos-formalizacion");
  assert.equal(formalizacion.filas.length, 6);
  assert.deepEqual(
    formalizacion.filas.map((fila) => fila.celdas[1]),
    [
      "Informe definitivo",
      "Resolución de nombramiento",
      "Diligencia",
      "Toma de posesión",
      "Notificación a la persona interesada",
      "Comunicación al centro",
    ],
  );
  assert.doesNotMatch(JSON.stringify(formalizacion), /Resolución manual/u);
  assert.match(
    JSON.stringify(panel(tareas, "tarea-formalizacion", "panel-vista-previa-resolucion")),
    /Borrador preparatorio; pendiente de portafirmas/u,
  );
  assert.deepEqual(
    panel(tareas, "tarea-ginpix", "panel-historial-ginpix").filas[1].celdas,
    ["—", "Pendiente", "Transmisión", "No iniciada", "—"],
  );
});
