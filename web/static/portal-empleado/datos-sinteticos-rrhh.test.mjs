import assert from "node:assert/strict";
import test from "node:test";

import {
  ATLAS_SINTETICO_RRHH,
  ESQUEMA_ATLAS_SINTETICO_RRHH,
  TEXTO_DATOS_FICTICIOS_RRHH,
  obtenerAtlasSinteticoRRHH,
  validarAtlasSinteticoRRHH,
} from "./datos-sinteticos-rrhh.js";

test("el atlas tiene esquema cerrado, marcas y aviso común", () => {
  const atlas = obtenerAtlasSinteticoRRHH();
  assert.equal(atlas.esquema, ESQUEMA_ATLAS_SINTETICO_RRHH);
  assert.equal(atlas.aviso_visible, TEXTO_DATOS_FICTICIOS_RRHH);
  assert.equal(atlas.sintetico, true);
  assert.equal(atlas.uso, "presentacion_rrhh");
  assert.equal(Object.isFrozen(atlas), true);
  assert.equal(Object.isFrozen(atlas.persona_principal), true);
  assert.throws(() => validarAtlasSinteticoRRHH({ ...atlas, campo_no_declarado: true }), /esquema cerrado/u);
});

test("la persona principal, el empleo, la relación y la unidad son coherentes", () => {
  const atlas = obtenerAtlasSinteticoRRHH();
  assert.equal(atlas.persona_principal.nombre_visible, "Antonio López Fernández");
  assert.equal(atlas.empleado.persona_ref, atlas.persona_principal.persona_ref);
  assert.equal(atlas.relacion.empleado_ref, atlas.empleado.empleado_ref);
  assert.equal(atlas.relacion.unidad_ref, atlas.unidad.unidad_ref);
  assert.equal(atlas.unidad.centro_ref, atlas.centro.centro_ref);
  assert.ok(atlas.localidades.some((localidad) => localidad.referencia === atlas.centro.localidad_ref));
});

test("cada entrega es defensiva y los datos no contienen términos ni PII sensible", () => {
  const primera = obtenerAtlasSinteticoRRHH();
  const segunda = obtenerAtlasSinteticoRRHH();
  assert.notEqual(primera, segunda);
  assert.notEqual(primera.persona_principal, segunda.persona_principal);
  assert.equal(ATLAS_SINTETICO_RRHH.persona_principal.nombre_visible, "Antonio López Fernández");
  const texto = JSON.stringify(primera).toLocaleLowerCase("es");
  assert.doesNotMatch(texto, /(?:demo|prueba|usuario|dni|nie|correo|email|tel[eé]fono|direcci[oó]n)/u);
  assert.doesNotMatch(texto, /\b\d{8}[a-z]\b/ui);
});
