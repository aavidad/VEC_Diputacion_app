import test from "node:test";
import assert from "node:assert/strict";
import { formatearPorcentajeJornada } from "./porcentaje-jornada.js";

test("valores validos", () => {
  assert.equal(formatearPorcentajeJornada(5000), "50\u00a0%");
  assert.equal(formatearPorcentajeJornada(10000), "100\u00a0%");
  assert.equal(formatearPorcentajeJornada(1), "0,01\u00a0%");
  assert.equal(formatearPorcentajeJornada(1234), "12,34\u00a0%");
  assert.equal(formatearPorcentajeJornada("5000"), "50\u00a0%");
});

test("locale en-US", () => {
  assert.equal(formatearPorcentajeJornada(1234, "en-US"), "12.34%");
});

test("valores invalidos", () => {
  const invalidos = [0, 10001, -1, NaN, Infinity, 1.5, null, undefined, true, [], {}, "01", " 5000", "5000 ", "1.0", "1e3", ""];
  for (const v of invalidos) {
    assert.equal(formatearPorcentajeJornada(v), "");
  }
  const obj = { valueOf() { throw new Error("no coercionar"); } };
  assert.equal(formatearPorcentajeJornada(obj), "");
});

test("locale invalido", () => {
  assert.throws(() => formatearPorcentajeJornada(5000, "es_ES"), RangeError);
});
