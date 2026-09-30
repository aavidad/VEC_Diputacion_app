import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { IDIOMA_DATOS_REGLAS, textoPresentacionRegla } from "./i18n.js";

const catalogo = async (idioma) => JSON.parse(await readFile(
  new URL(`../../textos/${idioma}/reglas.json`, import.meta.url), "utf8",
)).presentacion;

test("las explicaciones ES/EN conservan su fuente y no modifican las reglas", async () => {
  const es = await catalogo("es");
  for (const idioma of ["es", "en"]) {
    const presentacion = await catalogo(idioma);
    assert.deepEqual(Object.keys(presentacion), Object.keys(es));
    for (const [modulo, reglas] of Object.entries(presentacion)) {
      assert.deepEqual(Object.keys(reglas), Object.keys(es[modulo]));
      for (const [clave, campos] of Object.entries(reglas)) {
        assert.deepEqual(Object.keys(campos), Object.keys(es[modulo][clave]));
        for (const [campo, entrada] of Object.entries(campos)) {
          assert.equal(entrada.original, es[modulo][clave][campo].original);
          assert.ok(entrada.texto);
          const regla = Object.freeze({ clave, [campo]: entrada.original, version: 1 });
          assert.deepEqual(textoPresentacionRegla(modulo, regla, campo, { presentacion, idioma }),
            { texto: entrada.texto, idioma });
          assert.equal(regla[campo], entrada.original);
          const modificada = { ...regla, [campo]: `${entrada.original} actualizado` };
          assert.deepEqual(textoPresentacionRegla(modulo, modificada, campo, { presentacion, idioma }),
            { texto: modificada[campo], idioma: IDIOMA_DATOS_REGLAS });
        }
      }
    }
  }
});

test("el módulo, la clave y el campo ajenos conservan el texto y el idioma de los datos", async () => {
  const presentacion = await catalogo("en");
  const clave = "c20.cancelacion_expediente";
  const original = presentacion.contratacion_temporal[clave].descripcion.original;
  for (const [modulo, regla, campo] of [
    ["bolsa", { clave, descripcion: original }, "descripcion"],
    ["contratacion_temporal", { clave: "c20.otra", descripcion: original }, "descripcion"],
    ["contratacion_temporal", { clave, duda: original }, "duda"],
  ]) {
    assert.deepEqual(textoPresentacionRegla(modulo, regla, campo, { presentacion }),
      { texto: original, idioma: IDIOMA_DATOS_REGLAS });
  }
});

test("una presentación vacía o heredada no sustituye la fuente", () => {
  const regla = { clave: "c20.cancelacion_expediente", descripcion: "Texto de la fuente" };
  const campos = { descripcion: { original: regla.descripcion, texto: "" } };
  for (const presentacion of [null,
    { contratacion_temporal: { [regla.clave]: campos } },
    Object.create({ contratacion_temporal: { [regla.clave]: campos } }),
  ]) {
    assert.deepEqual(textoPresentacionRegla("contratacion_temporal", regla, "descripcion", { presentacion }),
      { texto: regla.descripcion, idioma: IDIOMA_DATOS_REGLAS });
  }
});
