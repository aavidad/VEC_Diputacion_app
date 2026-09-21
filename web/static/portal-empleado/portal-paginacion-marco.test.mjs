import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

const directorio = new URL("./", import.meta.url);
const [javascript, estilos, componentes, catalogo] = await Promise.all([
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-componentes.css", directorio), "utf8"),
  readFile(new URL("portal-i18n.js", directorio), "utf8"),
]);

function cargarCalculo() {
  const inicio = javascript.indexOf("export function calcularPaginaMarco");
  const fin = javascript.indexOf(" function navegadorRemotoDeTabla", inicio);
  assert.ok(inicio >= 0 && fin > inicio);
  const fuente = `${javascript.slice(inicio, fin).replace("export function", "function")}\ncalcularPaginaMarco(13, 2, 6);`;
  return runInNewContext(fuente, { Number, Math, Object });
}

test("la paginación de marco calcula límites estables con tamaño fijo", () => {
  assert.deepEqual({ ...cargarCalculo() }, {
    total: 13, tamano: 6, paginas: 3, pagina: 2, inicio: 7, fin: 12,
  });
});

test("la paginación distingue tablas locales de cursores remotos", () => {
  assert.match(javascript, /TAMANO_PAGINA_MARCO = 6/u);
  assert.match(javascript, /navegadorRemotoDeTabla/u);
  assert.match(javascript, /ct-exp-paginacion[\s\S]*paginacion-bolsa/u);
  assert.match(javascript, /filas\.length <= TAMANO_PAGINA_MARCO/u);
  assert.match(javascript, /new MutationObserver\(actualizarPaginacionesMarco\)/u);
});

test("los textos visibles proceden del catálogo común", () => {
  for (const clave of ["paginacion_marco_etiqueta", "paginacion_marco_recuento", "paginacion_marco_anterior", "paginacion_marco_siguiente"]) {
    assert.match(catalogo, new RegExp(`${clave}:`));
    assert.match(javascript, new RegExp(`traducirPortal\\(\\"${clave}\\"`));
  }
});

test("R10 bloquea el documento en escritorio y conserva móvil vertical", () => {
  assert.match(estilos, /@media \(min-width: 1024px\)[\s\S]*html,[\s\S]*body\.portal-empleado-app[\s\S]*overflow:\s*hidden/u);
  assert.match(estilos, /#espacio-trabajo[\s\S]*overflow:\s*auto/u);
  assert.match(componentes, /\.paginacion-marco[\s\S]*\.paginacion-marco__paginas/u);
  assert.match(componentes, /@media \(max-width: 1023\.98px\)/u);
});
