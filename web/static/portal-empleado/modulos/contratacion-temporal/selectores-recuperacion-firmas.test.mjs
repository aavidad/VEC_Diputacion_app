import assert from "node:assert/strict";
import test from "node:test";
import { validarSelectoresRecuperacionFirmas, validarMetadatosRecuperacionFirmas } from "./selectores-recuperacion-firmas.js";

const expediente = "expediente:ct:001";
const fila = () => ({ firma_ref: "firma:ct:original", selector: {
  expediente_ref: expediente, version_expediente: 2, documento: "resolucion",
  paso_orden: 3, clave_idempotencia: "operacion-original-1234",
  catalogo_huella: "a".repeat(64), via: "certificado_vec",
} });

test("el selector conserva el corte original aunque el expediente haya avanzado", () => {
  const entrada = [fila()];
  const resultado = validarSelectoresRecuperacionFirmas(entrada, expediente, 9);
  assert.equal(resultado[0].selector.version_expediente, 2);
  assert.equal(resultado[0].selector.paso_orden, 3);
  Object.assign(entrada[0].selector, { clave_idempotencia: "otra-operacion-12345" });
  assert.equal(resultado[0].selector.clave_idempotencia, "operacion-original-1234");
  assert.ok(Object.isFrozen(resultado[0].selector));
  assert.deepEqual(validarSelectoresRecuperacionFirmas([], expediente, 9), []);
});

test("rechaza selectores cruzados, duplicados, externos o fuera del corte visible", () => {
  for (const cambio of [
    { expediente_ref: "expediente:ct:otro" }, { version_expediente: 10 },
    { via: "portafirmas_registro_rrhh" }, { paso_orden: 4 },
    { catalogo_huella: "" }, { unidad_ref: "unidad:oculta" },
  ]) {
    const dato = fila(); Object.assign(dato.selector, cambio);
    assert.throws(() => validarSelectoresRecuperacionFirmas([dato], expediente, 9), TypeError);
  }
  assert.throws(() => validarSelectoresRecuperacionFirmas([fila(), fila()], expediente, 9), TypeError);
  assert.throws(() => validarSelectoresRecuperacionFirmas(Array.from({ length: 129 }, fila), expediente, 9), TypeError);
  assert.throws(() => validarSelectoresRecuperacionFirmas([{ ...fila(), actor: "persona:oculta" }], expediente, 9), TypeError);
});


test("el contrato ampliado exige ambos sellos incluso sin firmas y el antiguo los omite", () => {
  const recibo = { acceso_ref: "acceso:ct:original", sello_detalle_sha256: "a".repeat(64), sello_selectores_sha256: "b".repeat(64) };
  assert.deepEqual(validarMetadatosRecuperacionFirmas({}, expediente, 9), {});
  const completo = { selectores_recuperacion_firmas: [], recibo_selectores_recuperacion_firmas: recibo };
  assert.deepEqual(validarMetadatosRecuperacionFirmas(completo, expediente, 9), completo);
  for (const parcial of [
    { selectores_recuperacion_firmas: [] }, { recibo_selectores_recuperacion_firmas: recibo },
    { ...completo, recibo_selectores_recuperacion_firmas: { ...recibo, sello_selectores_sha256: "" } },
    { ...completo, recibo_selectores_recuperacion_firmas: { ...recibo, unidad_ref: "unidad:oculta" } },
  ]) assert.throws(() => validarMetadatosRecuperacionFirmas(parcial, expediente, 9), TypeError);
});
