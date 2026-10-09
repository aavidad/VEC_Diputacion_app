import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { cargarVistasDisponibles, leerVistasDisponibles } from "./vistas-disponibles.js";

const textoCatalogo = await readFile(new URL("./vistas.json", import.meta.url), "utf8");
const catalogo = JSON.parse(textoCatalogo);
const VISTAS_DISPONIBLES = leerVistasDisponibles(catalogo);

test("las vistas sin servicio en el servidor no se ofrecen", () => {
  for (const vista of ["subsanaciones", "alegaciones", "mensajes", "certificados"]) {
    assert.equal(VISTAS_DISPONIBLES.has(vista), false, vista);
  }
  for (const vista of ["inicio", "llamamientos", "perfil", "preferencias", "ayuda"]) {
    assert.equal(VISTAS_DISPONIBLES.has(vista), true, vista);
  }
});

test("convocatorias, méritos y expedientes se retiran del catálogo", () => {
  for (const vista of ["convocatorias", "convocatoria", "meritos", "seguimiento"]) {
    assert.equal(Object.hasOwn(catalogo.vistas, vista), false, vista);
    assert.equal(VISTAS_DISPONIBLES.has(vista), false, vista);
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

test("el arranque lee el catálogo del mismo origen sin caché y falla cerrado", async () => {
  let peticion;
  const vistas = await cargarVistasDisponibles({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return { status: 200, text: async () => textoCatalogo };
  } });
  assert.equal(peticion.ruta, "/area-personal/vistas.json");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.credentials, "same-origin");
  assert.equal(vistas.has("mensajes"), false);
  for (const fetchImpl of [
    async () => ({ status: 404, text: async () => "" }),
    async () => ({ status: 200, text: async () => "no es json" }),
    async () => ({ status: 200, text: async () => " ".repeat(9000) }),
    async () => { throw new Error("sin red"); },
  ]) await assert.rejects(() => cargarVistasDisponibles({ fetchImpl }));
});

test("el menú ya trae ocultas las vistas desactivadas antes de leer el catálogo", async () => {
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  for (const [vista, activa] of Object.entries(catalogo.vistas)) {
    const enlace = html.match(new RegExp(`<a href="\\?vista=${vista}" data-ruta="${vista}"[^>]*>`, "u"))?.[0];
    if (enlace) assert.equal(/\shidden>/u.test(enlace), !activa, enlace);
  }
});
