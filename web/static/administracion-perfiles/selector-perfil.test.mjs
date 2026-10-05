import assert from "node:assert/strict";
import test from "node:test";
import { validarSeleccion } from "./selector-perfil.js";

const PERFIL = "prf_aaaaaaaaaaaaaaaaaaaaaa";

function seleccion(auditoria) {
  return { perfil_activo_ref: PERFIL, revision: 4, seleccionada_en: "2026-10-05T08:00:00Z", auditoria_ref: auditoria };
}

test("la selección admite la referencia de la auditoría común y la heredada", () => {
  assert.equal(validarSeleccion(seleccion(`aud_v3_p_${"0a".repeat(16)}`), PERFIL, 3), true);
  assert.equal(validarSeleccion(seleccion(`auditoria_seleccion_admin:${"c".repeat(64)}`), PERFIL, 3), true);
});

test("la selección rechaza referencias de auditoría malformadas", () => {
  for (const mala of [`aud_v3_p_${"A".repeat(32)}`, `aud_v3_p_${"a".repeat(31)}`, `aud_v3_p_${"a".repeat(33)}`,
    `aud_v3_x_${"a".repeat(32)}`, `auditoria_seleccion_admin:${"c".repeat(63)}`, ""]) {
    assert.equal(validarSeleccion(seleccion(mala), PERFIL, 3), false, mala);
  }
});
