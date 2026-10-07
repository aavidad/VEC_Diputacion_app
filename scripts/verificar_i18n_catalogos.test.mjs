import assert from "node:assert/strict";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import { validarPar, validarRespuesta } from "./verificar_i18n_catalogos.mjs";
import { cargarCatalogosContratacionEnIdioma } from "../web/static/portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js";

const modulo = "contratacion-temporal-llamamiento";
const leer = async (idioma) => JSON.parse(await readFile(new URL(
  `../web/static/textos/${idioma}/${modulo}.json`, import.meta.url), "utf8"));

test("una clave inglesa eliminada falla contra la respuesta real y el catálogo español", async () => {
  const temporal = await mkdtemp(join(tmpdir(), "vec-i18n-guarda-"));
  try {
    const [es, en, respuesta] = await Promise.all([
      leer("es"), leer("en"), cargarCatalogosContratacionEnIdioma(modulo, "en"),
    ]);
    const clave = Object.keys(en.general)[0];
    delete en.general[clave];
    const copia = join(temporal, `${modulo}.json`);
    await writeFile(copia, JSON.stringify(en));
    const alterado = JSON.parse(await readFile(copia, "utf8"));
    assert.match(validarPar(modulo, es.general, alterado.general).join(" "), /claves distintas/u);
    assert.match(validarRespuesta(modulo, "en", respuesta, alterado.general).join(" "), /no coincide/u);
  } finally {
    await rm(temporal, { recursive: true, force: true });
  }
});

test("un locale inválido y un respaldo disfrazado de inglés fallan la guarda", async () => {
  const [es, en, invalido] = await Promise.all([
    leer("es"), leer("en"), cargarCatalogosContratacionEnIdioma(modulo, "zz"),
  ]);
  assert.match(validarRespuesta(modulo, "zz", invalido, en.general).join(" "), /idioma solicitado/u);
  const falso = { idioma: "en", incidenciaCatalogo: null, incidenciaIndice: null, actual: es.general };
  assert.match(validarRespuesta(modulo, "en", falso, en.general).join(" "), /no coincide/u);
});
