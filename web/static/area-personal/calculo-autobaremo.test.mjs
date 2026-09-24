import assert from "node:assert/strict";
import test from "node:test";

import { calcularAutobaremo } from "./calculo-autobaremo.js";
import { renderizarAutobaremacion, renderizarSolicitud } from "./vistas/perfil-meritos-solicitud.js";

// Doble mínimo de los datos del área personal: solo los campos que leen el
// cálculo y la vista del paso de autobaremo.
function datosPrueba() {
  return {
    meta: { presentacion: false },
    convocatorias: [{ id: "CONV-001", titulo: "Bolsa de prueba", referencia: "REF-001", estado: "Plazo abierto" }],
    meritos: [
      { id: "MER-001", tipo: "Titulación", titulo: "Titulación", estado: "Validado", puntos_estimados: 2 },
      { id: "MER-002", tipo: "Experiencia", titulo: "Experiencia", estado: "Aportado", puntos_estimados: 5.4 },
      { id: "MER-003", tipo: "Formación", titulo: "Curso sin regla", estado: "Aportado", puntos_estimados: 0.6 },
    ],
    baremo: [
      { id: "BAR-001", merito_id: "MER-002", nombre: "Experiencia", detalle: "18 meses", estado: "Autobaremado", puntos: 5.4, maximo: 8 },
      { id: "BAR-002", merito_id: "MER-001", nombre: "Titulaciones", detalle: "Una titulación", estado: "Validado", puntos: 2, maximo: 3 },
      { id: "BAR-003", nombre: "Ejercicio superado", detalle: "De oficio", estado: "De oficio", puntos: 4.95, maximo: 5, de_oficio: true },
    ],
    solicitudes: [],
  };
}

test("méritos seleccionados y conceptos de oficio siguen una sola regla", () => {
  const datos = datosPrueba();
  const meritos = ["MER-001"];
  const calculo = calcularAutobaremo(datos, meritos);
  assert.equal(calculo.total, 6.95);
  assert.deepEqual(calculo.criterios.map((item) => item.id), ["BAR-002", "BAR-003"]);

  const html = renderizarSolicitud(datos, {
    pasoSolicitud: 4,
    convocatoriaSolicitud: "CONV-001",
    progresoSolicitud: {
      convocatoria_id: "CONV-001",
      requisitos_confirmados: true,
      datos_confirmados: true,
      meritos_ids: meritos,
      autobaremo_revisado: false,
    },
    solicitudEdicionId: "",
    errorPasoSolicitud: "",
  });
  assert.match(html, />6,95</u);
});

test("un mérito seleccionado sin regla aplicable queda pendiente y suma su estimación", () => {
  const calculo = calcularAutobaremo(datosPrueba(), ["MER-003", "MER-003", "", 7]);
  assert.deepEqual(calculo.meritos_ids, ["MER-003"]);
  assert.deepEqual(calculo.criterios.map((item) => item.id), ["BAR-003", "criterio-MER-003"]);
  assert.equal(calculo.total, 5.55);
});

test("la autobaremación desglosada no ofrece rótulos de demostración", () => {
  const html = renderizarAutobaremacion({
    ...datosPrueba(),
    resultado_autobaremo: { convocatoria_id: "CONV-001", meritos_ids: ["MER-001"], puntos: 6.95, calculado_en: "2026-09-25T10:00:00Z" },
  }, { convocatoriaSolicitud: "CONV-001" });
  assert.match(html, /Recalcular autobaremo/u);
  assert.match(html, /versión vigente de bases/u);
  assert.match(html, /<p class="nota"><strong>Resultado recalculado\.<\/strong>/u);
  assert.doesNotMatch(html, /DEMO|Simular|class="nota demo"/u);
});
