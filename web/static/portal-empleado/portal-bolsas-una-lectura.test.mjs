import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// Inicio pide el cuadro en paralelo con el catálogo. CT directo no consulta
// Bolsa. Cuando llega el catálogo no se relanza una lectura del mismo ciclo:
// hacerlo cancelaría la conexión y haría empezar de nuevo al servidor.
test("el catálogo no relanza la lectura del cuadro de bolsas del mismo ciclo", async () => {
  const fuente = await readFile(new URL("./portal.js", import.meta.url), "utf8");
  const cuerpo = fuente.slice(fuente.indexOf("function alCambiarModulos("), fuente.indexOf("function alCambiarModulos(") + 900);
  assert.match(cuerpo, /clave === "catalogo" && vistaNecesitaBolsa\(\)\s*&& !\(cicloLecturaBolsas === secuenciaFuente && \["cargando", "listo"\]\.includes\(estado\.datosBolsas\?\.carga\)\)/u);
  assert.match(cuerpo, /pedirCuadroBolsas\(\);/u);
  assert.doesNotMatch(cuerpo, /controladorBolsas\.cargarBolsas\(\)/u, "el catálogo pasa por pedirCuadroBolsas");
  assert.match(fuente, /function pedirCuadroBolsas\(\) \{\s*const pedido = \+\+generacionCuadroBolsas;\s*cicloLecturaBolsas = secuenciaFuente;/u);
  assert.match(fuente, /pedido === generacionCuadroBolsas && vistaNecesitaBolsa\(\)[\s\S]*?void controladorBolsas\.cargarBolsas\(\)/u);
  const arranque = fuente.slice(fuente.indexOf("async function cargarFuenteDatos("), fuente.indexOf("await coordinadorModulos.cargarInterno("));
  assert.match(arranque, /pedirCuadroBolsas\(\);/u, "la lectura inicial registra su ciclo");
});
