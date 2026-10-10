import assert from "node:assert/strict";
import test from "node:test";
import { datosMinimosMiBolsa } from "./aplicacion.js";
import { validarRespuestaMiBolsa } from "./contrato.js";

// Datos sintéticos: el nombre llega solo en la consulta propia de la titular.
function respuestaPrueba(extra = {}) {
  return { data: { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-10-09T10:00:00.000Z", participaciones: [
    { bolsa: "bolsa:prueba:01", categoria: "Auxiliar", version: 1, orden_inicial: 3, total_instantanea: 20, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00.000Z", vigente_hasta: null },
  ], ...extra } };
}

test("la cabecera pinta el nombre y las iniciales que envía el servidor", () => {
  const consulta = validarRespuestaMiBolsa(respuestaPrueba({ nombre_visible: "Lucía Moreno Castillo", iniciales: "LM" }));
  const datos = datosMinimosMiBolsa(consulta);
  assert.equal(datos.sesion.nombre_visible, "Lucía Moreno Castillo");
  assert.equal(datos.sesion.iniciales, "LM");
  assert.equal(datos.perfil.nombre_visible, "Lucía Moreno Castillo");
  assert.equal(datos.sesion.persona_ref, null);
});

test("sin nombre la cabecera sigue con la raya y sin rótulo", () => {
  const datos = datosMinimosMiBolsa(validarRespuestaMiBolsa(respuestaPrueba()));
  assert.equal(datos.sesion.nombre_visible, "");
  assert.equal(datos.sesion.iniciales, "—");
});

test("el contrato rechaza un nombre vacío, demasiado largo o sin iniciales", () => {
  for (const extra of [{ nombre_visible: "", iniciales: "LM" }, { nombre_visible: "x".repeat(201), iniciales: "XX" }, { nombre_visible: "Lucía" }, { iniciales: "LM" }]) {
    assert.throws(() => validarRespuestaMiBolsa(respuestaPrueba(extra)), /mi-bolsa\.(nombre_visible|iniciales)/u);
  }
});

test("la etiqueta del menú de identidad nombra a la titular en ambos idiomas", async () => {
  const { readFile } = await import("node:fs/promises");
  for (const idioma of ["es", "en"]) {
    const catalogo = JSON.parse(await readFile(new URL(`../textos/${idioma}/preferencias.json`, import.meta.url), "utf8"));
    const textos = catalogo.areaPersonal.preferencias;
    assert.equal(typeof textos.menuIdentidad, "string");
    assert.match(textos.menuIdentidadNombre, /\{nombre\}/u);
  }
  const codigo = await readFile(new URL("./aplicacion.js", import.meta.url), "utf8");
  assert.match(codigo, /botonSesion\.setAttribute\("aria-label", nombre \? traducir\("areaPersonal\.preferencias\.menuIdentidadNombre", \{ nombre \}\)/u);
});
