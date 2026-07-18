import assert from "node:assert/strict";
import test from "node:test";
import { resolverSolicitudPropuestaLlamamiento } from "./portal-llamamientos-flujo.js";

function propuestaPresentacion() {
  return {
    esquema: "vec.bolsa.propuesta-llamamiento.presentacion.v1",
    demostracion: true,
    id: "DEMO-PRO-1",
    necesidad_id: "DEMO-NEC-1",
    estado: "demostracion",
    version_bolsa: "versión sintética",
    version_regla: "regla sintética",
    fecha_corte: "2026-07-18T08:00:00Z",
    personas_incluidas: "1",
    evaluaciones: [{
      orden: "1",
      resultado: "elegible",
      motivos: [{ regla: "R1", fundamento: "Supuesto sintético" }],
    }],
  };
}

test("la demostración se resuelve localmente sin tocar el cliente HTTP", async () => {
  let llamadas = 0;
  const resultado = await resolverSolicitudPropuestaLlamamiento({
    modoPresentacion: true,
    necesidadId: "DEMO-NEC-1",
    capacidad: false,
    obtenerPresentacion: propuestaPresentacion,
    cliente: { async solicitar() { llamadas += 1; throw new Error("no debe ejecutarse"); } },
  });
  assert.equal(llamadas, 0);
  assert.equal(resultado.ok, true);
  assert.equal(resultado.sintetica, true);
  assert.equal(resultado.avanzar, true);
  assert.equal(resultado.propuesta.demostracion, true);
});

test("una confirmación real no avanza a detalle ni configuración", async () => {
  const confirmacion = { propuesta_ref: "propuesta:01", necesidad: { referencia: "necesidad:01" } };
  const resultado = await resolverSolicitudPropuestaLlamamiento({
    modoPresentacion: false,
    necesidadId: "necesidad:01",
    capacidad: true,
    obtenerPresentacion() { throw new Error("no debe ejecutarse"); },
    cliente: { async solicitar() { return { ok: true, confirmacion, etag: '"etag"' }; } },
  });
  assert.equal(resultado.ok, true);
  assert.equal(resultado.sintetica, false);
  assert.equal(resultado.avanzar, false);
  assert.equal(resultado.confirmacion, confirmacion);
  assert.match(resultado.mensaje, /Detalle no disponible/);
});
