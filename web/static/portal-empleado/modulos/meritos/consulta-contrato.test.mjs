import assert from "node:assert/strict";
import test from "node:test";
import { validarResultadoConsulta } from "./consulta-contrato.js";
import { resultadoPrueba, noEncontradaPrueba } from "./consulta-prueba.test-helper.mjs";

test("el contrato minimizado copia y congela todos los datos sin congelar el dato ajeno", () => {
  const original = resultadoPrueba(); const actual = validarResultadoConsulta(original, "hecho:propio-a");
  original.hecho_actual.denominacion = "cambiada"; original.hecho_actual.evidencias[0].id = "otro";
  assert.equal(actual.hecho_actual.denominacion, "Curso de gestión");
  assert.equal(actual.hecho_actual.evidencias[0].id, "documento:curso");
  assert.ok(Object.isFrozen(actual.hecho_actual.procedencia));
  assert.ok(Object.isFrozen(actual.hecho_actual.evidencias[0]));
  assert.ok(Object.isFrozen(actual.recibo_consulta));
  assert.equal(Object.isFrozen(original), false);
});

test("rechaza identidades, bytes, puntos y actores en cualquier frontera cerrada", () => {
  const casos = [
    (v) => { v.hecho_actual.persona_ref = "persona:otra"; },
    (v) => { v.hecho_actual.declarante_ref = "actor:1"; },
    (v) => { v.hecho_actual.revision.actor_ref = "actor:1"; },
    (v) => { v.hecho_actual.evidencias[0].contenido = "privado"; },
    (v) => { v.hecho_actual.puntos = 20; },
    (v) => { v.recibo_consulta.huella_proyeccion = "a".repeat(64); },
    (v) => { v.recibo_mutacion = "recibo:viejo"; },
  ];
  for (const modificar of casos) { const datos = resultadoPrueba(); modificar(datos); assert.throws(() => validarResultadoConsulta(datos, "hecho:propio-a")); }
});

test("el recibo corresponde al hecho, versión y estado encontrados", () => {
  const no = validarResultadoConsulta(noEncontradaPrueba(), "hecho:propio-a");
  assert.equal(no.hecho_actual, null); assert.equal(no.recibo_consulta.version_consultada, 0);
  for (const cambiar of [
    (v) => { v.recibo_consulta.hecho_ref = "hecho:otro"; },
    (v) => { v.recibo_consulta.version_consultada = 1; },
    (v) => { v.hecho_actual.referencia = "hecho:otro"; },
    (v) => { v.hecho_actual.version = 2147483648; },
    (v) => { v.hecho_actual.evidencias.push(v.hecho_actual.evidencias[0]); },
    (v) => { v.hecho_actual.procedencia.capturada_en = "2026-02-30T08:30:00Z"; },
    (v) => { v.hecho_actual.vigencia.hasta = "2026-02-28"; },
    (v) => { v.hecho_actual.revision = null; },
    (v) => { v.recibo_consulta.consumo_huella_sha256 = "privada"; },
  ]) { const datos = resultadoPrueba(); cambiar(datos); assert.throws(() => validarResultadoConsulta(datos, "hecho:propio-a")); }
  const falsa = noEncontradaPrueba(); falsa.hecho_actual = resultadoPrueba().hecho_actual;
  assert.throws(() => validarResultadoConsulta(falsa, "hecho:propio-a"));
});
