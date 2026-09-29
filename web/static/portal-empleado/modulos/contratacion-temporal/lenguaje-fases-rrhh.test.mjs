import assert from "node:assert/strict";
import test from "node:test";
import {
  FASES_RRHH,
  formatearFaseRRHH,
  nombreEstadoRRHH,
  nombreFaseRRHH,
  obtenerFaseRRHH,
} from "./lenguaje-fases-rrhh.js";

test("las tareas intermedias conservan una de las ocho fases visibles de RRHH", () => {
  assert.equal(FASES_RRHH.length, 8);
  assert.deepEqual(FASES_RRHH.map(({ orden }) => orden), [1, 2, 3, 4, 5, 6, 7, 8]);
  assert.deepEqual([
    obtenerFaseRRHH("solicitud_registrada")?.orden,
    obtenerFaseRRHH("informe_juridico")?.orden,
    obtenerFaseRRHH("subsanacion_unidad")?.orden,
    obtenerFaseRRHH("llamamiento")?.orden,
    obtenerFaseRRHH("contratacion_temporal.fase.nombramiento")?.orden,
    obtenerFaseRRHH("fase-seguimiento")?.orden,
  ], [1, 3, 4, 5, 6, 8]);
  assert.equal(formatearFaseRRHH("llamamiento"), "Fase 5 de 8: Obtención del candidato");
  assert.equal(formatearFaseRRHH("llamamiento", "en"), "Phase 5 of 8: Candidate selection");
  assert.equal(nombreFaseRRHH("Análisis RRHH"), "Análisis de RRHH");
  assert.equal(nombreFaseRRHH("fase_ajena"), null, "un dato desconocido no se convierte en fase");
  assert.equal(formatearFaseRRHH("fase_ajena"), null);
  assert.ok(Object.isFrozen(FASES_RRHH));
  assert.ok(FASES_RRHH.every(Object.isFrozen));
});

test("los estados visibles usan la misma palabra en español e inglés", () => {
  assert.equal(nombreEstadoRRHH("en_curso"), "En tramitación");
  assert.equal(nombreEstadoRRHH("espera"), "Esperando a otro departamento");
  assert.equal(nombreEstadoRRHH("incidencia", "en"), "Needs attention");
  assert.equal(nombreEstadoRRHH("estado_ajeno"), null);
});
