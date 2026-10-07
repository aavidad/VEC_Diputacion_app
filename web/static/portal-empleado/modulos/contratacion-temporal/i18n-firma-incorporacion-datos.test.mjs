import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearTraductorCircuitoFirma, MENSAJES_CIRCUITO_FIRMA_ES, MENSAJES_CIRCUITO_FIRMA_EN } from "./i18n-circuito-firma.js?v=20261001-ct-firma-verificador-v2";
import { MENSAJES_FIRMA_REMISION_ES, MENSAJES_FIRMA_REMISION_EN } from "./i18n-firma-remision.js?v=20261001-ct-a-i18n-v1";
import { MENSAJES_SEGUIMIENTO_CESE, MENSAJES_SEGUIMIENTO_CESE_EN } from "./i18n-seguimiento-cese.js";
import { MENSAJES_FIRMA_INCORPORACION } from "./i18n-firma-incorporacion-datos.js?v=20261001-ct-a-i18n-v1";
import { MENSAJES_CONTRATACION_TEMPORAL_ES, MENSAJES_CONTRATACION_TEMPORAL_EN } from "./i18n.js";
import { MENSAJES_EXPEDIENTES_CONTRATACION_ES, MENSAJES_EXPEDIENTES_CONTRATACION_EN } from "./i18n-expedientes.js";
import { IDIOMA_ACTUAL, IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";

const conjuntos = [
  ["contratacion-temporal-circuito-firma", MENSAJES_CIRCUITO_FIRMA_ES, MENSAJES_CIRCUITO_FIRMA_EN],
  ["contratacion-temporal-firma-remision", MENSAJES_FIRMA_REMISION_ES, MENSAJES_FIRMA_REMISION_EN],
  ["contratacion-temporal-seguimiento-cese", MENSAJES_SEGUIMIENTO_CESE, MENSAJES_SEGUIMIENTO_CESE_EN],
  ["contratacion-temporal-firma-incorporacion-portal", MENSAJES_FIRMA_INCORPORACION.portal.ES, MENSAJES_FIRMA_INCORPORACION.portal.EN],
  ["contratacion-temporal-firma-incorporacion-expedientes", MENSAJES_FIRMA_INCORPORACION.expedientes.ES, MENSAJES_FIRMA_INCORPORACION.expedientes.EN],
];

const variables = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/gu)].map(([, nombre]) => nombre).sort();

test("los catálogos conservan claves y variables en ambos idiomas; las exportaciones siguen el activo", async () => {
  const [es, en] = IDIOMAS_DISPONIBLES;
  assert.ok(es && en, "el índice requiere dos idiomas");
  for (const [modulo, mensajesES, mensajesEN] of conjuntos) {
    const ruta = (codigo) => new URL(`../../../textos/${codigo}/${modulo}.json`, import.meta.url);
    const seccionesES = JSON.parse(await readFile(ruta(es.codigo), "utf8"));
    const seccionesEN = JSON.parse(await readFile(ruta(en.codigo), "utf8"));
    assert.deepEqual(Object.keys(seccionesES), Object.keys(seccionesEN), `${modulo}: secciones`);
    const mensajes = (secciones) => Object.assign({}, ...Object.entries(secciones)
      .filter(([nombre]) => nombre !== "valores_controlados")
      .map(([, valores]) => valores));
    const datosES = mensajes(seccionesES);
    const datosEN = mensajes(seccionesEN);
    if (seccionesES.valores_controlados) {
      assert.deepEqual(seccionesES.valores_controlados, seccionesEN.valores_controlados);
    }
    const datosActivos = IDIOMA_ACTUAL === es.codigo ? datosES : datosEN;
    assert.deepEqual(IDIOMA_ACTUAL === es.codigo ? mensajesES : mensajesEN, datosActivos,
      `${modulo}: catálogo activo`);
    assert.equal(IDIOMA_ACTUAL === es.codigo ? mensajesEN : mensajesES, undefined,
      `${modulo}: el idioma inactivo se carga sólo de forma explícita`);
    assert.deepEqual(Object.keys(datosES).sort(), Object.keys(datosEN).sort(), `${modulo}: claves`);
    for (const clave of Object.keys(datosES)) {
      assert.ok(datosES[clave] && datosEN[clave], `${modulo}.${clave}: texto vacío`);
      assert.deepEqual(variables(datosES[clave]), variables(datosEN[clave]), `${modulo}.${clave}: variables`);
    }
  }
});

test("los dos agregadores conservan todos los valores extraídos", () => {
  for (const [extraido, actual] of [
    [MENSAJES_FIRMA_INCORPORACION.portal.ES, MENSAJES_CONTRATACION_TEMPORAL_ES],
    [MENSAJES_FIRMA_INCORPORACION.portal.EN, MENSAJES_CONTRATACION_TEMPORAL_EN],
    [MENSAJES_FIRMA_INCORPORACION.expedientes.ES, MENSAJES_EXPEDIENTES_CONTRATACION_ES],
    [MENSAJES_FIRMA_INCORPORACION.expedientes.EN, MENSAJES_EXPEDIENTES_CONTRATACION_EN],
  ]) {
    if (extraido === undefined) continue;
    for (const [clave, valor] of Object.entries(extraido)) assert.equal(actual[clave], valor, clave);
    assert.deepEqual(Object.keys(extraido), Object.keys(actual).filter((clave) => Object.hasOwn(extraido, clave)));
  }
});

test("el traductor de firma conserva idioma, interpolación, sobrescritura y fallo cerrado", async () => {
  const [es, en] = IDIOMAS_DISPONIBLES;
  const datos = async (codigo) => JSON.parse(await readFile(
    new URL(`../../../textos/${codigo}/contratacion-temporal-circuito-firma.json`, import.meta.url), "utf8"));
  const catalogoES = (await datos(es.codigo)).general;
  const catalogoEN = (await datos(en.codigo)).general;
  const tES = crearTraductorCircuitoFirma(catalogoES, es.localizacion);
  const tEN = crearTraductorCircuitoFirma(catalogoEN, en.localizacion);
  assert.equal(tES("circuito_firma_pasos", { documento: "PDF" }), catalogoES.circuito_firma_pasos.replace("{documento}", "PDF"));
  assert.equal(tEN("circuito_firma_pasos", { documento: "PDF" }), catalogoEN.circuito_firma_pasos.replace("{documento}", "PDF"));
  assert.equal(crearTraductorCircuitoFirma({ circuito_firma_titulo: "Otro" })("circuito_firma_titulo"), "Otro");
  assert.throws(() => tES("clave_desconocida"), /falta la traducción/u);
});
