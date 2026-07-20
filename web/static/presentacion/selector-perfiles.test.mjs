import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  obtenerPerfilesPresentacion,
  obtenerRutaPerfilPresentacion,
} from "./selector-perfiles.js";

test("el selector declara exactamente los cuatro perfiles y sus superficies", () => {
  const perfiles = obtenerPerfilesPresentacion();
  assert.deepEqual(perfiles.map(({ clave }) => clave), [
    "usuario_externo", "funcionario", "tecnico", "administrador",
  ]);
  assert.match(obtenerRutaPerfilPresentacion("usuario_externo"), /^\/area-personal\/\?presentacion=rrhh/);
  assert.equal(obtenerRutaPerfilPresentacion("funcionario"), "/portal-empleado/?presentacion=rrhh&perfil=funcionario#portal");
  assert.equal(obtenerRutaPerfilPresentacion("tecnico"), "/portal-empleado/?presentacion=rrhh&perfil=tecnico#portal");
  assert.equal(obtenerRutaPerfilPresentacion("administrador"), "/portal-empleado/?presentacion=rrhh&perfil=administrador#portal");
  assert.ok(perfiles.every(Object.isFrozen));
  assert.throws(() => obtenerRutaPerfilPresentacion("desconocido"), /no reconocido/);
});

test("el selector es efímero, accesible por teclado y no concede autoridad", async () => {
  const codigo = await readFile(new URL("selector-perfiles.js", import.meta.url), "utf8");
  const estilos = await readFile(new URL("selector-perfiles.css", import.meta.url), "utf8");
  assert.doesNotMatch(codigo, /localStorage|sessionStorage|document\.cookie|fetch\s*\(|XMLHttpRequest|WebSocket/u);
  assert.match(codigo, /aria-haspopup", "dialog"/u);
  assert.match(codigo, /setAttribute\("role", "button"\)/u);
  assert.match(codigo, /aria-expanded/u);
  assert.match(codigo, /aria-current/u);
  assert.match(codigo, /"ArrowDown", "ArrowUp", "Home", "End"/u);
  assert.match(codigo, /evento\.key === "Escape"/u);
  assert.match(codigo, /no cambia permisos reales/u);
  assert.match(codigo, /documento\.head\.append\(enlace\)/u);
  assert.match(estilos, /@media \(max-width: 680px\)[\s\S]*position:\s*absolute/u);
  assert.doesNotMatch(estilos, /position:\s*fixed/u);
});
