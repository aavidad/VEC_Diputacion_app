import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { IDIOMA_DATOS_REGLAS, textoPresentacionRegla } from "./i18n.js";

const catalogo = async (idioma) => JSON.parse(await readFile(
  new URL(`../../textos/${idioma}/reglas.json`, import.meta.url), "utf8",
)).presentacion;

function explicaciones(seccion, ruta = [], salida = []) {
  for (const [clave, valor] of Object.entries(seccion)) {
    if (typeof valor.original === "string") {
      salida.push({ clave: ruta.join("."), campo: clave, entrada: valor });
    } else explicaciones(valor, [...ruta, clave], salida);
  }
  return salida;
}

test("las explicaciones ES/EN conservan su fuente y no modifican las reglas", async () => {
  const es = await catalogo("es");
  for (const idioma of ["es", "en"]) {
    const presentacion = await catalogo(idioma);
    assert.deepEqual(Object.keys(presentacion), Object.keys(es));
    for (const [modulo, reglas] of Object.entries(presentacion)) {
      assert.deepEqual(Object.keys(reglas), Object.keys(es[modulo]));
      const base = explicaciones(es[modulo]);
      const propias = explicaciones(reglas);
      assert.deepEqual(propias.map(({ clave, campo }) => [clave, campo]), base.map(({ clave, campo }) => [clave, campo]));
      for (const { clave, campo, entrada } of propias) {
        assert.equal(entrada.original, base.find((e) => e.clave === clave && e.campo === campo).entrada.original);
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
});

test("el módulo, la clave y el campo ajenos conservan el texto y el idioma de los datos", async () => {
  const presentacion = await catalogo("en");
  const clave = "c20.cancelacion_expediente";
  const original = presentacion.contratacion_temporal.c20.cancelacion_expediente.descripcion.original;
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
    { contratacion_temporal: { c20: { cancelacion_expediente: campos } } },
    Object.create({ contratacion_temporal: { c20: { cancelacion_expediente: campos } } }),
  ]) {
    assert.deepEqual(textoPresentacionRegla("contratacion_temporal", regla, "descripcion", { presentacion }),
      { texto: regla.descripcion, idioma: IDIOMA_DATOS_REGLAS });
  }
});

test("los segmentos reservados y heredados no sustituyen la fuente", () => {
  const descripcion = "Texto de la fuente";
  const entrada = { descripcion: { original: descripcion, texto: "Presentación" } };
  for (const segmento of ["__proto__", "constructor", "prototype"]) {
    const presentacion = { contratacion_temporal: { c20: { [segmento]: entrada } } };
    assert.deepEqual(textoPresentacionRegla("contratacion_temporal",
      { clave: `c20.${segmento}`, descripcion }, "descripcion", { presentacion }),
    { texto: descripcion, idioma: IDIOMA_DATOS_REGLAS });
  }
  const presentacion = { contratacion_temporal: { c20: Object.create({ cancelacion_expediente: entrada }) } };
  assert.deepEqual(textoPresentacionRegla("contratacion_temporal",
    { clave: "c20.cancelacion_expediente", descripcion }, "descripcion", { presentacion }),
  { texto: descripcion, idioma: IDIOMA_DATOS_REGLAS });
});
