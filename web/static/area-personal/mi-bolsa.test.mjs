import assert from "node:assert/strict";
import test from "node:test";

import { crearClienteMiBolsa, ErrorMiBolsa, renderizarMiBolsa, validarMiBolsa } from "./mi-bolsa.js";

const participacion = Object.freeze({
  bolsa: "Bolsa de auxiliares administrativos",
  categoria: "Auxiliar administrativo/a",
  version: 2,
  orden_inicial: 3,
  total_instantanea: 42,
  estado_bolsa: "Vigente",
  vigente_desde: "2026-09-01",
  vigente_hasta: null,
});

function respuestaJSON(data, status = 200) {
  const texto = JSON.stringify(data);
  return { status, headers: { get: (nombre) => ({ "Content-Type": "application/json; charset=utf-8", "Content-Length": String(new TextEncoder().encode(texto).byteLength) })[nombre] ?? null }, text: async () => texto };
}

test("mi bolsa usa solo GET sin cuerpo, query ni credenciales", async () => {
  let peticion;
  const cliente = crearClienteMiBolsa({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuestaJSON({ data: { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-19T10:00:00Z", participaciones: [participacion] } });
  } });
  const datos = await cliente.cargar();
  assert.equal(datos.participaciones[0].orden_inicial, 3);
  assert.equal(peticion.ruta, "/api/vec/bolsa/mi-bolsa");
  assert.equal(peticion.opciones.method, "GET");
  assert.equal(peticion.opciones.body, undefined);
  assert.equal(peticion.opciones.credentials, "omit");
  assert.equal(peticion.opciones.headers.Authorization, undefined);
});

test("mi bolsa rechaza esquema, totales inconsistentes y campos obligatorios", () => {
  const base = { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-19T10:00:00Z", participaciones: [participacion] };
  assert.equal(validarMiBolsa(base).esquema, "vec.bolsa.mi-bolsa.v1");
  assert.throws(() => validarMiBolsa({ ...base, esquema: "otro" }), /esquema/u);
  assert.throws(() => validarMiBolsa({ ...base, participaciones: [{ ...participacion, total_instantanea: 2 }] }), /total_instantanea/u);
  assert.throws(() => validarMiBolsa({ ...base, participaciones: [{ ...participacion, vigente_hasta: undefined }] }), /vigente_hasta/u);
  assert.throws(() => validarMiBolsa({ ...base, participaciones: [{ ...participacion, vigente_desde: "2026-02-31" }] }), /calendario válida/u);
});

test("mi bolsa trata 403 como denegación y no muestra puntuación ni datos de llamamiento", async () => {
  const cliente = crearClienteMiBolsa({ fetchImpl: async () => ({ status: 403, headers: { get: () => null }, text: async () => "" }) });
  await assert.rejects(() => cliente.cargar(), (error) => error instanceof ErrorMiBolsa && error.codigo === "acceso_denegado");
  const html = renderizarMiBolsa(validarMiBolsa({ esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-19T10:00:00Z", participaciones: [participacion] }));
  assert.match(html, /Orden inicial en la constitución/u);
  assert.match(html, /no indica la posición actual/u);
  assert.doesNotMatch(html, /Puntuación|Último llamamiento|Situación actual/u);
});
