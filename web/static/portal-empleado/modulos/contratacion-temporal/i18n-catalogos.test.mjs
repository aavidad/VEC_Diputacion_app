import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const fuente = (await readFile(new URL("./i18n-catalogos.js", import.meta.url), "utf8"))
  .replace(/^import .*;\n/gmu, "")
  .replaceAll("export async function", "async function");

function preparar(idioma, fallo = () => false) {
  const peticiones = [];
  const disponibles = {
    "contratacion-temporal-compatibilidad": { idiomas_exportados: { ES: "es", EN: "en" } },
    "contratacion-temporal-prueba": { general: { titulo: "Texto del catálogo" } },
  };
  const lector = async (url) => {
    peticiones.push(url.href);
    if (fallo(url.href, peticiones.filter((peticion) => peticion === url.href).length)) {
      throw new Error("fallo de red de prueba");
    }
    return disponibles[url.href.split("/")[1]];
  };
  const crear = new Function("IDIOMA_ACTUAL", "IDIOMA_POR_DEFECTO", "localizacionDe", "crearTextos", "leerCatalogoUnaVez", "urlCatalogo", "console",
    `${fuente}\nreturn cargarCatalogosContratacion;`);
  return {
    peticiones,
    cargar: crear(idioma, "es", (codigo) => codigo,
      ({ respaldo }) => ({ seccion: (nombre) => respaldo[nombre] }),
      lector, (codigo, modulo) => ({ href: `${codigo}/${modulo}` }), { warn() {} }),
  };
}

test("lee solo el idioma activo y conserva una única sección para los consumidores", async () => {
  const { cargar, peticiones } = preparar("en");
  const catalogos = await cargar("contratacion-temporal-prueba");
  assert.deepEqual(peticiones.sort(), [
    "en/contratacion-temporal-compatibilidad", "en/contratacion-temporal-prueba",
  ]);
  assert.deepEqual(Object.keys(catalogos.porIdioma), ["en"]);
  assert.equal(catalogos.actual.titulo, "Texto del catálogo");
  assert.equal(catalogos.exportaciones.ES, catalogos.actual);
  assert.equal(catalogos.exportaciones.EN, catalogos.actual);
});

test("reintenta un fallo aislado sin pedir el idioma de respaldo", async () => {
  const { cargar, peticiones } = preparar("en", (ruta, intento) =>
    ruta === "en/contratacion-temporal-prueba" && intento === 1);
  await cargar("contratacion-temporal-prueba");
  assert.equal(peticiones.filter((ruta) => ruta === "en/contratacion-temporal-prueba").length, 2);
  assert.ok(peticiones.every((ruta) => ruta.startsWith("en/")));
});

test("usa el respaldo tras dos fallos y permite volver a intentar después", async () => {
  let caido = true;
  const { cargar, peticiones } = preparar("en", (ruta) =>
    caido && ruta === "en/contratacion-temporal-prueba");
  const respaldo = await cargar("contratacion-temporal-prueba");
  assert.equal(respaldo.actual.titulo, "Texto del catálogo");
  assert.equal(peticiones.filter((ruta) => ruta === "es/contratacion-temporal-prueba").length, 1);
  caido = false;
  const recuperado = await cargar("contratacion-temporal-prueba");
  assert.equal(recuperado.actual.titulo, "Texto del catálogo");
  assert.equal(peticiones.filter((ruta) => ruta === "en/contratacion-temporal-prueba").length, 3);
});

test("un fallo de ambos catálogos se informa y no queda memorizado por el helper", async () => {
  let caido = true;
  const { cargar } = preparar("es", (ruta) => caido && ruta.endsWith("contratacion-temporal-prueba"));
  await assert.rejects(cargar("contratacion-temporal-prueba"), /fallo de red/u);
  caido = false;
  const recuperado = await cargar("contratacion-temporal-prueba");
  assert.equal(recuperado.actual.titulo, "Texto del catálogo");
});
