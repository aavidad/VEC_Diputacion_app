import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const fuente = (await readFile(new URL("./i18n-catalogos.js", import.meta.url), "utf8"))
  .replace(/^import .*;\n/gmu, "")
  .replaceAll("export async function", "async function");

function preparar(lector) {
  const crear = new Function("lector", `
    let IDIOMA_ACTUAL = "und";
    const prepararIdiomas = async () => { IDIOMA_ACTUAL = "en"; };
    const cargarTextos = lector;
    ${fuente}
    return { cargarCatalogosContratacion, cargarCatalogosContratacionEnIdioma };
  `);
  return crear(lector);
}

function datos(modulo, idioma) {
  return {
    idioma, incidenciaCatalogo: null, incidenciaIndice: null,
    seccion: (seccion) => modulo === "contratacion-temporal-compatibilidad"
      ? { ES: "es", EN: "en" } : { titulo: `${idioma}:${seccion}` },
  };
}

test("prepara el índice antes de elegir el idioma predeterminado y pide solo ese idioma", async () => {
  const leidos = [];
  const helper = preparar(async (modulo, { idioma }) => {
    leidos.push(`${idioma}/${modulo}`);
    return datos(modulo, idioma);
  });
  const catalogos = await helper.cargarCatalogosContratacion("contratacion-temporal-prueba");
  assert.deepEqual(leidos.sort(), ["en/contratacion-temporal-compatibilidad", "en/contratacion-temporal-prueba"]);
  assert.equal(catalogos.idioma, "en");
  assert.deepEqual(Object.keys(catalogos.porIdioma), ["en"]);
  assert.equal(catalogos.exportaciones.EN, catalogos.actual);
  assert.equal(catalogos.exportaciones.ES, undefined);
});

test("el idioma explícito se consulta por separado y el respaldo común informa su idioma real", async () => {
  const leidos = [];
  const helper = preparar(async (modulo, { idioma }) => {
    leidos.push(`${idioma}/${modulo}`);
    return datos(modulo, modulo === "contratacion-temporal-prueba" && idioma === "en" ? "es" : idioma);
  });
  const explicito = await helper.cargarCatalogosContratacionEnIdioma("contratacion-temporal-prueba", "es");
  assert.equal(explicito.actual.titulo, "es:general");
  assert.ok(leidos.every((ruta) => ruta.startsWith("es/")));
  leidos.length = 0;
  const respaldo = await helper.cargarCatalogosContratacion("contratacion-temporal-prueba");
  assert.equal(respaldo.idioma, "es");
  assert.equal(respaldo.exportaciones.ES, respaldo.actual);
  assert.equal(respaldo.exportaciones.EN, undefined);
  assert.deepEqual(Object.keys(respaldo.porIdioma), ["es"]);
  assert.ok(leidos.every((ruta) => ruta.startsWith("en/")));
});

test("el helper no memoriza un rechazo y permite reintentar en la misma sesión", async () => {
  let caido = true;
  const helper = preparar(async (modulo, { idioma }) => {
    if (caido && modulo === "contratacion-temporal-prueba") throw new Error("caída de prueba");
    return datos(modulo, idioma);
  });
  await assert.rejects(helper.cargarCatalogosContratacion("contratacion-temporal-prueba"), /caída de prueba/u);
  caido = false;
  assert.equal((await helper.cargarCatalogosContratacion("contratacion-temporal-prueba")).actual.titulo, "en:general");
});
