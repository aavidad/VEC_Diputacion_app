import assert from "node:assert/strict";
import test from "node:test";
import { urlCatalogo } from "../comun/textos.js";
import { idiomaAreaPersonal, iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

test("carga inglés por preferencia del navegador sin usar el lang español de respaldo", async () => {
  const documento = { documentElement: { lang: "es" }, querySelectorAll: () => [] };
  const leer = lectorCatalogos();
  const idioma = await iniciarI18nAreaPersonal(documento, {
    leer, preferidos: ["en-GB"], ubicacion: { href: "https://vec.example/area-personal/" },
  });
  assert.equal(idioma, "en");
  assert.equal(documento.documentElement.lang, "en");
  assert.equal(traducir("areaPersonal.rutas.inicio"), (await catalogoPlano("en"))["areaPersonal.rutas.inicio"]);
  assert.ok(leer.pedidas.includes("en/area-personal.json"));
  assert.ok(leer.pedidas.includes("en/preferencias.json"));
  assert.deepEqual(leer.pedidas.filter((ruta) => ruta.startsWith("es/")), []);
});

test("el catálogo español solo se consulta al elegirlo", async () => {
  const documento = { documentElement: {}, querySelectorAll: () => [] };
  const leer = lectorCatalogos();
  await iniciarI18nAreaPersonal(documento, { leer, ubicacion: { href: "https://vec.example/area-personal/?lang=en" } });
  assert.deepEqual(leer.pedidas.filter((ruta) => ruta.startsWith("es/")), []);
  await iniciarI18nAreaPersonal(documento, { leer, ubicacion: { href: "https://vec.example/area-personal/?lang=es" } });
  assert.deepEqual(leer.pedidas.filter((ruta) => ruta.startsWith("es/")).sort(),
    ["es/area-personal.json", "es/preferencias.json"]);
  assert.equal(documento.documentElement.lang, "es");
});

test("un fallo del catálogo elegido recupera todo el idioma por defecto", async () => {
  const documento = { documentElement: {}, querySelectorAll: () => [] };
  const local = lectorCatalogos();
  const pedidas = [];
  const leer = async (url) => {
    const ruta = url.pathname.split("/textos/").at(-1);
    pedidas.push(ruta);
    if (ruta === "en/area-personal.json") throw new Error("lectura interrumpida");
    return local(url);
  };
  const idioma = await iniciarI18nAreaPersonal(documento, {
    leer, ubicacion: { href: "https://vec.example/area-personal/?lang=en" },
  });
  assert.equal(idioma, "es");
  assert.equal(documento.documentElement.lang, "es");
  assert.ok(pedidas.includes("es/area-personal.json"));
  assert.ok(pedidas.includes("es/preferencias.json"));
  assert.equal(traducir("areaPersonal.rutas.inicio"), (await catalogoPlano("es"))["areaPersonal.rutas.inicio"]);
});

test("la URL explícita prevalece y una preferencia extraña no construye rutas", () => {
  assert.equal(idiomaAreaPersonal(["en-GB"], { href: "https://vec.example/area-personal/?lang=es" }), "es");
  assert.equal(idiomaAreaPersonal(["../../privado"], { href: "https://vec.example/area-personal/?lang=../../privado" }), "es");
  assert.throws(() => urlCatalogo("../../privado", "area-personal"), TypeError);
});

test("el catálogo inglés del área personal no conserva textos en castellano", async () => {
  const [es, en] = await Promise.all([catalogoPlano("es"), catalogoPlano("en")]);
  // Iguales en ambos idiomas por ser nombres propios, siglas o números.
  const invariantes = new Set(["areaPersonal.ficha.tipo.dni", "areaPersonal.ficha.tipo.nie", "areaPersonal.html.logo", "areaPersonal.vista.perfil.avisos.telegram",
    "areaPersonal.preferencias.opcion.tamano_texto.normal"]);
  const iguales = Object.keys(es).filter((clave) => es[clave] === en[clave] && !invariantes.has(clave)
    && /\p{L}{3,}/u.test(es[clave]));
  assert.deepEqual(iguales, []);
});
