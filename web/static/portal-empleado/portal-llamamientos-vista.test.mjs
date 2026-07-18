import assert from "node:assert/strict";
import test from "node:test";
import {
  renderizarConfirmacionCompacta,
  renderizarDetalleLlamamientoBloqueado,
  renderizarPasosLlamamiento,
} from "./portal-llamamientos-vista.js";

test("el recorrido real mantiene los tres pasos posteriores realmente deshabilitados", () => {
  const html = renderizarPasosLlamamiento({ modoPresentacion: false, pasoActual: 4 });
  assert.equal(html.match(/ disabled/g)?.length, 3);
  assert.match(html, /data-paso="1" aria-current="step"/);
  assert.match(html, /Revisar propuesta/);
  assert.match(renderizarDetalleLlamamientoBloqueado(), /Detalle no disponible/);
});

test("cada paso navegable de la presentación se identifica como demostración", () => {
  const html = renderizarPasosLlamamiento({ modoPresentacion: true, pasoActual: 3 });
  assert.doesNotMatch(html, / disabled/);
  assert.match(html, /Elegir necesidad de demostración/);
  assert.match(html, /Revisar demostración/);
  assert.match(html, /Configurar demostración/);
  assert.match(html, /Comprobar presentación/);
});

test("la confirmación compacta escapa referencias y declara el bloqueo", () => {
  const html = renderizarConfirmacionCompacta({
    propuesta_ref: "propuesta:<script>",
    necesidad: { referencia: "necesidad:1", version: "1" },
    bolsa: { referencia: "bolsa:1", version: "1" },
    instantanea: { referencia: "instantanea:1", version: "1" },
    politica: { referencia: "politica:1", version: "1" },
    generada_en: "2026-07-18T08:00:00Z",
    total_evaluaciones: "2",
    orden_seleccionado: "2",
  });
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /propuesta:&lt;script&gt;/);
  assert.match(html, /Detalle no disponible/);
  assert.match(html, /configuración permanecen bloqueados/);
});
