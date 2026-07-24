import assert from "node:assert/strict";
import test from "node:test";
import { crearCoordinadorOperacionesBorradores } from "./portal-borradores-operaciones.js";

test("una operación posterior cancela la anterior del mismo tipo", () => {
  const coordinador = crearCoordinadorOperacionesBorradores();
  const anterior = coordinador.iniciar("carga");
  const reciente = coordinador.iniciar("carga");
  assert.equal(anterior.signal.aborted, true);
  assert.equal(anterior.vigente(), false);
  assert.equal(reciente.vigente(), true);
});

test("la invalidación cancela todas las clases y vence cualquier respuesta tardía", () => {
  const coordinador = crearCoordinadorOperacionesBorradores();
  const operaciones = ["carga", "detalle", "guardado", "postguardado", "cas"]
    .map((nombre) => coordinador.iniciar(nombre));
  coordinador.invalidar();
  for (const operacion of operaciones) {
    assert.equal(operacion.signal.aborted, true);
    assert.equal(operacion.vigente(), false);
  }
});

test("finalizar una generación antigua no retira su sustituta", () => {
  const coordinador = crearCoordinadorOperacionesBorradores();
  const anterior = coordinador.iniciar("detalle");
  const reciente = coordinador.iniciar("detalle");
  anterior.finalizar();
  assert.equal(reciente.vigente(), true);
  reciente.finalizar();
  assert.equal(reciente.vigente(), false);
});
