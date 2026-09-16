import assert from "node:assert/strict";
import test from "node:test";

import {
  validarSolicitudRectificacionAnalisis,
  validarSolicitudRegistroAnalisis,
} from "./contrato-analisis.js";

const HUELLA = "9".repeat(64);

function solicitudBase(analisisExtra = {}) {
  return {
    expediente_ref: "expediente:analisis:http:001",
    version_esperada: 1,
    clave_idempotencia: "11111111-2222-4333-8444-555555555555",
    artefacto_ref: "artefacto:analisis:http:001",
    analisis: {
      modalidad_clave: "sustitucion",
      categoria_ref: "categoria:tecnico:001",
      grupo_subgrupo: "A2",
      causa_clave: "sustitucion",
      periodo: {
        inicio: "2026-09-01T00:00:00Z",
        fin: "2027-02-28T00:00:00Z",
      },
      porcentaje_jornada: 7_500,
      entrada_rc: {
        referencia: "entrada:rc:http:001",
        huella_sha256: HUELLA,
      },
      ...analisisExtra,
    },
  };
}

test("contrato-analisis: admite solicitud sin observaciones y omite la clave", () => {
  const solicitud = solicitudBase();
  const validada = validarSolicitudRegistroAnalisis(solicitud);
  assert.equal(Object.hasOwn(validada.analisis, "observaciones"), false);
});

test("contrato-analisis: con observaciones vacías omite la clave", () => {
  const solicitud = solicitudBase({ observaciones: "" });
  const validada = validarSolicitudRegistroAnalisis(solicitud);
  assert.equal(Object.hasOwn(validada.analisis, "observaciones"), false);
});

test("contrato-analisis: con observaciones válidas incluye el valor", () => {
  const solicitud = solicitudBase({
    observaciones: "Observaciones de prueba válidas para el análisis.",
  });
  const validada = validarSolicitudRegistroAnalisis(solicitud);
  assert.equal(Object.hasOwn(validada.analisis, "observaciones"), true);
  assert.equal(
    validada.analisis.observaciones,
    "Observaciones de prueba válidas para el análisis.",
  );
});

test("contrato-analisis: rechaza observaciones mayores a 4000 caracteres", () => {
  const solicitud = solicitudBase({
    observaciones: "a".repeat(4001),
  });
  assert.throws(
    () => validarSolicitudRegistroAnalisis(solicitud),
    TypeError,
  );
});

test("contrato-analisis: rechaza observaciones con caracteres de control o no recortadas", () => {
  assert.throws(
    () => validarSolicitudRegistroAnalisis(solicitudBase({ observaciones: " texto " })),
    TypeError,
  );
  assert.throws(
    () => validarSolicitudRegistroAnalisis(solicitudBase({ observaciones: "texto\x00invalido" })),
    TypeError,
  );
});

test("contrato-analisis: rectificación admite observaciones válidas", () => {
  const solicitud = {
    ...solicitudBase({ observaciones: "Observación de rectificación" }),
    version_esperada: 2,
    motivo_rectificacion_clave: "ajuste_jornada",
  };
  const validada = validarSolicitudRectificacionAnalisis(solicitud);
  assert.equal(validada.analisis.observaciones, "Observación de rectificación");
});
