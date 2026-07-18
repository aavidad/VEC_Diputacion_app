import assert from "node:assert/strict";
import test from "node:test";

import {
  aplicarPasoSolicitud, crearPayloadBorrador, crearProgresoSolicitud,
  declaracionFinalConfirmada, estadoActosSolicitud, localizarSolicitudEdicion,
} from "./flujo-solicitud.js";

test("el asistente no avanza sin bases, datos y al menos un mérito", () => {
  let progreso = crearProgresoSolicitud();
  assert.throws(() => aplicarPasoSolicitud(progreso, 1, { convocatoria: "DEMO-CONV-001" }), /bases.*requisitos/iu);
  progreso = aplicarPasoSolicitud(progreso, 1, { convocatoria: "DEMO-CONV-001", requisitos_confirmados: "on" });
  assert.throws(() => aplicarPasoSolicitud(progreso, 2, {}), /datos personales/iu);
  progreso = aplicarPasoSolicitud(progreso, 2, { datos_confirmados: "on" });
  assert.throws(() => aplicarPasoSolicitud(progreso, 3, {}), /al menos un mérito/iu);
  progreso = aplicarPasoSolicitud(progreso, 3, { meritos: ["DEMO-MER-001", "DEMO-MER-002"] });
  assert.throws(() => crearPayloadBorrador(progreso), /autobaremación revisada/iu);
  progreso = aplicarPasoSolicitud(progreso, 4);
  assert.deepEqual(crearPayloadBorrador(progreso).meritos_ids, ["DEMO-MER-001", "DEMO-MER-002"]);
});

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
