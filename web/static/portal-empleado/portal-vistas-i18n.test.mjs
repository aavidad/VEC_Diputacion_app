import assert from "node:assert/strict";
import test from "node:test";
import { TIPOS_LISTA_EN, TIPOS_LISTA_ES, traducirTipoLista } from "./portal-vistas-i18n.js";

test("los tipos conocidos tienen traducción paritaria y los desconocidos conservan el código", () => {
  assert.deepEqual(Object.keys(TIPOS_LISTA_EN).sort(), Object.keys(TIPOS_LISTA_ES).sort());
  assert.equal(traducirTipoLista("ordinaria", "es"), "Ordinaria");
  assert.equal(traducirTipoLista("ordinaria", "en"), "Ordinary");
  assert.equal(traducirTipoLista("rotatoria", "es"), "Rotatoria");
  assert.equal(traducirTipoLista("rotatoria", "en"), "Rotating");
  assert.equal(traducirTipoLista("tipo_sin_catalogo", "en"), "tipo_sin_catalogo");
  assert.equal(traducirTipoLista("<img>", "en"), "<img>");
});
