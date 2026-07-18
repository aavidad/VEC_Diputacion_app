import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";
import { calcularAutobaremo } from "./calculo-autobaremo.js";
import { renderizarSolicitud } from "./vistas/perfil-meritos-solicitud.js";

test("méritos seleccionados y conceptos de oficio siguen una sola regla", async () => {
  const adaptador = crearAdaptadorPresentacion();
  const inicial = await adaptador.cargar();
  const meritos = ["DEMO-MER-001"];
  const calculo = calcularAutobaremo(inicial, meritos);
  assert.equal(calculo.total, 6.95);
  assert.deepEqual(calculo.criterios.map((item) => item.id), ["DEMO-BAR-003", "DEMO-BAR-005"]);

  const html = renderizarSolicitud(inicial, {
    pasoSolicitud: 4,
    convocatoriaSolicitud: "DEMO-CONV-001",
    progresoSolicitud: {
      convocatoria_id: "DEMO-CONV-001",
      requisitos_confirmados: true,
      datos_confirmados: true,
      meritos_ids: meritos,
      autobaremo_revisado: false,
    },
    solicitudEdicionId: "",
    errorPasoSolicitud: "",
  });
  assert.match(html, />6,95</u);

  await adaptador.ejecutar({
    accion: "guardar_borrador",
    payload: {
      convocatoria_id: "DEMO-CONV-001",
      requisitos_confirmados: true,
      datos_confirmados: true,
      meritos_ids: meritos,
      autobaremo_revisado: true,
    },
    confirmacion: true,
    capacidad: true,
  });
  const borrador = (await adaptador.cargar()).solicitudes.find((item) => item.id === "DEMO-SOL-BORRADOR-0001");
  assert.equal(borrador.puntuacion, calculo.total);

  const recalculo = await adaptador.ejecutar({
    accion: "calcular_autobaremo",
    payload: { convocatoria_id: "DEMO-CONV-001", meritos_ids: meritos },
    confirmacion: true,
    capacidad: true,
  });
  assert.equal(recalculo.datos.resultado_autobaremo.puntos, calculo.total);
});
