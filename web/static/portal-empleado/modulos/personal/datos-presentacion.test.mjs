import assert from "node:assert/strict";
import test from "node:test";
import {
  REFERENCIA_TITULAR_DEMO,
  crearDatosPersonalPresentacion,
} from "./datos-presentacion.js";

test("Personal entrega solo datos sintéticos, efímeros y sin efectos", () => {
  const datos = crearDatosPersonalPresentacion();
  assert.equal(datos.esquema, "vec.personal.panel.v1");
  assert.equal(datos.origen.demostracion, true);
  assert.equal(datos.origen.efimero, true);
  assert.equal(datos.origen.efectos_reales, false);
  assert.equal(datos.titular_ref, REFERENCIA_TITULAR_DEMO);
  assert.match(datos.origen.aviso, /no acreditan/i);
});

test("Personal no calcula derechos ni expone importes, bases o pagos", () => {
  const datos = crearDatosPersonalPresentacion();
  assert.ok(datos.servicios.every((servicio) => servicio.computable === false));
  assert.ok(datos.servicios.every((servicio) => /No se calcula antigüedad, trienios/i.test(servicio.observacion)));
  assert.ok(datos.nominas.every((nomina) => nomina.importes_disponibles === false && nomina.recibo_disponible === false));
  assert.ok(datos.dietas_cobradas.every((dieta) => dieta.importe_disponible === false && dieta.pago_acreditado === false));
  assert.doesNotMatch(JSON.stringify(datos), /DNI|@|importe_total|base_cotizacion|cuenta_bancaria/i);
});

test("cada lectura de la presentación devuelve una copia aislada", () => {
  const inicial = crearDatosPersonalPresentacion();
  inicial.relacion_actual.puesto = "alterado";
  inicial.servicios.pop();
  const posterior = crearDatosPersonalPresentacion();
  assert.equal(posterior.relacion_actual.puesto, "Puesto de ejemplo · DEMO");
  assert.equal(posterior.servicios.length, 2);
});
