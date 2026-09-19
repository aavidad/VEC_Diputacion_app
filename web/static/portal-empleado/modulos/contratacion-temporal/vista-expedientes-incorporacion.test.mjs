import assert from "node:assert/strict";
import test from "node:test";
import { crearGestorIncorporacion } from "./vista-expedientes-incorporacion.js";

test("preparación pendiente muestra actualización GET sin formulario ni POST", async () => {
  const expedienteRef = "expediente:ejercicio:sin-plan";
  const contenedor = { innerHTML: "" };
  const raiz = {
    contains: (nodo) => nodo === contenedor,
    querySelector: (selector) => selector === "[data-ct-exp-incorporacion-ejercicio]" ? contenedor : null,
  };
  const estado = Object.freeze({ carga: "listo", vista: "expediente", expediente: Object.freeze({
    expediente_ref: expedienteRef, version: 8, demostracion: false,
  }) });
  let lecturas = 0, posts = 0;
  const gestor = crearGestorIncorporacion({
    raiz,
    presentador: { obtenerEstado: () => estado },
    incorporacionEjercicioDisponible: true,
    clienteLlamamiento: {
      async prepararIncorporacionEjercicio(ref, { signal }) {
        lecturas++;
        assert.equal(ref, expedienteRef);
        assert.ok(signal instanceof AbortSignal);
        throw Object.assign(new Error("pendiente"), {
          estado: 409, codigo: "preparacion_pendiente", envelopeValido: true,
        });
      },
      confirmarIncorporacionEjercicio() { posts++; },
    },
  });

  await gestor.montarIncorporacionEjercicio();
  assert.equal(lecturas, 1);
  assert.equal(posts, 0);
  assert.match(contenedor.innerHTML, /preparación de incorporación no está disponible/u);
  assert.match(contenedor.innerHTML, /preparación de incorporación no está disponible/u);
  assert.match(contenedor.innerHTML, /Actualizar la lectura de incorporación/u);
  assert.match(contenedor.innerHTML, /data-ct-exp-accion="reintentar-incorporacion"/u);
  assert.doesNotMatch(contenedor.innerHTML, /<form|confirmar incorporación/u);
});
