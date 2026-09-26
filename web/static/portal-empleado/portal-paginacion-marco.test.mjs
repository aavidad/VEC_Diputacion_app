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

test("la paginación larga conserva primera, vecinas, puntos y última sin dibujar 24 números", () => {
  const inicio = javascript.indexOf("function botonesPaginaMarco(calculo)");
  const fin = javascript.indexOf("function pintarPaginacionMarco", inicio);
  assert.ok(inicio >= 0 && fin > inicio);
  const fuente = `${javascript.slice(inicio, fin)}\nbotonesPaginaMarco({ pagina: 12, paginas: 24 });`;
  const html = runInNewContext(fuente, { Array, textoPortal: (clave, datos) => `${clave}:${datos?.pagina ?? ""}` });
  assert.deepEqual([...html.matchAll(/data-paginacion-marco-pagina="(\d+)"/gu)].map((m) => Number(m[1])), [1, 11, 12, 13, 24]);
  assert.equal((html.match(/paginacion-marco__puntos/gu) || []).length, 2);
  assert.match(javascript, /data-paginacion-marco-accion="primera"/u);
});

test("los textos visibles proceden del catálogo común", () => {
  for (const clave of ["paginacion_marco_etiqueta", "paginacion_marco_recuento", "paginacion_marco_primera", "paginacion_marco_anterior", "paginacion_marco_siguiente"]) {
    assert.match(catalogo, new RegExp(`${clave}:`));
    assert.match(javascript, new RegExp(`traducirPortal\\([\\"\\']${clave}[\\"\\']`));
  }
});

test("R10 bloquea el documento en escritorio y conserva móvil vertical", () => {
  assert.match(estilos, /@media \(min-width: 1024px\)[\s\S]*html,[\s\S]*body\.portal-empleado-app[\s\S]*overflow:\s*hidden/u);
  assert.match(estilos, /#espacio-trabajo[\s\S]*overflow:\s*auto/u);
  assert.match(componentes, /\.paginacion-marco[\s\S]*\.paginacion-marco__paginas/u);
  assert.match(componentes, /@media \(max-width: 1023\.98px\)/u);
});

function cargarPintado() {
  const inicio = javascript.indexOf("export function calcularPaginaMarco");
  const fin = javascript.indexOf(" function prepararTablaPaginable", inicio);
  assert.ok(inicio >= 0 && fin > inicio);
  const tablasPaginadas = new Map();
  const contexto = { Array, Math, Number, Object, tablasPaginadas, traducirPortal: (clave, datos) => `${clave}:${datos?.total ?? ""}`, textoPortal: (clave, datos) => `${clave}:${datos?.pagina ?? ""}` };
  runInNewContext(`${javascript.slice(inicio, fin).replace("export function", "function")}
    this.pintarPaginacionMarco = pintarPaginacionMarco; this.filasPaginablesMarco = filasPaginablesMarco; this.paginaInicialMarco = paginaInicialMarco;`, contexto);
  return contexto;
}

function tablaCandidatos(total, fichaTras) {
  const filas = [];
  for (let i = 0; i < total; i += 1) {
    filas.push({ clases: [], hidden: false });
    if (i === fichaTras) filas.push({ clases: ["fila-ficha-participacion"], hidden: false });
  }
  filas.forEach((fila, i) => Object.assign(fila, {
    nextElementSibling: filas[i + 1] || null,
    classList: { contains: (c) => fila.clases.includes(c) },
    hasAttribute: () => false,
    querySelector: () => null,
  }));
  return { isConnected: true, tBodies: [{ rows: filas }], filas };
}

test("la ficha de participación abierta no cuenta como candidato y sigue a su fila", () => {
  const ctx = cargarPintado();
  const tabla = tablaCandidatos(25, 8);
  const navegacion = { innerHTML: "" };
  ctx.tablasPaginadas.set(tabla, { navegacion, pagina: 1, tamano: 6 });
  assert.equal(ctx.filasPaginablesMarco(tabla).length, 25);
  const ficha = tabla.filas.find((f) => f.clases.length);
  const candidato = tabla.filas[tabla.filas.indexOf(ficha) - 1];
  assert.equal(ctx.paginaInicialMarco(ctx.filasPaginablesMarco(tabla), 6), 2);
  ctx.pintarPaginacionMarco(tabla, 1);
  assert.match(navegacion.innerHTML, /paginacion_marco_recuento:25/u);
  assert.equal(candidato.hidden, true);
  assert.equal(ficha.hidden, true);
  ctx.pintarPaginacionMarco(tabla, 2);
  assert.equal(candidato.hidden, false);
  assert.equal(ficha.hidden, false);
  assert.equal(tabla.filas.filter((f) => !f.hidden && !f.clases.length).length, 6);
  assert.equal(ctx.paginaInicialMarco(ctx.filasPaginablesMarco(tablaCandidatos(25, -1)), 6), 1);
});
