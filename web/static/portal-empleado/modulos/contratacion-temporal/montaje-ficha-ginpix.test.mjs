import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { RUTA_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";

const recibo = {
  esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
  expediente_ref: "expediente:ct:ficha", solicitud_personal_ref: "solicitud:ficha",
  relacion_ref: "relacion:ficha", recibo_ref: "recibo:ficha", actuacion_ref: "actuacion:ficha",
  registrada_en: "2026-09-10T13:07:06Z", periodo_incorporacion: { desde: "2027-01-01T00:00:00Z", hasta: "2027-03-31T00:00:00Z" },
  version_solicitud_personal: 7, version_actual_expediente: 8, seguimiento_ref: "seguimiento:ficha",
  version_seguimiento_anterior: 0, version_seguimiento_resultante: 1,
  auditoria_ref: "auditoria:ficha", outbox_ref: "outbox:ficha",
  ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false,
};
const fichero = {
  esquema: "vec.dipgra.contratacion-temporal.ginpix.fichero.v1", version: 1,
  metadatos: {
    esquema_modelo: "vec.dipgra.contratacion-temporal.ginpix.modelo.v1",
    esquema_mapeo: "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1",
    esquema_carga: "vec.dipgra.contratacion-temporal.ginpix.carga.v1",
    version_expediente: 8, expediente_ref: recibo.expediente_ref, incorporacion_ref: recibo.actuacion_ref,
    procedencia_modelo_ref: recibo.recibo_ref, correlacion_ref: "correlacion:original", idempotencia_ref: "idempotencia:original",
    huella_modelo_sha256: "a".repeat(64), mapeo_ref: "mapeo:ficha", mapeo_version: 1,
    procedencia_mapeo_ref: "configuracion:ficha", huella_mapeo_sha256: "b".repeat(64), huella_carga_sha256: "c".repeat(64),
  },
  campos: [{ clave: "actuacion_ref", estado: "valor", valor: recibo.actuacion_ref }],
};

test("ficha: agregador y cliente conservan bytes con GET único sin cuerpo", async () => {
  const bytes = new TextEncoder().encode(JSON.stringify(fichero));
  let peticiones = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    peticiones++;
    assert.equal(ruta, `${RUTA_FICHA_GINPIX}?expediente_ref=${encodeURIComponent(recibo.expediente_ref)}`);
    assert.equal(opciones.method, "GET"); assert.equal(opciones.body, undefined);
    assert.equal(opciones.redirect, "error"); assert.equal(opciones.cache, "no-store");
    assert.deepEqual([...opciones.headers.keys()], ["accept"]);
    return new Response(bytes, { status: 200, headers: {
      "content-type": "application/json; charset=utf-8", "content-length": String(bytes.length),
      "content-disposition": "attachment; filename=ficha-ginpix-ejercicio.json",
    } });
  } });
  const descarga = await cliente.descargarFichaGINPIX(recibo);
  assert.deepEqual(descarga.contenido, bytes); assert.deepEqual(descarga.json, fichero);
  assert.equal(peticiones, 1);
});

test("ficha: errores nominales conservan estado y nunca producen archivo", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [409, "recibo_no_confirmado"], [503, "servicio_no_disponible"]]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => new Response(JSON.stringify({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.ficha_ginpix.error.${codigo}`, correlacion_ref: "corr_no_disponible",
    } }), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } }) });
    await assert.rejects(cliente.descargarFichaGINPIX(recibo), (e) => e.estado === estado && e.codigo === codigo && e.envelopeValido === true);
  }
});

test("ficha: rechazo de recibo inválido antes de red y de attachment divergente", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => {
    llamadas++;
    return new Response(JSON.stringify(fichero), { headers: { "content-type": "application/json; charset=utf-8", "content-disposition": "inline" } });
  } });
  await assert.rejects(cliente.descargarFichaGINPIX(null)); assert.equal(llamadas, 0);
  await assert.rejects(cliente.descargarFichaGINPIX(recibo), (e) => e.codigo === "respuesta_incompatible");
  assert.equal(llamadas, 1);
});
