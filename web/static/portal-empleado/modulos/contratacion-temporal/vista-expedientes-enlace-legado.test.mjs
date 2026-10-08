import assert from "node:assert/strict";
import test from "node:test";
import { insertarAvisoEnlaceLegado } from "./vista-expedientes.js";

test("el aviso del enlace antiguo aparece sobre el cuadro como texto recuperable", () => {
  const crearElemento = (nombre) => ({ nombre, atributos: {}, hijos: [], textContent: "",
    setAttribute(clave, valor) { this.atributos[clave] = valor; },
    append(hijo) { this.hijos.push(hijo); } });
  const raiz = { ownerDocument: { createElement: crearElemento }, hijos: [],
    prepend(nodo) { this.hijos.unshift(nodo); } };
  const mensaje = "Este enlace ya no abre la ficha. Busque la petición en la lista y ábrala de nuevo.";
  insertarAvisoEnlaceLegado(raiz, mensaje);
  assert.equal(raiz.hijos[0].atributos.class, "panel panel-separado");
  assert.equal(raiz.hijos[0].atributos.role, "status");
  assert.equal(raiz.hijos[0].hijos[0].hijos[0].textContent, mensaje);
  assert.equal(raiz.hijos[0].hijos[0].hijos[0].hijos.length, 0);
  insertarAvisoEnlaceLegado(raiz, "");
  assert.equal(raiz.hijos.length, 1);
});
