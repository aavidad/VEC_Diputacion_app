import assert from "node:assert/strict";
import test from "node:test";
import { MENSAJES_RRHH_PLANTILLAS_ES, MENSAJES_RRHH_PLANTILLAS_EN } from "./rrhh-plantillas-i18n.js";
import { MENSAJES_REINCORPORACION_RRHH_ES, MENSAJES_REINCORPORACION_RRHH_EN } from "./rrhh-reincorporacion-i18n.js";
import { MENSAJES_BORRADORES_PUBLICADOS_ES, MENSAJES_BORRADORES_PUBLICADOS_EN } from "./i18n-borradores-publicados.js?v=20261001-ct-a-i18n-v1";

test("los tres catálogos RRHH C3/B55 conservan claves y marcadores en inglés", () => {
  for (const [es, en] of [
    [MENSAJES_RRHH_PLANTILLAS_ES, MENSAJES_RRHH_PLANTILLAS_EN],
    [MENSAJES_REINCORPORACION_RRHH_ES, MENSAJES_REINCORPORACION_RRHH_EN],
    [MENSAJES_BORRADORES_PUBLICADOS_ES, MENSAJES_BORRADORES_PUBLICADOS_EN],
  ]) {
    assert.deepEqual(Object.keys(en).sort(), Object.keys(es).sort());
    for (const clave of Object.keys(es)) {
      assert.ok(en[clave]?.trim(), clave);
      const marcadores = (valor) => [...valor.matchAll(/\{[a-z_]+\}/gu)].map((m) => m[0]).sort();
      assert.deepEqual(marcadores(en[clave]), marcadores(es[clave]), clave);
    }
  }
});

test("la recuperación reutiliza la petición y la publicación muestra su recibo", () => {
  assert.match(MENSAJES_RRHH_PLANTILLAS_EN.plantillas_rrhh_ayuda, /same request using its original key/u);
  assert.match(MENSAJES_RRHH_PLANTILLAS_EN.plantillas_rrhh_publicacion_indeterminada, /do not start another/u);
  assert.equal(MENSAJES_BORRADORES_PUBLICADOS_ES.bp_publicacion_recibo, "Recibo de publicación: {recibo}");
  assert.equal(MENSAJES_BORRADORES_PUBLICADOS_EN.bp_publicacion_recibo, "Publication receipt: {recibo}");
  assert.match(MENSAJES_REINCORPORACION_RRHH_EN.rrhh_reincorporacion_ayuda, /not to the substantive postholder/u);
});
