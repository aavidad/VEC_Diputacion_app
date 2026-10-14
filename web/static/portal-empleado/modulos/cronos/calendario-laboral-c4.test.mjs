import test from "node:test";
import assert from "node:assert/strict";
import { validarCalendarioLaboralC4, fechaCalendarioC4Valida } from "./cliente-calendario-laboral-c4.js";
import { renderizarCalendarioLaboralC4 } from "./vista-calendario-laboral-c4.js";

const fecha = "2026-06-15";
const base = {
  fecha, estado: "programado", centro_ref: "centro:demo", fuente_unidad_ref: "personal:fuente",
  fuente_unidad_version: "1", fuente_organizacion_ref: "organizacion:fuente", version_organizacion: 1,
  versiones_calendario: Array.from({ length: 4 }, (_, i) => ({ denominacion: `Ámbito ${i + 1}`, numero: 1 })),
  dia: { laborable: true }, minutos_programados: 480, turno_ref: "noche", regimen_ref: "regimen:demo",
  relacion_ref: "rel_demo", programacion_ref: "programacion:cronos:demo", politica_version_ref: "politica:v1",
  fuente_programacion_ref: "fuente:cronos:demo", version_programacion: 1, fuente_version_programacion: 1,
  inicio_utc: "2026-06-15T18:00:00Z", fin_utc: "2026-06-16T02:00:00Z",
};

test("C4 muestra turno que termina al día siguiente solo con fuente completa", () => {
  assert.equal(fechaCalendarioC4Valida("2026-02-30"), false);
  assert.equal(validarCalendarioLaboralC4(base, fecha), base);
  const html = renderizarCalendarioLaboralC4({ fecha, estado: "listo", datos: base, locale: "es-ES" });
  assert.match(html, /16\/6\/26|16\/06\/2026|16\/6\/2026/u);
  assert.match(html, /480|08:00/u);
  assert.match(html, /fuente:cronos:demo/u);
  assert.throws(() => validarCalendarioLaboralC4({ ...base, centro_ref: "" }, fecha));
  assert.throws(() => validarCalendarioLaboralC4({ ...base, inicio_utc: base.fin_utc }, fecha));
});

test("C4 deja la jornada indeterminada cuando falta programación", () => {
  const sinJornada = { ...base, estado: "calendario_publicado_jornada_no_determinable", minutos_programados: null };
  delete sinJornada.turno_ref;
  assert.equal(validarCalendarioLaboralC4(sinJornada, fecha), sinJornada);
  const html = renderizarCalendarioLaboralC4({ fecha, estado: "listo", datos: sinJornada, locale: "es-ES" });
  assert.doesNotMatch(html, /08:00/u);
});
