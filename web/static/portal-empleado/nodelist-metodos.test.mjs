import assert from "node:assert/strict";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import test from "node:test";

// En el navegador querySelectorAll devuelve un NodeList y children, options,
// elements y getElementsBy* devuelven colecciones HTML: ninguno tiene find,
// filter, map ni concat (las colecciones tampoco forEach). El DOM falso de las
// pruebas devuelve arrays y lo oculta, así que se vigila el código servido.
const RAIZ = new URL("..", import.meta.url).pathname;

// Métodos de Array que no existen en NodeList.
const METODOS_ARRAY = ["at", "concat", "every", "fill", "filter", "find", "findIndex", "findLast", "findLastIndex",
  "flat", "flatMap", "includes", "indexOf", "join", "lastIndexOf", "map", "pop", "push", "reduce", "reduceRight",
  "reverse", "shift", "slice", "some", "sort", "splice", "toReversed", "toSorted", "toSpliced", "unshift", "with"];
// Las colecciones HTML tampoco tienen forEach.
const METODOS_COLECCION = [...METODOS_ARRAY, "forEach"];
const LLAMADAS = /\bquerySelectorAll\s*\(|\bgetElementsBy[A-Za-z]+\s*\(/gu;
const PROPIEDADES = /\??\.\s*(children|options|elements|childNodes)\b(?!\s*\()/gu;
const DECLARACION = /\b(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*/gu;

/** Salta una cadena simple ('…', "…" o `…` sin anidar) y devuelve el índice posterior. */
function saltarCadena(texto, inicio) {
  const comilla = texto[inicio];
  for (let i = inicio + 1; i < texto.length; i += 1) {
    if (texto[i] === "\\") i += 1;
    else if (texto[i] === comilla) return i + 1;
    else if (comilla !== "`" && texto[i] === "\n") return i;
  }
  return texto.length;
}

/** Índice del paréntesis que cierra el abierto en `abre`, contando niveles e ignorando cadenas. */
export function cierreParentesis(texto, abre) {
  let nivel = 0;
  for (let i = abre; i < texto.length;) {
    const caracter = texto[i];
    if (caracter === "\"" || caracter === "'" || caracter === "`") { i = saltarCadena(texto, i); continue; }
    if (caracter === "(") nivel += 1;
    else if (caracter === ")" && (nivel -= 1) === 0) return i;
    i += 1;
  }
  return -1;
}

const metodoTrasIndice = (texto, indice, metodos) => {
  const resto = texto.slice(indice).match(/^\s*\??\.\s*([A-Za-z_$][\w$]*)\s*\(/u);
  return resto && metodos.includes(resto[1]) ? resto[1] : null;
};
const lineaDe = (texto, indice) => texto.slice(0, indice).split("\n").length;

/** Devuelve los números de línea en que se llama a un método de Array sobre un NodeList o colección. */
export function usosIndebidos(texto) {
  const lineas = new Set();
  // Fin de cada expresión que produce una colección viva, con los métodos que no admite.
  const colecciones = [];
  for (const llamada of texto.matchAll(LLAMADAS)) {
    const cierre = cierreParentesis(texto, llamada.index + llamada[0].length - 1);
    if (cierre < 0) continue;
    const metodos = llamada[0].startsWith("querySelectorAll") ? METODOS_ARRAY : METODOS_COLECCION;
    colecciones.push({ inicio: llamada.index, fin: cierre + 1, metodos });
  }
  for (const propiedad of texto.matchAll(PROPIEDADES))
    colecciones.push({ inicio: propiedad.index, fin: propiedad.index + propiedad[0].length, metodos: METODOS_COLECCION });
  for (const { inicio, fin, metodos } of colecciones)
    if (metodoTrasIndice(texto, fin, metodos)) lineas.add(lineaDe(texto, inicio));
  // Variables inicializadas directamente con la colección y usadas después como array.
  for (const declaracion of texto.matchAll(DECLARACION)) {
    const desde = declaracion.index + declaracion[0].length;
    const coleccion = colecciones.find(({ inicio }) => inicio >= desde && inicio - desde < 400 &&
      /^[\w$]+(?:\??\.[\w$]+|\([^()]*\))*\??\.?$/u.test(texto.slice(desde, inicio)));
    // La colección debe ser el valor completo: nada encadenado detrás salvo «|| []» o «?? []».
    if (!coleccion || !/^[ \t]*(?:(?:\|\||\?\?)[ \t]*\[[ \t]*\][ \t]*)?(?:[;,]|\n(?!\s*\??\.))/u
      .test(texto.slice(coleccion.fin, coleccion.fin + 200))) continue;
    const nombre = declaracion[1];
    const siguiente = new RegExp(`\\b(?:const|let|var)\\s+${nombre.replace(/\$/gu, "\\$")}\\b|(?<![\\w$.])${nombre.replace(/\$/gu, "\\$")}\\s*=(?!=)`, "gu");
    siguiente.lastIndex = coleccion.fin;
    const hasta = siguiente.exec(texto)?.index ?? texto.length;
    const uso = new RegExp(`(?<![\\w$.])${nombre.replace(/\$/gu, "\\$")}\\b`, "gu");
    uso.lastIndex = coleccion.fin;
    for (let encontrado = uso.exec(texto); encontrado && encontrado.index < hasta; encontrado = uso.exec(texto))
      if (metodoTrasIndice(texto, encontrado.index + encontrado[0].length, coleccion.metodos))
        lineas.add(lineaDe(texto, encontrado.index));
  }
  return [...lineas].sort((a, b) => a - b);
}

function ficheros(dir) {
  return readdirSync(dir).flatMap((nombre) => {
    const ruta = join(dir, nombre);
    if (statSync(ruta).isDirectory()) return nombre === "vendor" ? [] : ficheros(ruta);
    return /\.(m?js)$/u.test(nombre) && !/\.test\.m?js$|test-helper/u.test(nombre) ? [ruta] : [];
  });
}

test("la guarda detecta los usos indebidos en varias líneas, en colecciones HTML y a través de variables", () => {
  const fuente = [
    "const a = raiz.querySelectorAll(\"select\").filter((s) =>",
    "  s.name === \")\");",
    "const b = form.querySelectorAll(",
    "  \"[data-x='(']\",",
    ")",
    "  .map((x) => x);",
    "raiz.children.forEach((hijo) => hijo);",
    "selector.options?.find((o) => o.selected);",
    "form.elements.map((e) => e.name);",
    "documento.getElementsByTagName(\"a\").filter(Boolean);",
    "const filas = form.querySelectorAll(\"tr\") || [];",
    "const total = filas.length;",
    "const rutas = filas.map((fila) => fila);",
    "const hijos = lista.children;",
    "hijos.some(Boolean);",
    // Usos correctos: conversión explícita, métodos propios de NodeList y reasignación.
    "Array.from(raiz.querySelectorAll(\"a\")).map(String);",
    "[...raiz.querySelectorAll(\"a\")].filter(Boolean);",
    "raiz.querySelectorAll(\"a\").forEach(String);",
    "let nodos = raiz.querySelectorAll(\"a\");",
    "nodos = Array.from(nodos);",
    "nodos.map(String);",
    "const datos = { children: [] }; datos.children.length;",
  ].join("\n");
  assert.deepEqual(usosIndebidos(fuente), [1, 3, 7, 8, 9, 10, 13, 15]);
});

test("ningún fichero servido usa métodos de array sobre un NodeList o una colección HTML", () => {
  const infractores = ficheros(RAIZ).flatMap((ruta) =>
    usosIndebidos(readFileSync(ruta, "utf8")).map((linea) => `${ruta.slice(RAIZ.length)}:${linea}`));
  assert.deepEqual(infractores, []);
});
