import assert from "node:assert/strict";
import test from "node:test";
import { iniciarI18nAreaPersonal, rutaCatalogoAreaPersonal, traducir } from "./i18n.js";

test("carga inglés solo cuando se elige en la interfaz (lang=en)", async () => {
  const documento = { documentElement: { lang: "es" }, querySelectorAll: () => [] };
  const llamadas = [];
  const idioma = await iniciarI18nAreaPersonal(documento, async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return { ok: true, json: async () => ({ "areaPersonal.rutas.inicio": "Home and deadlines" }) };
  }, ["es-ES"], { href: "https://vec.example/area-personal/?lang=en" });
  assert.equal(idioma, "en");
  assert.equal(documento.documentElement.lang, "en");
  assert.equal(traducir("areaPersonal.rutas.inicio"), "Home and deadlines");
  assert.deepEqual(llamadas, [["/area-personal/locales/en.json", { credentials: "omit" }]]);
});

test("un navegador en inglés sin elección explícita sigue en castellano", () => {
  assert.equal(rutaCatalogoAreaPersonal(["en-GB"], { href: "https://vec.example/area-personal/" }), "/area-personal/locales/es.json");
});

test("la URL explícita prevalece y una preferencia extraña no construye rutas", () => {
  assert.equal(rutaCatalogoAreaPersonal(["en-GB"], { href: "https://vec.example/area-personal/?lang=es" }), "/area-personal/locales/es.json");
  assert.equal(rutaCatalogoAreaPersonal(["../../privado"], { href: "https://vec.example/area-personal/?lang=../../privado" }), "/area-personal/locales/es.json");
});
