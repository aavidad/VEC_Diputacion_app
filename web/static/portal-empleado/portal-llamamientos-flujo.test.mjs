import assert from "node:assert/strict";
import test from "node:test";
import { resolverSolicitudPropuestaLlamamiento } from "./portal-llamamientos-flujo.js";

test("una confirmación real no avanza a detalle ni configuración", async () => {
  const confirmacion = { propuesta_ref: "propuesta:01", necesidad: { referencia: "necesidad:01" } };
  const resultado = await resolverSolicitudPropuestaLlamamiento({
    necesidadId: "necesidad:01",
    capacidad: true,
    cliente: { async solicitar() { return { ok: true, confirmacion, etag: '"etag"' }; } },
  });
  assert.equal(resultado.ok, true);
  assert.equal(resultado.avanzar, false);
  assert.equal(resultado.confirmacion, confirmacion);
  assert.match(resultado.mensaje, /Detalle no disponible/);
});
