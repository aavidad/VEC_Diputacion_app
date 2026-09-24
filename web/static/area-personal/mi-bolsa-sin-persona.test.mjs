import assert from "node:assert/strict";
import test from "node:test";
import { datosMinimosMiBolsa } from "./aplicacion.js";
import { renderizarPerfil } from "./vistas/perfil-meritos-solicitud.js";

test("Mi bolsa vacía no fabrica persona, iniciales ni referencias para el área personal", () => {
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-09-24T09:00:00Z", participaciones: [] });
  assert.equal(datos.meta.presentacion, false);
  assert.equal(datos.sesion.nombre_visible, "Identidad no facilitada");
  assert.equal(datos.sesion.iniciales, "—");
  assert.equal(datos.sesion.persona_ref, null);
  assert.equal(datos.perfil.referencia, null);
  assert.equal(datos.perfil.nombre_visible, "Identidad no facilitada");
  assert.deepEqual(datos.capacidades, {});
  assert.equal(datos.disponibilidad.disponible, false);
  const texto = `${JSON.stringify(datos)}\n${renderizarPerfil(datos)}`;
  assert.doesNotMatch(texto, /Candidato identificado|candidato:identificado|perfil:pendiente|"CI"|DEMO-/u);
  assert.match(texto, /Identidad no facilitada/u);
});
