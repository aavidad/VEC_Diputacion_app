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

// Recorrido del 25/09/2026: abrir un expediente en Nombramiento pedía la
// incorporación y el servidor respondía 409 («preparacion_pendiente») con error
// de consola. Ahora se consulta al pedirla; una vez pedida, los repintados del
// mismo expediente la vuelven a consultar solos.
test("abrir el expediente no consulta la incorporación: se ofrece un botón y se consulta al pedirla", async () => {
  const expedienteRef = "expediente:ejercicio:bajo-demanda";
  let contenedor = { innerHTML: "" };
  const raiz = {
    contains: (nodo) => nodo === contenedor,
    querySelector: (selector) => selector === "[data-ct-exp-incorporacion-ejercicio]" ? contenedor : null,
  };
  const estado = Object.freeze({ carga: "listo", vista: "expediente", expediente: Object.freeze({
    expediente_ref: expedienteRef, version: 8, demostracion: false,
  }) });
  let lecturas = 0;
  const gestor = crearGestorIncorporacion({
    raiz,
    presentador: { obtenerEstado: () => estado },
    incorporacionEjercicioDisponible: true,
    clienteLlamamiento: {
      async prepararIncorporacionEjercicio() {
        lecturas++;
        throw Object.assign(new Error("pendiente"), { estado: 409, codigo: "preparacion_pendiente", envelopeValido: true });
      },
      confirmarIncorporacionEjercicio() { assert.fail("no confirma"); },
    },
  });

  await gestor.ofrecerIncorporacionEjercicio();
  assert.equal(lecturas, 0, "abrir el expediente no hace la consulta");
  assert.match(contenedor.innerHTML, /data-ct-exp-accion="consultar-incorporacion"/u);
  assert.match(contenedor.innerHTML, /Consultar la incorporación al ejercicio/u);

  await gestor.montarIncorporacionEjercicio();
  assert.equal(lecturas, 1);
  assert.match(contenedor.innerHTML, /preparación de incorporación no está disponible/u);

  // Repintado del mismo expediente: contenedor nuevo, se consulta sin pedirlo.
  contenedor = { innerHTML: "" };
  await gestor.ofrecerIncorporacionEjercicio();
  assert.equal(lecturas, 2);
  assert.doesNotMatch(contenedor.innerHTML, /consultar-incorporacion/u);
});

test("la incorporación no se ofrece fuera de un detalle v8+ real", async () => {
  const contenedor = { innerHTML: "" };
  for (const expediente of [
    { expediente_ref: "expediente:ejercicio:v7", version: 7, demostracion: false },
    { expediente_ref: "expediente:ejercicio:demo", version: 8, demostracion: true },
  ]) {
    const gestor = crearGestorIncorporacion({
      raiz: { contains: () => true, querySelector: () => contenedor },
      presentador: { obtenerEstado: () => ({ carga: "listo", vista: "expediente", expediente }) },
      incorporacionEjercicioDisponible: true,
      clienteLlamamiento: { prepararIncorporacionEjercicio() { assert.fail("sin consulta"); } },
    });
    await gestor.ofrecerIncorporacionEjercicio();
    assert.equal(contenedor.innerHTML, "");
  }
});
