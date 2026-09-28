import assert from "node:assert/strict";
import test from "node:test";

globalThis.location = { href: "https://vec.example/portal-empleado/?lang=es" };
const { IDIOMA_ACTUAL } = await import("../comun/idioma.js");
const { AYUDA_PORTAL_BOLSA, AYUDA_PORTAL_RRHH } = await import("./ayuda-contenido.js");
const { MENSAJES_AYUDA_INICIO_RRHH_EN, MENSAJES_AYUDA_PORTAL_ES } = await import("./portal-i18n-ayuda.js");
const { crearTraductorPortal, MENSAJES_PORTAL_ES } = await import("./portal-i18n.js");

const CLAVES_INICIO = Object.keys(MENSAJES_AYUDA_INICIO_RRHH_EN)
  .filter((clave) => clave.startsWith("ayuda_rrhh_portada_"));

test("la ayuda española de Inicio RRHH orienta sobre las tres áreas sin describir un llamamiento", () => {
  assert.equal(IDIOMA_ACTUAL, "es");
  const traducir = crearTraductorPortal(MENSAJES_PORTAL_ES);
  assert.equal(AYUDA_PORTAL_RRHH.esquema, "vec.portal.ayuda.v1");
  assert.equal(AYUDA_PORTAL_RRHH.titulo, traducir("ayuda_rrhh_portada_titulo"));
  assert.equal(AYUDA_PORTAL_RRHH.introduccion, traducir("ayuda_rrhh_portada_introduccion"));
  assert.equal(AYUDA_PORTAL_RRHH.pasos.length, 4);
  assert.equal(AYUDA_PORTAL_RRHH.preguntas.length, 3);
  const visible = [AYUDA_PORTAL_RRHH.titulo, AYUDA_PORTAL_RRHH.introduccion,
    ...AYUDA_PORTAL_RRHH.pasos, ...AYUDA_PORTAL_RRHH.preguntas.flatMap((item) => [item.pregunta, item.respuesta]),
    AYUDA_PORTAL_RRHH.transcripcion].join(" ");
  assert.match(visible, /expedientes en trámite|Trámites recientes/u);
  assert.match(visible, /Bolsas de trabajo/u);
  assert.match(visible, /Ofertas al SAE/u);
  assert.match(visible, /fuente autorizada|accesos de su sesión/u);
  assert.match(visible, /no registra ninguna actuación/u);
  assert.doesNotMatch(visible, /gestionar un llamamiento|selecciona personas|prelación/u);
  assert.notEqual(AYUDA_PORTAL_RRHH.titulo, AYUDA_PORTAL_BOLSA.titulo);
});

test("las claves de ayuda RRHH tienen pareja inglesa y conservan los parámetros", () => {
  assert.equal(CLAVES_INICIO.length, 13);
  for (const clave of Object.keys(MENSAJES_AYUDA_INICIO_RRHH_EN)) {
    assert.ok(typeof MENSAJES_AYUDA_PORTAL_ES[clave] === "string" && MENSAJES_AYUDA_PORTAL_ES[clave]);
    assert.ok(MENSAJES_AYUDA_INICIO_RRHH_EN[clave]);
    const variables = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/gu)].map((coincidencia) => coincidencia[1]).sort();
    assert.deepEqual(variables(MENSAJES_AYUDA_PORTAL_ES[clave]), variables(MENSAJES_AYUDA_INICIO_RRHH_EN[clave]), clave);
  }
});
