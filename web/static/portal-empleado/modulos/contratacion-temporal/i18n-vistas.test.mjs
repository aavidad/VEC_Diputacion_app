import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { prepararTextosContratacionVista } from "./i18n-vistas.js";

const leerJSON = async (url) => JSON.parse(await readFile(url, "utf8"));

for (const idioma of ["es", "en"]) {
  test(`el cuadro ${idioma} lee solo las tres fuentes de la lista`, async () => {
    const llamadas = [];
    const resultado = await prepararTextosContratacionVista("cuadro", { idioma, leer: async (url) => {
      llamadas.push(url.pathname.split("/textos/")[1]);
      return leerJSON(url);
    } });
    assert.deepEqual(llamadas.sort(), [
      `${idioma}/portal.json`,
      `${idioma}/contratacion-temporal-ficha-lista.json`,
      `${idioma}/contratacion-temporal-lista-plazos.json`,
    ].sort());
    assert.equal(resultado.idioma, idioma);
    assert.equal(resultado.incidencias.length, 0);
    assert.ok(resultado.secciones["portal.fases_rrhh"].fase_solicitud);
    assert.ok(Object.keys(resultado.secciones["contratacion-temporal-ficha-lista.general"]).length > 0);
  });
}

test("el detalle añade sus tres fuentes al abrir expediente y lee cada JSON una vez", async () => {
  const llamadas = [];
  const resultado = await prepararTextosContratacionVista("expediente", { idioma: "en", leer: async (url) => {
    llamadas.push(url.pathname.split("/textos/")[1]);
    return leerJSON(url);
  } });
  assert.deepEqual(llamadas.sort(), [
    "en/portal.json", "en/contratacion-temporal-ficha-lista.json",
    "en/contratacion-temporal-lista-plazos.json", "en/contratacion-temporal-analisis-catalogo.json",
    "en/contratacion-temporal-cambios-expediente.json",
    "en/contratacion-temporal-firma-incorporacion-expedientes.json",
  ].sort());
  assert.equal(resultado.idioma, "en");
  assert.ok(resultado.secciones["contratacion-temporal-firma-incorporacion-expedientes.firma"]);
});

test("un catálogo inválido aislado se reintenta y conserva el idioma inglés", async () => {
  const llamadas = [];
  let caida = true;
  const resultado = await prepararTextosContratacionVista("cuadro", { idioma: "en", leer: async (url) => {
    llamadas.push(url.pathname);
    if (caida && url.pathname.endsWith("/contratacion-temporal-ficha-lista.json")) {
      caida = false;
      return {};
    }
    return leerJSON(url);
  } });
  assert.equal(resultado.idioma, "en");
  assert.equal(llamadas.filter((ruta) => ruta.endsWith("/en/contratacion-temporal-ficha-lista.json")).length, 2);
  assert.ok(llamadas.every((ruta) => ruta.includes("/en/")));
});

test("si falla el inglés, el cuadro se prepara íntegro en español y admite reintento", async () => {
  let caido = true;
  const leer = async (url) => {
    if (caido && url.pathname.includes("/en/contratacion-temporal-ficha-lista.json")) {
      throw new Error("503 persistente de prueba");
    }
    return leerJSON(url);
  };
  const respaldo = await prepararTextosContratacionVista("cuadro", { idioma: "en", leer });
  assert.equal(respaldo.idioma, "es");
  assert.equal(respaldo.reintentar, true);
  assert.ok(Object.values(respaldo.secciones).every((seccion) => Object.keys(seccion).length > 0));
  caido = false;
  const recuperado = await prepararTextosContratacionVista("cuadro", { idioma: "en", leer, reintentar: true });
  assert.equal(recuperado.idioma, "en");
  assert.equal(recuperado.reintentar, false);
});

test("vista inválida y catálogo por defecto caído no producen un cuadro vacío", async () => {
  await assert.rejects(prepararTextosContratacionVista("desconocida"), /vista CT/u);
  await assert.rejects(prepararTextosContratacionVista("cuadro", { idioma: "zz" }), /idioma CT no admitido/u);
  await assert.rejects(prepararTextosContratacionVista("cuadro", {
    idioma: "es", leer: async () => { throw new Error("503 persistente de prueba"); },
  }), /503 persistente/u);
});
