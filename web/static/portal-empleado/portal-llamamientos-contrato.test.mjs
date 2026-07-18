import assert from "node:assert/strict";
import test from "node:test";
import {
  extraerDatosEnvelopeLlamamiento,
  validarConfirmacionPropuestaLlamamiento,
  validarPropuestaLlamamientoPresentacion,
  validarReferenciaOpacaLlamamiento,
} from "./portal-llamamientos-contrato.js";

const HUELLA = "a".repeat(64);
const ETAG = `"vec-propuesta-llamamiento-v1.sha256-${HUELLA}"`;

function version(referencia, caracter) {
  return { referencia, version: "3", huella_sha256: caracter.repeat(64) };
}

function confirmacionValida() {
  return {
    esquema: "vec.bolsa.propuesta-llamamiento.confirmacion.v1",
    propuesta_ref: "propuesta:01K0VS7P",
    huella_propuesta_sha256: HUELLA,
    bolsa: version("bolsa:01K0VS7Q", "b"),
    necesidad: version("necesidad:01K0VS7R", "c"),
    instantanea: version("instantanea:01K0VS7S", "d"),
    politica: version("politica:01K0VS7T", "e"),
    instante_referencia: "2026-07-18T08:00:00Z",
    instantanea_generada_en: "2026-07-18T08:00:00.123456Z",
    total_participaciones_instantanea: "24",
    total_evaluaciones: "4",
    orden_seleccionado: "4",
    generada_en: "2026-07-18T08:00:01Z",
  };
}

test("la confirmación real acepta exclusivamente el cuerpo compacto del servidor", () => {
  const entrada = confirmacionValida();
  const salida = validarConfirmacionPropuestaLlamamiento(
    extraerDatosEnvelopeLlamamiento({ data: entrada }),
    ETAG,
  );
  assert.deepEqual(salida, entrada);
  assert.deepEqual(Object.keys(salida), [
    "esquema", "propuesta_ref", "huella_propuesta_sha256", "bolsa", "necesidad",
    "instantanea", "politica", "instante_referencia", "instantanea_generada_en",
    "total_participaciones_instantanea", "total_evaluaciones", "orden_seleccionado",
    "generada_en",
  ]);
  assert.deepEqual(Object.keys(salida.bolsa), ["referencia", "version", "huella_sha256"]);
});

test("el envelope, sus objetos y el ETag fallan cerrados ante cualquier extra", () => {
  assert.throws(
    () => extraerDatosEnvelopeLlamamiento({ data: confirmacionValida(), meta: {} }),
    /contrato cerrado/,
  );
  for (const [campo, valor] of [
    ["evaluaciones", []],
    ["puntuacion", "99"],
    ["demostracion", true],
    ["sujeto_ref", "sujeto:01K0"],
    ["participacion_ref", "participacion:01K0"],
  ]) {
    assert.throws(
      () => validarConfirmacionPropuestaLlamamiento({ ...confirmacionValida(), [campo]: valor }, ETAG),
      /contrato cerrado/,
      campo,
    );
  }
  const nested = confirmacionValida();
  nested.bolsa.nombre = "dato inventado";
  assert.throws(() => validarConfirmacionPropuestaLlamamiento(nested, ETAG), /contrato cerrado/);
  assert.throws(
    () => validarConfirmacionPropuestaLlamamiento(confirmacionValida(), `W/${ETAG}`),
    /ETag/,
  );
});

test("versiones, contadores, referencias y tiempos conservan su representación canónica", () => {
  assert.equal(validarReferenciaOpacaLlamamiento("á".repeat(256)), "á".repeat(256));
  assert.throws(() => validarReferenciaOpacaLlamamiento("á".repeat(257)), /no válido/);
  for (const mutar of [
    (datos) => { datos.bolsa.version = 3; },
    (datos) => { datos.bolsa.version = "03"; },
    (datos) => { datos.total_evaluaciones = 4; },
    (datos) => { datos.orden_seleccionado = "3"; },
    (datos) => { datos.total_participaciones_instantanea = "250001"; },
    (datos) => { datos.necesidad.referencia = "necesidad: 01"; },
    (datos) => { datos.propuesta_ref = "propuesta:á"; },
    (datos) => { datos.propuesta_ref = "á".repeat(257); },
    (datos) => { datos.instantanea_generada_en = "2026-07-18T08:00:00.120000Z"; },
    (datos) => {
      datos.instante_referencia = "2026-07-18T08:00:00.000002Z";
      datos.instantanea_generada_en = "2026-07-18T08:00:00.000001Z";
    },
    (datos) => { datos.generada_en = "2026-07-18T07:59:59Z"; },
  ]) {
    const datos = confirmacionValida();
    mutar(datos);
    assert.throws(() => validarConfirmacionPropuestaLlamamiento(datos, ETAG));
  }
});

test("la presentación usa otro contrato, siempre demo y sin puntuaciones inventadas", () => {
  const demo = validarPropuestaLlamamientoPresentacion({
    esquema: "vec.bolsa.propuesta-llamamiento.presentacion.v1",
    demostracion: true,
    id: "PRO-DEMO-0045",
    necesidad_id: "NEC-DEMO-0045",
    estado: "demostracion",
    version_bolsa: "versión sintética 3",
    version_regla: "regla sintética 3",
    fecha_corte: "2026-07-18T08:00:00Z",
    personas_incluidas: "1",
    evaluaciones: [
      { orden: "1", resultado: "no_elegible", motivos: [{ regla: "R4", fundamento: "Indisponibilidad sintética" }] },
      { orden: "2", resultado: "elegible", motivos: [{ regla: "R1", fundamento: "Orden sintético vigente" }] },
    ],
  });
  assert.equal(demo.demostracion, true);
  assert.deepEqual(Object.keys(demo.evaluaciones[0]), ["orden", "resultado", "motivos"]);
  assert.throws(() => validarPropuestaLlamamientoPresentacion({ ...demo, demostracion: false }), /no compatible/);
  assert.throws(() => validarPropuestaLlamamientoPresentacion({
    ...demo,
    evaluaciones: [{ ...demo.evaluaciones[0], puntuacion: 99 }, demo.evaluaciones[1]],
  }), /contrato cerrado/);
});
