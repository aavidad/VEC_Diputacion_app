import assert from "node:assert/strict";
import test from "node:test";

import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { RUTA_PREPARACION_COBERTURA_VIGENTE } from "./cliente-http-preparacion-cobertura.js";

const RESPUESTA = { data: {
  esquema: "vec.contratacion-temporal.preparacion-cobertura.v1",
  catalogo: { referencia: "catalogo:ct:preparacion:v2", version: 2,
    huella_sha256: "a".repeat(64), es_ejemplo: true, vias: [
      { clave: "bolsa_vigente", orden: 1, documentos: [], datos: [] },
      { clave: "oferta_sae", orden: 2, documentos: [], datos: [] },
    ] },
} };

function respuestaJSON(datos, status = 200) {
  return new Response(JSON.stringify(datos), { status,
    headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("GET previo al alta usa ruta sin cuerpo, caché ni identidad del navegador", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push([ruta, opciones]);
      return respuestaJSON(RESPUESTA);
    },
  });
  const resultado = await cliente.obtenerPreparacionCoberturaVigente();
  assert.equal(resultado.catalogo.vias.length, 2);
  assert.equal(resultado.catalogo.es_ejemplo, true);
  assert.equal(Object.isFrozen(resultado.catalogo.vias[0]), true);
  assert.equal(llamadas.length, 1);
  const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, RUTA_PREPARACION_COBERTURA_VIGENTE);
  assert.equal(opciones.method, "GET");
  assert.equal(opciones.body, undefined);
  assert.equal(opciones.cache, "no-store");
  assert.equal(opciones.mode, "same-origin");
  assert.equal(opciones.referrerPolicy, "no-referrer");
  assert.equal(opciones.headers.get("content-type"), null);
});

test("GET falla cerrado ante lista parcial, perfil sin datos y ausencia de publicación", async () => {
  const incompleta = structuredClone(RESPUESTA);
  incompleta.data.catalogo.vias.pop();
  for (const [datos, estado, codigo] of [
    [incompleta, 200, "respuesta_incompatible"],
    [{ error: { codigo: "datos_no_disponibles_perfil",
      clave_i18n: "api.contratacion_temporal.cobertura.error.datos_no_disponibles_perfil",
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef" } }, 403, "datos_no_disponibles_perfil"],
    [{ error: { codigo: "servicio_no_disponible",
      clave_i18n: "api.contratacion_temporal.cobertura.error.servicio_no_disponible",
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef" } }, 503, "servicio_no_disponible"],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuestaJSON(datos, estado),
    });
    await assert.rejects(cliente.obtenerPreparacionCoberturaVigente(),
      (error) => error.codigo === codigo && !Object.hasOwn(error, "catalogo"));
  }
});
