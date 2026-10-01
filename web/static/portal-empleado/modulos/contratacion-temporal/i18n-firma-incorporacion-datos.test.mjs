import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearTraductorCircuitoFirma, MENSAJES_CIRCUITO_FIRMA_ES, MENSAJES_CIRCUITO_FIRMA_EN } from "./i18n-circuito-firma.js";
import { MENSAJES_FIRMA_REMISION_ES, MENSAJES_FIRMA_REMISION_EN } from "./i18n-firma-remision.js";
import { MENSAJES_SEGUIMIENTO_CESE, MENSAJES_SEGUIMIENTO_CESE_EN } from "./i18n-seguimiento-cese.js";
import {
  MENSAJES_FIRMA_INCORPORACION_PORTAL_ES, MENSAJES_FIRMA_INCORPORACION_PORTAL_EN,
  MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_ES, MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_EN,
} from "./i18n-firma-incorporacion-datos.js";
import { MENSAJES_CONTRATACION_TEMPORAL_ES, MENSAJES_CONTRATACION_TEMPORAL_EN } from "./i18n.js";
import { MENSAJES_EXPEDIENTES_CONTRATACION_ES, MENSAJES_EXPEDIENTES_CONTRATACION_EN } from "./i18n-expedientes.js";
import { IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";

const conjuntos = [
  ["contratacion-temporal-circuito-firma", MENSAJES_CIRCUITO_FIRMA_ES, MENSAJES_CIRCUITO_FIRMA_EN],
  ["contratacion-temporal-firma-remision", MENSAJES_FIRMA_REMISION_ES, MENSAJES_FIRMA_REMISION_EN],
  ["contratacion-temporal-seguimiento-cese", MENSAJES_SEGUIMIENTO_CESE, MENSAJES_SEGUIMIENTO_CESE_EN],
  ["contratacion-temporal-firma-incorporacion-portal", MENSAJES_FIRMA_INCORPORACION_PORTAL_ES, MENSAJES_FIRMA_INCORPORACION_PORTAL_EN],
  ["contratacion-temporal-firma-incorporacion-expedientes", MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_ES, MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_EN],
];

const variables = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/gu)].map(([, nombre]) => nombre).sort();

test("los catálogos conservan claves, exportaciones y variables en ambos idiomas", async () => {
  const [es, en] = IDIOMAS_DISPONIBLES;
  assert.ok(es && en, "el índice requiere dos idiomas");
  for (const [modulo, mensajesES, mensajesEN] of conjuntos) {
    const ruta = (codigo) => new URL(`../../../textos/${codigo}/${modulo}.json`, import.meta.url);
    const datosES = JSON.parse(await readFile(ruta(es.codigo), "utf8")).general;
    const datosEN = JSON.parse(await readFile(ruta(en.codigo), "utf8")).general;
    assert.deepEqual(mensajesES, datosES, `${modulo}: exportación ES`);
    assert.deepEqual(mensajesEN, datosEN, `${modulo}: exportación EN`);
    assert.deepEqual(Object.keys(datosES).sort(), Object.keys(datosEN).sort(), `${modulo}: claves`);
    for (const clave of Object.keys(datosES)) {
      assert.ok(datosES[clave] && datosEN[clave], `${modulo}.${clave}: texto vacío`);
      assert.deepEqual(variables(datosES[clave]), variables(datosEN[clave]), `${modulo}.${clave}: variables`);
    }
  }
});

test("los dos agregadores conservan todos los valores extraídos", () => {
  for (const [extraido, actual] of [
    [MENSAJES_FIRMA_INCORPORACION_PORTAL_ES, MENSAJES_CONTRATACION_TEMPORAL_ES],
    [MENSAJES_FIRMA_INCORPORACION_PORTAL_EN, MENSAJES_CONTRATACION_TEMPORAL_EN],
    [MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_ES, MENSAJES_EXPEDIENTES_CONTRATACION_ES],
    [MENSAJES_FIRMA_INCORPORACION_EXPEDIENTES_EN, MENSAJES_EXPEDIENTES_CONTRATACION_EN],
  ]) {
    for (const [clave, valor] of Object.entries(extraido)) assert.equal(actual[clave], valor, clave);
  }
});

test("el traductor de firma conserva idioma, interpolación, sobrescritura y fallo cerrado", () => {
  const [es, en] = IDIOMAS_DISPONIBLES;
  const tES = crearTraductorCircuitoFirma({}, es.localizacion);
  const tEN = crearTraductorCircuitoFirma({}, en.localizacion);
  assert.equal(tES("circuito_firma_pasos", { documento: "PDF" }), MENSAJES_CIRCUITO_FIRMA_ES.circuito_firma_pasos.replace("{documento}", "PDF"));
  assert.equal(tEN("circuito_firma_pasos", { documento: "PDF" }), MENSAJES_CIRCUITO_FIRMA_EN.circuito_firma_pasos.replace("{documento}", "PDF"));
  assert.equal(crearTraductorCircuitoFirma({ circuito_firma_titulo: "Otro" })("circuito_firma_titulo"), "Otro");
  assert.throws(() => tES("clave_desconocida"), /falta la traducción/u);
});
