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
  assert.deepEqual(leer.pedidas, ["en/area-personal.json"]);
});

test("las preferencias se cargan solo al abrir esa pantalla y en el idioma elegido", async () => {
  const leer = lectorCatalogos();
  const documento = { documentElement: {}, querySelectorAll: () => [] };
  await iniciarI18nAreaPersonal(documento, { leer, pantalla: "preferencias",
    ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias&lang=en" } });
  assert.deepEqual(leer.pedidas, ["en/area-personal.json", "en/preferencias.json"]);
  assert.equal(traducir("areaPersonal.preferencias.campo.idioma"), "Language");
});

test("un fallo transitorio reintenta el idioma elegido antes de acudir al de defecto", async () => {
  const base = lectorCatalogos();
  const pedidas = [];
  let primera = true;
  const leer = async (url) => {
    pedidas.push(url.pathname.split("/textos/").at(-1));
    if (primera) { primera = false; throw new Error("fallo transitorio"); }
    return base(url);
  };
  const documento = { documentElement: {}, querySelectorAll: () => [] };
  const idioma = await iniciarI18nAreaPersonal(documento, { leer,
    ubicacion: { href: "https://vec.example/area-personal/?lang=en" } });
  assert.equal(idioma, "en");
  assert.deepEqual(pedidas, ["en/area-personal.json", "en/area-personal.json"]);
  assert.equal(documento.documentElement.lang, "en");
});

test("si falla el idioma elegido, se reintenta y se abre el catálogo de defecto", async () => {
  const base = lectorCatalogos();
  const pedidas = [];
  const leer = async (url) => {
    const ruta = url.pathname.split("/textos/").at(-1);
    pedidas.push(ruta);
    if (ruta.startsWith("en/")) throw new Error("catálogo no disponible");
    return base(url);
  };
  const documento = { documentElement: {}, querySelectorAll: () => [] };
  const idioma = await iniciarI18nAreaPersonal(documento, { leer,
    ubicacion: { href: "https://vec.example/area-personal/?lang=en" } });
  assert.equal(idioma, "es");
  assert.deepEqual(pedidas, ["en/area-personal.json", "en/area-personal.json", "es/area-personal.json"]);
  assert.equal(documento.documentElement.lang, "es");
  assert.equal(traducir("areaPersonal.rutas.inicio"), "Inicio");
});

test("una carga tardía de Preferencias no cambia el idioma de otra vista", async () => {
  const base = lectorCatalogos();
  const documento = { documentElement: {}, querySelectorAll: () => [] };
  await iniciarI18nAreaPersonal(documento, { leer: base,
    ubicacion: { href: "https://vec.example/area-personal/?lang=en" } });
  let liberar;
  const leerLento = async (url) => url.pathname.endsWith("/preferencias.json")
    ? new Promise((resolve) => { liberar = () => base(url).then(resolve); }) : base(url);
  const pendiente = iniciarI18nAreaPersonal(documento, { leer: leerLento, pantalla: "preferencias",
    ubicacion: { href: "https://vec.example/area-personal/?vista=preferencias&lang=en" } });
  while (!liberar) await new Promise((resolver) => setImmediate(resolver));
  await iniciarI18nAreaPersonal(documento, { leer: base,
    ubicacion: { href: "https://vec.example/area-personal/?vista=inicio&lang=es" } });
  liberar();
  await pendiente;
  assert.equal(documento.documentElement.lang, "es");
  assert.equal(traducir("areaPersonal.rutas.inicio"), "Inicio");
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
    "areaPersonal.vista.seguimiento.provisional.estado",
    "areaPersonal.vista.seguimiento.posicion.ordenDe",
    "areaPersonal.preferencias.opcion.tamano_texto.normal"]);
  const iguales = Object.keys(es).filter((clave) => es[clave] === en[clave] && !invariantes.has(clave)
    && /\p{L}{3,}/u.test(es[clave]));
  assert.deepEqual(iguales, []);
});
