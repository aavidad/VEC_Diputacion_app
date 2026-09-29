import test from "node:test";
import assert from "node:assert/strict";

import {
  crearHerramientasCatalogoCategorias,
  normalizarPaginaCategoriasProfesionales,
  ofertaPerteneceAlCatalogoActual,
  traducirCatalogoCategorias,
} from "./catalogo-categorias.js";

const HUELLA = "b800a7e9c306fa8027709cfb4304cc8ccf8065f888673da71bd73a138c519233";

function herramientas() {
  return crearHerramientasCatalogoCategorias({
    getPersonalCatalog: (vista) => vista.personalCatalog,
    workTableTextCollator: new Intl.Collator("es", { sensitivity: "base" }),
  });
}

test("normaliza la respuesta gobernada sin perder categorías ni metadatos", () => {
  const items = Array.from({ length: 68 }, (_, indice) => ({
    clave: `categoria-${indice + 1}`,
    etiqueta: `Categoría ${indice + 1}`,
  }));
  const pagina = normalizarPaginaCategoriasProfesionales({
    items,
    total: 68,
    catalogo: { catalogo_id: "categorias-profesionales", catalogo_version: 1 },
    fuente: { demostracion: true, aviso: "Contenido de demostración" },
  });

  assert.equal(pagina.items.length, 68);
  assert.equal(pagina.total, 68);
  assert.equal(pagina.catalogo.catalogo_id, "categorias-profesionales");
  assert.equal(pagina.demostracion, true);
  assert.equal(pagina.aviso, "Contenido de demostración");
});

test("el directorio conserva las 68 categorías y sus claves estables", () => {
  const items = Array.from({ length: 68 }, (_, indice) => ({
    slug: `categoria-${indice + 1}`,
    name: `Categoría ${indice + 1}`,
    orden: 68 - indice,
  }));
  const vista = { personalCatalog: { categories: { items } } };
  const categorias = herramientas().obtenerCategoriasProfesionales(vista);

  assert.equal(categorias.length, 68);
  assert.equal(categorias[0].clave, "categoria-68");
  assert.equal(categorias.at(-1).clave, "categoria-1");
});

test("solo reutiliza ofertas vinculadas a la versión y huella vigentes", () => {
  const referencia = {
    id: "categorias-profesionales",
    version: 1,
    huellaSHA256: HUELLA,
  };
  const oferta = {
    categoryKey: "administrativo",
    categoryCatalogID: referencia.id,
    categoryCatalogVersion: referencia.version,
    categoryCatalogSHA256: referencia.huellaSHA256,
  };

  assert.equal(ofertaPerteneceAlCatalogoActual(oferta, ["administrativo"], referencia), true);
  assert.equal(ofertaPerteneceAlCatalogoActual({ ...oferta, categoryCatalogSHA256: "0".repeat(64) }, ["administrativo"], referencia), false);
  assert.equal(ofertaPerteneceAlCatalogoActual({ categoryKey: "administrativo" }, ["administrativo"], referencia), false);
  assert.equal(ofertaPerteneceAlCatalogoActual(oferta, ["otra-categoria"], referencia), false);
});

test("el catálogo de categorías traduce rótulos, estados y plurales", () => {
  assert.equal(traducirCatalogoCategorias("titulo", {}, "es"), "Categorías profesionales");
  assert.equal(traducirCatalogoCategorias("titulo", {}, "en"), "Job categories");
  assert.equal(traducirCatalogoCategorias("estado_retirado", {}, "en"), "Withdrawn");
  assert.equal(traducirCatalogoCategorias("mostrando_una", { mostradas: 1, total: 1 }, "en"), "Showing 1 of 1 category received");
  assert.equal(traducirCatalogoCategorias("ver_detalle_de", { categoria: "<script>" }, "en"), "View details for <script>");
  assert.throws(() => traducirCatalogoCategorias("clave_inexistente", {}, "en"), /Missing category catalogue key/u);
});
