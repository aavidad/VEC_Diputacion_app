// El cuadro de bolsas de Inicio y de Bolsa fallaba al azar: el validador pasaba
// los instantes por los patrones de datos personales y una fracción de segundo
// de ocho cifras seguida de «Z» casaba con el de DNI.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import { consultarBolsas } from "./portal-bolsas-api.js";
import { validarRespuestaBolsas } from "./portal-bolsas-contrato.js";

const real = JSON.parse(readFileSync(new URL("./testdata/bolsas-cidonia-hz12.json", import.meta.url), "utf8"));

function responder(cuerpo) {
  return async () => new Response(JSON.stringify(cuerpo), { status: 200, headers: { "Content-Type": "application/json" } });
}

function conGeneradoEn(generadoEn) {
  return { data: { ...real.data, generado_en: generadoEn } };
}

test("la respuesta real de cidonia se acepta tal cual llega", async () => {
  const res = await consultarBolsas({ fetchImpl: responder(real) });
  assert.equal(res.ok, true, res.mensaje);
  assert.equal(res.datos.bolsas.length, 12);
});

test("una fracción de segundo de ocho cifras ya no se toma por un DNI", async () => {
  // Go quita los ceros finales: «…31.070147320Z» viaja como «…31.07014732Z».
  for (const generadoEn of ["2026-10-09T01:26:31.07014732Z", "2026-10-09T01:26:31.67014732+02:00", "2026-10-09T01:26:31Z", "2026-10-09T01:26:31.1Z"]) {
    const res = await consultarBolsas({ fetchImpl: responder(conGeneradoEn(generadoEn)) });
    assert.equal(res.ok, true, `${generadoEn}: ${res.mensaje}`);
    assert.equal(res.datos.generado_en, generadoEn);
  }
});

test("un instante con texto añadido o en otro formato sigue rechazándose", () => {
  const { esquema, bolsas } = real.data;
  for (const generadoEn of ["2026-10-09T01:26:31Z 12345678Z", "12345678Z 2026-10-09", "Fri, 09 Oct 2026 01:26:31 GMT", "2026-10-09T01:26:31.1234567890Z", "no-es-fecha"]) {
    assert.throws(() => validarRespuestaBolsas({ data: { esquema, generado_en: generadoEn, bolsas } }), /instante|fecha/, generadoEn);
  }
});
