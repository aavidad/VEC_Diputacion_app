import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";
import { esModoPresentacion, validarDatosAreaPersonal, validarRecibo } from "./contrato.js";

test("el selector de presentación es único y explícito", () => {
  assert.equal(esModoPresentacion(new URLSearchParams("presentacion=rrhh")), true);
  assert.equal(esModoPresentacion(new URLSearchParams("presentacion=aspirante")), false);
  assert.equal(esModoPresentacion(new URLSearchParams()), false);
  assert.equal(esModoPresentacion(new URLSearchParams("presentacion=RRHH")), false);
});

test("el contrato acepta el juego sintético completo y lo congela", async () => {
  const datos = await crearAdaptadorPresentacion().cargar();
  assert.equal(datos.meta.presentacion, true);
  assert.equal(datos.meta.esquema, "vec.bolsa.area-personal.v1");
  assert.equal(Object.isFrozen(datos), true);
  assert.equal(Object.isFrozen(datos.convocatorias), true);
  assert.ok(datos.sesion.persona_ref.startsWith("DEMO-"));
  assert.ok(datos.perfil.referencia.startsWith("DEMO-"));
});

test("la presentación rechaza DNI o NIE formalmente válidos", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.perfil.identificador_visible = "12345678Z";
  assert.throws(
    () => validarDatosAreaPersonal(datos, { presentacionEsperada: true }),
    /no admite DNI o NIE/,
  );
});

test("la presentación solo acepta correos reservados .test", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.perfil.correo = "persona@example.com";
  assert.throws(
    () => validarDatosAreaPersonal(datos, { presentacionEsperada: true }),
    /dominio \.test/,
  );
});

test("un origen sintético nunca pasa por el contrato productivo", async () => {
  const datos = await crearAdaptadorPresentacion().cargar();
  assert.throws(
    () => validarDatosAreaPersonal(datos, { presentacionEsperada: false }),
    /origen no coincide/,
  );
});

test("el recibo DEMO es inequívoco y no pasa como recibo real", () => {
  const entrada = {
    esquema: "vec.bolsa.area-personal.recibo-demo.v1",
    presentacion: true,
    referencia: "DEMO-REC-0001",
    accion: "guardar_borrador",
    resultado: "Simulación completada sin efectos administrativos",
    actor: "Persona Aspirante de Demostración",
    fecha: "2026-07-18T09:00:00Z",
    advertencia: "RECIBO DEMO · Sin validez administrativa.",
  };
  assert.equal(validarRecibo(entrada, { presentacionEsperada: true }).presentacion, true);
  assert.throws(() => validarRecibo(entrada, { presentacionEsperada: false }), /esquema del recibo no es compatible/);
});
