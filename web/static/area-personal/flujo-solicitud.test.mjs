import assert from "node:assert/strict";
import test from "node:test";

import { declaracionFinalConfirmada, estadoActosSolicitud, localizarSolicitudEdicion } from "./flujo-solicitud.js";

test("la selección de solicitud es exacta y los actos se derivan del expediente", () => {
  const datos = {
    solicitudes: [
      { id: "DEMO-SOL-REGISTRADA", convocatoria_id: "DEMO-CONV-001", estado: "Registrada", pago: "Tasa abonada", firma: "Firma válida" },
      { id: "DEMO-SOL-BORRADOR", convocatoria_id: "DEMO-CONV-001", estado: "Borrador efímero", pago: "Pendiente", firma: "Pendiente" },
    ],
  };
  assert.equal(localizarSolicitudEdicion(datos, { solicitudId: "DEMO-SOL-BORRADOR" }).id, "DEMO-SOL-BORRADOR");
  assert.equal(localizarSolicitudEdicion(datos, { convocatoriaId: "DEMO-CONV-001" }).id, "DEMO-SOL-BORRADOR");
  assert.deepEqual(estadoActosSolicitud(datos.solicitudes[1]), {
    pagoConfirmado: false, firmaConfirmada: false, registrada: false,
  });
  assert.equal(declaracionFinalConfirmada("on"), true);
  assert.equal(declaracionFinalConfirmada(undefined), false);
});
