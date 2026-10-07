import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { IDIOMA_DATOS_REGLAS, textoPresentacionRegla } from "./i18n.js";
import { cargarTextos } from "../../comun/textos.js";
import { INDICE_IDIOMAS } from "../../comun/idioma.js";

test("el idioma de los datos se fija tras preparar el índice en cada navegación", () => {
  for (const { codigo } of INDICE_IDIOMAS.idiomas) {
    const salida = execFileSync(process.execPath, ["--input-type=module", "-e", `
      globalThis.location = { href: ${JSON.stringify(`https://vec.example/portal-empleado/reglas/?lang=${codigo}`)} };
      const modulo = await import(${JSON.stringify(new URL("./i18n.js", import.meta.url).href)});
      const { INDICE_IDIOMAS } = await import(${JSON.stringify(new URL("../../comun/idioma.js", import.meta.url).href)});
      console.log(JSON.stringify({ interfaz: modulo.IDIOMA_REGLAS, datos: modulo.IDIOMA_DATOS_REGLAS,
        defecto: INDICE_IDIOMAS.porDefecto, error: modulo.ERROR_TEXTOS_REGLAS?.message ?? null }));
    `], { encoding: "utf8" });
    const resultado = JSON.parse(salida);
    assert.deepEqual(resultado, { interfaz: codigo, datos: INDICE_IDIOMAS.porDefecto,
      defecto: INDICE_IDIOMAS.porDefecto, error: null });
  }
});

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

test("el idioma efectivo y las hojas de respaldo proceden del cargador común", async () => {
  const es = JSON.parse(await readFile(new URL("../../textos/es/reglas.json", import.meta.url), "utf8"));
  const en = JSON.parse(await readFile(new URL("../../textos/en/reglas.json", import.meta.url), "utf8"));
  const descripcion = es.presentacion.contratacion_temporal.c20.cancelacion_expediente.descripcion;
  const regla = { clave: "c20.cancelacion_expediente", descripcion: descripcion.original };
  for (const escenario of ["no_disponible", "hoja_ausente", "traducido"]) {
    const propio = structuredClone(en);
    if (escenario === "hoja_ausente") delete propio.presentacion.contratacion_temporal.c20.cancelacion_expediente.descripcion.texto;
    const textos = await cargarTextos("reglas", { idioma: "en", porDefecto: "es", avisar: () => {},
      leer: async (url) => {
        if (url.pathname.endsWith("/es/reglas.json")) return es;
        if (escenario === "no_disponible") throw new Error("404");
        return propio;
      },
    });
    const presentado = textoPresentacionRegla("contratacion_temporal", regla, "descripcion", {
      presentacion: textos.mensajes.presentacion, idioma: textos.idioma, faltantes: textos.faltantes,
    });
    assert.deepEqual(presentado, escenario === "traducido"
      ? { texto: en.presentacion.contratacion_temporal.c20.cancelacion_expediente.descripcion.texto, idioma: "en" }
      : escenario === "hoja_ausente"
        ? { texto: descripcion.original, idioma: "es" }
        : { texto: descripcion.texto, idioma: "es" });
  }
});
