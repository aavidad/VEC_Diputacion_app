import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";
import {
  renderizarAlegaciones, renderizarLlamamientos, renderizarSeguimiento, renderizarSubsanaciones,
} from "./vistas/seguimiento-tramites.js";

test("seguimiento y trámites informan de ámbitos vacíos sin datos aparentes", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.solicitudes = [];
  datos.actividad = [];
  datos.llamamientos = [];
  datos.subsanaciones = [];
  datos.alegaciones = [];
  delete datos.posicion;

  const seguimiento = renderizarSeguimiento(datos, { expedienteSeleccionado: "DEMO-SOL-INEXISTENTE" });
  assert.match(seguimiento, /Sin expedientes en el ámbito autorizado/u);
  assert.match(seguimiento, /No hay acciones disponibles hasta que el servicio facilite un expediente autorizado/u);
  assert.doesNotMatch(seguimiento, /undefined|\[object Object\]|Descargar expediente/u);
  assert.match(renderizarLlamamientos(datos), /Sin llamamientos/u);
  assert.match(renderizarSubsanaciones(datos), /Sin subsanaciones/u);
  assert.match(renderizarAlegaciones(datos), /Sin alegaciones/u);
});
