import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// Al arrancar, el portal pide el cuadro de bolsas en paralelo con el catálogo
// de módulos. Cuando llega el catálogo no debe relanzarlo si la lectura del
// mismo ciclo sigue en curso: relanzarla cancelaba la petición (y su conexión)
// y el servidor volvía a empezar, retrasando Bolsa y al resto del portal.
test("el catálogo no relanza la lectura del cuadro de bolsas del mismo ciclo", async () => {
  const fuente = await readFile(new URL("./portal.js", import.meta.url), "utf8");
  const cuerpo = fuente.slice(fuente.indexOf("function alCambiarModulos("), fuente.indexOf("function alCambiarModulos(") + 900);
  assert.match(cuerpo, /clave === "catalogo"\s*&& !\(cicloLecturaBolsas === secuenciaFuente && \["cargando", "listo"\]\.includes\(estado\.datosBolsas\?\.carga\)\)/u);
  assert.match(cuerpo, /pedirCuadroBolsas\(\);/u);
  assert.doesNotMatch(cuerpo, /controladorBolsas\.cargarBolsas\(\)/u, "el catálogo pasa por pedirCuadroBolsas");
  assert.match(fuente, /function pedirCuadroBolsas\(\) \{\s*cicloLecturaBolsas = secuenciaFuente;\s*void controladorBolsas\.cargarBolsas\(\);\s*\}/u);
  const arranque = fuente.slice(fuente.indexOf("async function cargarFuenteDatos("), fuente.indexOf("await coordinadorModulos.cargarInterno("));
  assert.match(arranque, /pedirCuadroBolsas\(\);/u, "la lectura inicial registra su ciclo");
});
