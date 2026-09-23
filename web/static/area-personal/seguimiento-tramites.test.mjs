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
  assert.match(renderizarLlamamientos(datos), /Pendiente de integración/u);
  assert.match(renderizarSubsanaciones(datos), /Sin subsanaciones/u);
  assert.match(renderizarAlegaciones(datos), /Sin alegaciones/u);
});

test("mi bolsa muestra tarjetas propias, provisionalidad y paginación", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  const participaciones = Array.from({ length: 7 }, (_, indice) => ({ bolsa: `bolsa:prueba:${indice}`, categoria: "Auxiliar", version: 3, orden_inicial: indice + 1, total_instantanea: 87, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00Z", vigente_hasta: null }));
  const vista = renderizarLlamamientos(datos, { participaciones, paginaParticipaciones: 1, fuenteBolsa: "ejemplo" });
  assert.match(vista, /Datos de ejemplo\.<\/strong> El acceso del candidato con DNIe o certificado está pendiente de desarrollo en VEC\./u);
  assert.match(vista, /Mi número de orden[\s\S]*1 de 87/u);
  assert.match(vista, /Versión de la bolsa/u);
  assert.match(vista, /Mostrando 1 a 6 de 7/u);
  assert.match(vista, /Identificarse con certificado no firma documentos\./u);
  assert.match(vista, /Pendiente de integración/u);
  assert.doesNotMatch(vista, /Ensayar pausa|Ensayar reactivación/u);
});
