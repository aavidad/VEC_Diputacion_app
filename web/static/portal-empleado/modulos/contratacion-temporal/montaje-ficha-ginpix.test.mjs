import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js?v=VEC_WEB_VERSION";
import { RUTA_FICHA_GINPIX, NOMBRE_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js?v=VEC_WEB_VERSION";
import { consultaB2GINPIX, ficheroB2GINPIX } from "./ficha-ginpix-fixture.mjs";

const consulta = consultaB2GINPIX();
const fichero = ficheroB2GINPIX();

test("ficha: agregador y cliente conservan bytes con GET único sin cuerpo", async () => {
  const bytes = new TextEncoder().encode(JSON.stringify(fichero));
  let peticiones = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    peticiones++;
    assert.equal(ruta, `${RUTA_FICHA_GINPIX}?expediente_ref=${encodeURIComponent(consulta.expediente_ref)}`);
    assert.equal(opciones.method, "GET"); assert.equal(opciones.body, undefined);
    assert.equal(opciones.redirect, "error"); assert.equal(opciones.cache, "no-store");
    assert.deepEqual([...opciones.headers.keys()], ["accept"]);
    return new Response(bytes, { status: 200, headers: {
      "content-type": "application/json; charset=utf-8", "content-length": String(bytes.length),
      "content-disposition": `attachment; filename=${NOMBRE_FICHA_GINPIX}`,
    } });
  } });
  const descarga = await cliente.descargarFichaGINPIX(consulta);
  assert.deepEqual(descarga.contenido, bytes); assert.deepEqual(descarga.json, fichero);
  assert.equal(peticiones, 1);
});

test("ficha: errores nominales conservan estado y nunca producen archivo", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [409, "recibo_no_confirmado"], [503, "servicio_no_disponible"]]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => new Response(JSON.stringify({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.ficha_ginpix.error.${codigo}`, correlacion_ref: "corr_no_disponible",
    } }), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } }) });
    await assert.rejects(cliente.descargarFichaGINPIX(consulta), (e) => e.estado === estado && e.codigo === codigo && e.envelopeValido === true);
  }
});

test("ficha: rechazo de recibo inválido antes de red y de attachment divergente", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => {
    llamadas++;
    return new Response(JSON.stringify(fichero), { headers: { "content-type": "application/json; charset=utf-8", "content-disposition": "inline" } });
  } });
  await assert.rejects(cliente.descargarFichaGINPIX(null)); assert.equal(llamadas, 0);
  await assert.rejects(cliente.descargarFichaGINPIX(consulta), (e) => e.codigo === "respuesta_incompatible");
  assert.equal(llamadas, 1);
});
