import assert from "node:assert/strict";
import test from "node:test";

import { NOMBRES_ICONO, icono } from "./iconos-vec.js";

test("cada icono es un SVG decorativo en línea que hereda el color", () => {
  for (const nombre of NOMBRES_ICONO) {
    const svg = icono(nombre);
    assert.match(svg, /^<svg aria-hidden="true" focusable="false" viewBox="0 0 24 24"/);
    assert.match(svg, /stroke="currentColor"/);
    assert.doesNotMatch(svg, /href|url\(|<script|on\w+=|#[0-9a-f]{3,8}\b/i);
  }
});

test("un nombre desconocido devuelve el genérico y la clase solo admite nombres simples", () => {
  assert.equal(icono("no-existe"), icono("generico"));
  assert.match(icono("bolsas", "icono-menu"), /^<svg class="icono-menu" /);
  assert.equal(icono("bolsas", '" onload="x'), icono("bolsas"));
});
