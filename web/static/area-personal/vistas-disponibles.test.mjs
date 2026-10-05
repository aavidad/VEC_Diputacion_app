import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { leerVistasDisponibles, VISTAS_DISPONIBLES } from "./vistas-disponibles.js";

const catalogo = JSON.parse(await readFile(new URL("./vistas.json", import.meta.url), "utf8"));

test("las vistas sin servicio en el servidor no se ofrecen", () => {
  for (const vista of ["subsanaciones", "alegaciones", "mensajes", "certificados"]) {
    assert.equal(VISTAS_DISPONIBLES.has(vista), false, vista);
  }
  for (const vista of ["inicio", "llamamientos", "perfil", "preferencias", "ayuda"]) {
    assert.equal(VISTAS_DISPONIBLES.has(vista), true, vista);
  }
});

test("el catálogo cubre cada vista que registra la aplicación", async () => {
  const aplicacion = await readFile(new URL("./aplicacion.js", import.meta.url), "utf8");
  const bloque = aplicacion.slice(aplicacion.indexOf("const RUTAS = Object.freeze({"), aplicacion.indexOf("});", aplicacion.indexOf("const RUTAS")));
  const registradas = [...bloque.matchAll(/^ {2}([a-z]+): \[/gmu)].map(([, vista]) => vista).sort();
  assert.deepEqual(Object.keys(catalogo.vistas).sort(), registradas);
});

test("activar una vista es cambiar el catálogo, y lo que no nombra queda cerrado", () => {
  const activada = structuredClone(catalogo);
  activada.vistas.mensajes = true;
  assert.equal(leerVistasDisponibles(activada).has("mensajes"), true);
  const { mensajes: _omitida, ...sinMensajes } = catalogo.vistas;
  assert.equal(leerVistasDisponibles({ ...catalogo, vistas: sinMensajes }).has("mensajes"), false);
});

test("un catálogo mal formado detiene el arranque en lugar de abrir vistas", () => {
  for (const malo of [
    null, [], {}, { ...catalogo, version: "otra" }, { ...catalogo, extra: true },
    { ...catalogo, vistas: { ...catalogo.vistas, ayuda: "si" } },
    { ...catalogo, vistas: { ...catalogo.vistas, llamamientos: false } },
    { ...catalogo, vistas: { ...catalogo.vistas, inicio: false } },
  ]) assert.throws(() => leerVistasDisponibles(malo), TypeError);
});
