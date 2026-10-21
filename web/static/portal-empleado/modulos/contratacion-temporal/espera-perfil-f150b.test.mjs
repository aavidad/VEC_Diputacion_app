import assert from "node:assert/strict";
import test from "node:test";

import { claveI18nValida, codigoValidoParaRuta } from "./cliente-http-transporte.js";
import { RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js?v=VEC_WEB_VERSION";

test("el error de espera de la frontera se admite en las rutas dinámicas", () => {
  for (const ruta of [
    "/api/vec/contratacion-temporal/llamamientos/seleccion",
    "/api/vec/contratacion-temporal/seguimiento-cese",
    "/api/vec/contratacion-temporal/formalizacion/propuestas",
  ]) {
    const codigo = "servicio_no_disponible";
    const referenciaI18n = "api.vec.ruta_exacta.error.servicio_no_disponible";
    assert.equal(codigoValidoParaRuta(ruta, 503, codigo, RUTAS_HTTP_CONTRATACION_TEMPORAL), true);
    assert.equal(claveI18nValida(ruta, codigo, referenciaI18n, RUTAS_HTTP_CONTRATACION_TEMPORAL), true);
    assert.equal(claveI18nValida(ruta, codigo,
      "api.contratacion_temporal.catalogos_alta.error.servicio_no_disponible", RUTAS_HTTP_CONTRATACION_TEMPORAL), false);
  }
});
